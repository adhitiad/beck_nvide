package delivery

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"nvide-live/internal/domain"
)

// OB Handlers (Host Offers)
func (h *Handler) CreateHostOffer(w http.ResponseWriter, r *http.Request) {
	var req domain.HostOffer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	hostID := h.getUserID(r)
	offer, err := h.offerUseCase.CreateOB(r.Context(), hostID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, offer)
}

func (h *Handler) BookHostOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerID, _ := domain.FromString(vars["id"])
	
	var req struct {
		SlotStart time.Time `json:"slot_start"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	userID := h.getUserID(r)
	booking, err := h.offerUseCase.BookOB(r.Context(), userID, offerID, req.SlotStart)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, booking)
}

// BO Handlers (User Offers)
func (h *Handler) CreateUserOffer(w http.ResponseWriter, r *http.Request) {
	var req domain.UserOffer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	offer, err := h.offerUseCase.CreateBO(r.Context(), userID, req.HostID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, offer)
}

func (h *Handler) RespondToUserOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerID, _ := domain.FromString(vars["id"])
	
	var req struct {
		Action       string   `json:"action"` // accept, reject, counter
		Message      string   `json:"message"`
		CounterPrice *float64 `json:"counter_price"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	hostID := h.getUserID(r)
	err := h.offerUseCase.RespondToBO(r.Context(), hostID, offerID, req.Action, req.Message, req.CounterPrice)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Response sent successfully"})
}

func (h *Handler) GetOfferFeed(w http.ResponseWriter, r *http.Request) {
	offers, err := h.offerUseCase.GetOfferFeed(r.Context(), nil)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, offers)
}
