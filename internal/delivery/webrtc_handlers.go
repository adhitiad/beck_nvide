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
	roomManager     *webrtcManager.RoomManager
	streamUseCase   *usecase.StreamUseCase
	authUseCase     *usecase.AuthUseCase
	trendingUseCase *usecase.TrendingUseCase
	scheduleUseCase domain.LiveScheduleUseCase
	logger          *zap.Logger
}

func NewWebRTCHandler(
	roomManager *webrtcManager.RoomManager,
	streamUseCase *usecase.StreamUseCase,
	authUseCase *usecase.AuthUseCase,
	trendingUseCase *usecase.TrendingUseCase,
	scheduleUseCase domain.LiveScheduleUseCase,
	logger *zap.Logger,
) *WebRTCHandler {
	return &WebRTCHandler{
		roomManager:     roomManager,
		streamUseCase:   streamUseCase,
		authUseCase:     authUseCase,
		trendingUseCase: trendingUseCase,
		scheduleUseCase: scheduleUseCase,
		logger:          logger,
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

	// Setup unique peer ID based on user ID and role to avoid conflicts in local same-browser testing
	roleStr := "viewer"
	if isHost {
		roleStr = "host"
	}
	peerIDStr := userIDStr + "_" + roleStr

	// Setup peer
	sendChan := make(chan webrtcManager.SignalingMessage, 256)
	if isHost {
		_, err = h.roomManager.HandleHostConnection(roomIDStr, peerIDStr, sendChan)
	} else {
		// Join stream
		ip := strings.Split(r.RemoteAddr, ":")[0]
		password := r.URL.Query().Get("password")
		err = h.streamUseCase.JoinStream(r.Context(), roomID, userID, ip, password)
		if err == nil {
			_, err = h.roomManager.HandleViewerConnection(roomIDStr, peerIDStr, sendChan)
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
		h.roomManager.RemovePeer(roomIDStr, peerIDStr)
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
		if err := h.roomManager.ProcessSignaling(roomIDStr, peerIDStr, msg); err != nil {
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
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	ThumbnailURL        string   `json:"thumbnail_url"`
	RoomMode            string   `json:"room_mode"`
	RoomPassword        string   `json:"room_password"`
	EntryFeeIDR         float64  `json:"entry_fee_idr"`
	MinLevelToEnter     int      `json:"min_level_to_enter"`
	Category            string   `json:"category"`
	Tags                string   `json:"tags"`
	MaxResolution       string   `json:"max_resolution"`
	IsScreenShare       bool     `json:"is_screen_share"`
	IsCoHostEnabled     bool     `json:"is_co_host_enabled"`
	MaxCoHosts          int      `json:"max_co_hosts"`
	ChatMode            string   `json:"chat_mode"`
	ChatSlowModeSeconds int      `json:"chat_slow_mode_seconds"`
	CountryCode         string   `json:"country_code"`
	Language            string   `json:"language"`
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

	input := domain.CreateStreamInput{
		Title:               req.Title,
		Description:         req.Description,
		ThumbnailURL:        req.ThumbnailURL,
		RoomMode:            req.RoomMode,
		RoomPassword:        req.RoomPassword,
		EntryFeeIDR:         req.EntryFeeIDR,
		MinLevelToEnter:     req.MinLevelToEnter,
		Category:            req.Category,
		Tags:                req.Tags,
		MaxResolution:       req.MaxResolution,
		IsScreenShare:       req.IsScreenShare,
		IsCoHostEnabled:     req.IsCoHostEnabled,
		MaxCoHosts:          req.MaxCoHosts,
		ChatMode:            req.ChatMode,
		ChatSlowModeSeconds: req.ChatSlowModeSeconds,
		CountryCode:         req.CountryCode,
		Language:            req.Language,
	}

	stream, err := h.streamUseCase.CreateStream(r.Context(), userID, input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, stream)
}

// SwitchRoomModeRequest represents request payload to change stream room mode
type SwitchRoomModeRequest struct {
	RoomMode     string  `json:"room_mode"`
	RoomPassword string  `json:"room_password"`
	EntryFeeIDR  float64 `json:"entry_fee_idr"`
}

// SwitchRoomMode handles switching room mode
func (h *WebRTCHandler) SwitchRoomMode(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	streamID, err := domain.FromString(vars["stream_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}

	var req SwitchRoomModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.streamUseCase.SwitchRoomMode(r.Context(), userID, streamID, req.RoomMode, req.RoomPassword, req.EntryFeeIDR); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Room mode updated successfully"})
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

	stream, err := h.streamUseCase.GetStreamByID(r.Context(), streamID)
	if err != nil {
		h.writeJSON(w, http.StatusOK, map[string]string{"message": "Stream started"})
		return
	}

	// Automatically check and link schedule occurrence (Fitur 7)
	if h.scheduleUseCase != nil {
		_, _ = h.scheduleUseCase.CheckAndAutoLinkStream(r.Context(), stream.HostID, stream.ID)
	}

	h.writeJSON(w, http.StatusOK, stream)
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

// GetStream handles fetching a single stream by ID
func (h *WebRTCHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID, err := domain.FromString(vars["stream_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}

	stream, err := h.streamUseCase.GetStreamByID(r.Context(), streamID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "STREAM_NOT_FOUND", "Stream not found")
		return
	}

	h.writeJSON(w, http.StatusOK, stream)
}

// GetTrendingStreams lists all live streams ordered by trending score (Fase 6)
func (h *WebRTCHandler) GetTrendingStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.trendingUseCase.GetTrendingStreams(r.Context(), 10)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, streams)
}
