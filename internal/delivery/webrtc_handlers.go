package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
	webrtcManager "nvide-live/internal/webrtc"
)

type WebRTCHandler struct {
	roomManager  *webrtcManager.RoomManager
	streamUseCase *usecase.StreamUseCase
	authUseCase   *usecase.AuthUseCase
	logger        *zap.Logger
}

func NewWebRTCHandler(roomManager *webrtcManager.RoomManager, streamUseCase *usecase.StreamUseCase, authUseCase *usecase.AuthUseCase, logger *zap.Logger) *WebRTCHandler {
	return &WebRTCHandler{
		roomManager:   roomManager,
		streamUseCase: streamUseCase,
		authUseCase:   authUseCase,
		logger:        logger,
	}
}

var webrtcUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

// SignalWS handles WebRTC signaling WebSocket
func (h *WebRTCHandler) SignalWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	// Token validation via query param
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token is required", http.StatusUnauthorized)
		return
	}

	claims, err := h.authUseCase.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Extract viewer/host info
	userIDStr := claims.UserID
	isHost := r.URL.Query().Get("role") == "host"

	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}
	userID, err := domain.FromString(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Upgrade connection
	conn, err := webrtcUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection", zap.Error(err))
		return
	}

	// Setup peer
	sendChan := make(chan webrtcManager.SignalingMessage, 256)
	if isHost {
		_, err = h.roomManager.HandleHostConnection(roomIDStr, userIDStr, sendChan)
	} else {
		// Join stream
		ip := strings.Split(r.RemoteAddr, ":")[0]
		err = h.streamUseCase.JoinStream(r.Context(), roomID, userID, ip)
		if err == nil {
			_, err = h.roomManager.HandleViewerConnection(roomIDStr, userIDStr, sendChan)
		}
	}

	if err != nil {
		h.logger.Error("Failed to setup peer", zap.Error(err))
		conn.Close()
		return
	}

	// Goroutine to send messages to client
	go func() {
		defer conn.Close()
		for msg := range sendChan {
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}()

	// Read loop
	defer func() {
		h.roomManager.RemovePeer(roomIDStr, userIDStr)
		if !isHost {
			h.streamUseCase.LeaveStream(r.Context(), roomID, userID)
		}
		close(sendChan)
		conn.Close()
	}()

	for {
		var msg webrtcManager.SignalingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("Unexpected WS close", zap.Error(err))
			}
			break
		}

		// Process signaling
		if err := h.roomManager.ProcessSignaling(roomIDStr, userIDStr, msg); err != nil {
			h.logger.Error("Failed to process signaling", zap.Error(err))
		}
	}
}

// Write helper
func (h *WebRTCHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WebRTCHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// CreateStreamRequest represents stream creation payload
type CreateStreamRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// CreateStream handles stream creation
func (h *WebRTCHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	// Use helper to get user ID
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	stream, err := h.streamUseCase.CreateStream(r.Context(), userID, req.Title, req.Description, req.ThumbnailURL)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, stream)
}

// StartStream handles starting a stream
func (h *WebRTCHandler) StartStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID, err := domain.FromString(vars["stream_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}

	if err := h.streamUseCase.StartStream(r.Context(), streamID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Stream started"})
}

// EndStream handles ending a stream
func (h *WebRTCHandler) EndStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID, err := domain.FromString(vars["stream_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}

	if err := h.streamUseCase.EndStream(r.Context(), streamID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Stream ended"})
}

// GetLiveStreams lists all live streams
func (h *WebRTCHandler) GetLiveStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.streamUseCase.GetLiveStreams(r.Context(), 10, 0)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, streams)
}
