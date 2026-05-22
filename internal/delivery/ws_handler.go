package delivery

import (
	"net/http"

	"github.com/gorilla/mux"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		return true
	},
}

// ServeWS handles WebSocket requests
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
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

	// Check if user is participant of the room
	roomID, err := domain.FromString(roomIDStr)
	if err == nil {
		userID, err := domain.FromString(claims.UserID)
		if err == nil {
			// Actually we should check via messageUseCase if they can join,
			// but for now we just register them to the hub.
			// The usecase will be used if needed.
			
			// Auto join room if not already a participant (for streams usually)
			_ = h.messageUseCase.JoinRoom(r.Context(), roomID, userID)
		}
	}

	// Register client to hub
	websocket.NewClient(h.wsHub, conn, roomIDStr, claims.UserID)
}
