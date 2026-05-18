package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// ServeHostWS handles host WS signaling connection with grace period and heartbeat check
func (h *WebRTCHandler) ServeHostWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Room ID is required")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token is required")
		return
	}

	claims, err := h.authUseCase.ValidateToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
		return
	}

	userIDStr := claims.UserID
	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid room ID")
		return
	}
	userID, err := domain.FromString(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid user ID")
		return
	}

	// Security: Verify that the user connecting as host actually owns this stream/room
	stream, err := h.streamUseCase.GetStreamByID(r.Context(), roomID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Stream not found")
		return
	}
	if stream.HostID != userID {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not the host of this stream")
		return
	}

	// Upgrade connection
	conn, err := webrtcUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection for host", zap.Error(err))
		return
	}

	peerIDStr := userIDStr + "_host"
	sendChan := make(chan webrtcManager.SignalingMessage, 100)

	// Track host connection in Redis to manage reconnect grace period
	hostConnectedKey := fmt.Sprintf("stream:host_connected:%s", roomIDStr)
	redisClient := h.streamUseCase.GetRedisClient()
	redisClient.GetClient().Set(r.Context(), hostConnectedKey, "true", 24*time.Hour)

	// Set up PeerConnection
	_, err = h.roomManager.HandleHostConnection(roomIDStr, peerIDStr, sendChan)
	if err != nil {
		h.logger.Error("Failed to handle host connection", zap.Error(err))
		conn.Close()
		return
	}

	// Start reading signaling messages from sendChan and writing to WS
	go func() {
		for msg := range sendChan {
			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Error("Failed to write to host WS", zap.Error(err))
				break
			}
		}
	}()

	// Heartbeat setup: Ping every 10 seconds, missing 3 consecutive pings = disconnect
	// We set a read deadline of 35 seconds (30 seconds ping timeout + 5 seconds buffer)
	_ = conn.SetReadDeadline(time.Now().Add(35 * time.Second))
	conn.SetPingHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(35 * time.Second))
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	// Defer cleanup and grace period scheduling
	defer func() {
		h.logger.Info("Host connection closed, cleaning up and initiating grace period", zap.String("room_id", roomIDStr))
		h.roomManager.RemovePeer(roomIDStr, peerIDStr)
		close(sendChan)
		_ = conn.Close()

		// Set host connected state to false
		redisClient.GetClient().Set(context.Background(), hostConnectedKey, "false", 1*time.Hour)

		// Start 30-second host disconnect grace period timer
		go func() {
			time.Sleep(30 * time.Second)
			// Check if host reconnected (redis key becomes "true")
			val, err := redisClient.GetClient().Get(context.Background(), hostConnectedKey).Result()
			if err != nil || val != "true" {
				h.logger.Warn("Host failed to reconnect within 30 seconds grace period. Ending stream.", zap.String("room_id", roomIDStr))
				_ = h.streamUseCase.EndStream(context.Background(), roomID)
				h.roomManager.RemoveRoom(roomIDStr)
			} else {
				h.logger.Info("Host successfully reconnected within grace period", zap.String("room_id", roomIDStr))
			}
		}()
	}()

	// Read loop
	for {
		var msg webrtcManager.SignalingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("Unexpected host WS close", zap.Error(err))
			}
			break
		}

		// Support explicit application-level ping message if client wants to send ping via text
		if msg.Type == "ping" {
			_ = conn.SetReadDeadline(time.Now().Add(35 * time.Second))
			_ = conn.WriteJSON(webrtcManager.SignalingMessage{Type: "pong"})
			continue
		}

		// Process signaling
		if err := h.roomManager.ProcessSignaling(roomIDStr, peerIDStr, msg); err != nil {
			h.logger.Error("Failed to process host signaling", zap.Error(err))
		}
	}
}

// ServeViewerWS handles viewer WS signaling connection with active viewer count tracking in Redis
func (h *WebRTCHandler) ServeViewerWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Room ID is required")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token is required")
		return
	}

	claims, err := h.authUseCase.ValidateToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
		return
	}

	userIDStr := claims.UserID
	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid room ID")
		return
	}
	userID, err := domain.FromString(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid user ID")
		return
	}

	// Upgrade connection
	conn, err := webrtcUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection for viewer", zap.Error(err))
		return
	}

	peerIDStr := userIDStr + "_viewer"
	sendChan := make(chan webrtcManager.SignalingMessage, 100)

	// Set up PeerConnection
	_, err = h.roomManager.HandleViewerConnection(roomIDStr, peerIDStr, sendChan)
	if err != nil {
		h.logger.Error("Failed to handle viewer connection", zap.Error(err))
		conn.Close()
		return
	}

	// Track Viewer Count Accuracy - Atomic INCR in Redis on Join
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	password := r.URL.Query().Get("password")

	err = h.streamUseCase.JoinStream(r.Context(), roomID, userID, ip, password)
	if err != nil {
		h.logger.Error("Viewer failed to join stream", zap.Error(err))
		conn.WriteJSON(webrtcManager.SignalingMessage{
			Type: "error",
			Data: []byte(err.Error()),
		})
		conn.Close()
		return
	}

	// Start reading signaling messages from sendChan and writing to WS
	go func() {
		for msg := range sendChan {
			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Error("Failed to write to viewer WS", zap.Error(err))
				break
			}
		}
	}()

	// Defer cleanup and decrement count in Redis
	defer func() {
		h.logger.Info("Viewer connection closed, cleaning up", zap.String("room_id", roomIDStr), zap.String("user_id", userIDStr))
		h.roomManager.RemovePeer(roomIDStr, peerIDStr)
		close(sendChan)
		_ = conn.Close()

		// Track Viewer Count Accuracy - Atomic DECR in Redis on Leave
		_ = h.streamUseCase.LeaveStream(context.Background(), roomID, userID)
	}()

	// Read loop
	for {
		var msg webrtcManager.SignalingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("Unexpected viewer WS close", zap.Error(err))
			}
			break
		}

		// Process signaling
		if err := h.roomManager.ProcessSignaling(roomIDStr, peerIDStr, msg); err != nil {
			h.logger.Error("Failed to process viewer signaling", zap.Error(err))
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
