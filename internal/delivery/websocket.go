package delivery

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
)

// ServeStreamChatWS handles WebSocket requests for live stream chat with comments saving and private chat fallback
func (h *Handler) ServeStreamChatWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	
	// Support both {stream_id}, {conversation_id}, and general {id}
	idStr := vars["stream_id"]
	if idStr == "" {
		idStr = vars["conversation_id"]
	}
	if idStr == "" {
		idStr = vars["id"]
	}
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Identifier is required")
		return
	}

	// Token validation via query param
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

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection", zap.Error(err))
		return
	}

	// Dynamic routing check: Is this a Live Stream room?
	if h.wsHub.IsStreamRoom(idStr) {
		h.logger.Info("Smart routing: WebSocket chat connected as LIVE STREAM", zap.String("id", idStr))
		
		// Auto join stream room if not already a participant
		streamID, err := domain.FromString(idStr)
		if err == nil {
			userID, err := domain.FromString(claims.UserID)
			if err == nil {
				_ = h.messageUseCase.JoinRoom(r.Context(), streamID, userID)
			}
		}

		// Register client to hub using the stream ID directly as the room identifier
		websocket.NewClient(h.wsHub, conn, idStr, claims.UserID)
	} else {
		h.logger.Info("Smart routing: WebSocket chat connected as PRIVATE CHAT (DM)", zap.String("id", idStr))
		
		// Register client to hub with "chat:" prefix for private room isolation
		roomID := "chat:" + idStr
		websocket.NewClient(h.wsHub, conn, roomID, claims.UserID)
	}
}
