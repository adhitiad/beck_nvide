package delivery

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

type PushHandler struct {
	uc     domain.PushNotificationUsecase
	logger *zap.Logger
}

func NewPushHandler(uc domain.PushNotificationUsecase, logger *zap.Logger) *PushHandler {
	return &PushHandler{uc: uc, logger: logger}
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	Topics []string `json:"topics"`
}

type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *PushHandler) getUserID(w http.ResponseWriter, r *http.Request) (domain.UUID, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "UNAUTHORIZED", "message": "User not authenticated"})
		return "", false
	}
	return userID, true
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	var req pushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "INVALID_REQUEST"})
		return
	}
	if err := h.uc.Subscribe(r.Context(), userID, req.Endpoint, req.Keys.P256DH, req.Keys.Auth, req.Topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	var req pushUnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "INVALID_REQUEST"})
		return
	}
	if err := h.uc.Unsubscribe(r.Context(), userID, req.Endpoint); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *PushHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	sent, err := h.uc.SendTest(r.Context(), userID, req.Title, req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "sent": sent})
}
