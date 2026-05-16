package delivery

import (
	"encoding/json"
	"net/http"

	"nvide-live/internal/domain"

	"github.com/gorilla/mux"
)

// PaidInteraction handlers
func (h *Handler) SetCallRates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VoiceRate int64 `json:"voice_call_rate_idr"`
		VideoRate int64 `json:"video_call_rate_idr"`
		IsEnabled bool  `json:"is_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// TODO: Check if user has role host or agency

	err := h.paidInteractionUseCase.SetHostRates(r.Context(), userID, req.VoiceRate, req.VideoRate, req.IsEnabled)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Call rates updated successfully"})
}

func (h *Handler) GetHostCallRates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostIDStr := vars["id"]
	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "Invalid host ID")
		return
	}

	rates, err := h.paidInteractionUseCase.GetHostRates(r.Context(), hostID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if rates == nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Host has not set call rates")
		return
	}

	h.writeJSON(w, http.StatusOK, rates)
}

func (h *Handler) UnlockChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.paidInteractionUseCase.UnlockChat(r.Context(), userID, convID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Chat unlocked successfully"})
}

func (h *Handler) RequestCall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HostID string `json:"host_id"`
		Type   string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	hostID, err := domain.FromString(req.HostID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "Invalid host ID")
		return
	}

	session, err := h.paidInteractionUseCase.RequestCall(r.Context(), userID, hostID, req.Type)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) AcceptCall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID")
		return
	}

	err = h.paidInteractionUseCase.AcceptCall(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Call accepted"})
}

func (h *Handler) EndCall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID")
		return
	}

	err = h.paidInteractionUseCase.EndCall(r.Context(), id, "user_end")
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Call ended"})
}
