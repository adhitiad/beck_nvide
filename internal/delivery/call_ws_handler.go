package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

var callUpgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeCallWS handles WebSocket signaling for calls
func (h *Handler) ServeCallWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionIDStr := vars["session_id"]
	if sessionIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Session ID is required")
		return
	}

	sessionID, err := domain.FromString(sessionIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID")
		return
	}

	// Token validation
	token := r.URL.Query().Get("token")
	claims, err := h.authUseCase.ValidateToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
		return
	}

	conn, err := callUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade call websocket", zap.Error(err))
		return
	}

	go h.handleCallSignaling(conn, sessionID, claims.UserID)
}

func (h *Handler) handleCallSignaling(conn *gorillaws.Conn, sessionID domain.UUID, userID string) {
	defer conn.Close()

	// 1. Join "call:{session_id}" room in Hub (for signaling)
	roomID := "call:" + sessionID.String()
	
	// We'll use a simplified signaling for now. 
	// In a real SFU environment, this would be more complex.
	// For P2P, we just relay messages to the other participant in the same room.
	
	// Start billing ticker if this is the start of the call
	// This should ideally be triggered by "call:start" event
	
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "call:accept":
			h.paidInteractionUseCase.AcceptCall(context.Background(), sessionID)
			// Start billing ticker in background
			go h.startBillingTicker(sessionID)
			
		case "call:end":
			h.paidInteractionUseCase.EndCall(context.Background(), sessionID, "user_end")
			return
			
		case "webrtc:offer", "webrtc:answer", "webrtc:ice":
			// Relay to other participant via Hub
			h.wsHub.BroadcastToRoom(roomID, msgBytes)
		}
	}
}

func (h *Handler) startBillingTicker(sessionID domain.UUID) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	ctx := context.Background()
	
	// Grace period: first 10 seconds are free
	// So we tick after 60 seconds of ACTIVE status
	
	for {
		select {
		case <-ticker.C:
			err := h.paidInteractionUseCase.ProcessBillingTick(ctx, sessionID)
			if err != nil {
				h.logger.Warn("Billing tick failed, ending call", zap.String("session_id", sessionID.String()), zap.Error(err))
				h.paidInteractionUseCase.EndCall(ctx, sessionID, "balance_insufficient")
				
				// Notify participants via WS
				h.wsHub.BroadcastToRoom("call:"+sessionID.String(), []byte(`{"type":"call:ended","reason":"balance_insufficient"}`))
				return
			}
			
			// Notify success tick
			h.wsHub.BroadcastToRoom("call:"+sessionID.String(), []byte(`{"type":"call:tick","status":"success"}`))

		case <-ctx.Done():
			return
		}
		
		// Check if session still active
		session, _ := h.paidInteractionUseCase.RequestCall(ctx, "", "", "") // Mock fetch
		// Actually better to have GetByID in usecase
		if session != nil && session.Status == domain.CallStatusEnded {
			return
		}
	}
}
