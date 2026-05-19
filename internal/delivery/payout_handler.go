package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

type PayoutHandler struct {
	uc     domain.PayoutUsecase
	logger *zap.Logger
}

func NewPayoutHandler(uc domain.PayoutUsecase, logger *zap.Logger) *PayoutHandler {
	return &PayoutHandler{uc: uc, logger: logger}
}

func (h *PayoutHandler) getUserID(w http.ResponseWriter, r *http.Request) (domain.UUID, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"UNAUTHORIZED","message":"User not authenticated"}`, http.StatusUnauthorized)
		return "", false
	}
	return userID, true
}

func (h *PayoutHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *PayoutHandler) writeErr(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{"error_code": code, "message": message})
}

func (h *PayoutHandler) ListPayoutMethods(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	data, err := h.uc.ListPayoutMethods(r.Context(), userID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, data)
}

func (h *PayoutHandler) CreatePayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	var req domain.CreatePayoutMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	data, err := h.uc.CreatePayoutMethod(r.Context(), userID, &req)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, data)
}

func (h *PayoutHandler) UpdatePayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	methodID, err := domain.FromString(mux.Vars(r)["id"])
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid payout method id")
		return
	}
	var req domain.UpdatePayoutMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	data, err := h.uc.UpdatePayoutMethod(r.Context(), userID, methodID, &req)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, data)
}

func (h *PayoutHandler) DeletePayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	methodID, err := domain.FromString(mux.Vars(r)["id"])
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid payout method id")
		return
	}
	if err := h.uc.DeletePayoutMethod(r.Context(), userID, methodID); err != nil {
		h.writeErr(w, http.StatusBadRequest, "DELETE_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Payout method deleted"})
}

func (h *PayoutHandler) SetPrimaryPayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	methodID, err := domain.FromString(mux.Vars(r)["id"])
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid payout method id")
		return
	}
	if err := h.uc.SetPrimaryPayoutMethod(r.Context(), userID, methodID); err != nil {
		h.writeErr(w, http.StatusBadRequest, "UPDATE_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Primary payout method updated"})
}

func (h *PayoutHandler) ListCryptoPayoutAddresses(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	data, err := h.uc.ListCryptoPayoutAddresses(r.Context(), userID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, data)
}

func (h *PayoutHandler) CreateCryptoPayoutAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	var req domain.CreateCryptoPayoutAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	data, err := h.uc.CreateCryptoPayoutAddress(r.Context(), userID, &req)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, data)
}

func (h *PayoutHandler) DeleteCryptoPayoutAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	id, err := domain.FromString(mux.Vars(r)["id"])
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid crypto payout id")
		return
	}
	if err := h.uc.DeleteCryptoPayoutAddress(r.Context(), userID, id); err != nil {
		h.writeErr(w, http.StatusBadRequest, "DELETE_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Crypto payout address deleted"})
}
