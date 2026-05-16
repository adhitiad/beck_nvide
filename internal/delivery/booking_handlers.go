package delivery

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"nvide-live/internal/domain"
)

// Booking handlers
func (h *Handler) SetHostSchedule(w http.ResponseWriter, r *http.Request) {
	var req []domain.HostSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	schedules := make([]*domain.HostSchedule, len(req))
	for i := range req {
		req[i].ID = domain.NewUUIDv7()
		req[i].HostID = userID
		schedules[i] = &req[i]
	}

	err := h.bookingUseCase.SetSchedule(r.Context(), schedules)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Schedule updated successfully"})
}

func (h *Handler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID, _ := domain.FromString(vars["id"])
	dateStr := r.URL.Query().Get("date")
	date, _ := time.Parse("2006-01-02", dateStr)

	slots, err := h.bookingUseCase.GetAvailableSlots(r.Context(), hostID, date)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, slots)
}

func (h *Handler) RequestBooking(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HostID      string    `json:"host_id"`
		TypeID      string    `json:"booking_type_id"`
		ScheduledAt time.Time `json:"scheduled_at"`
		Duration    int       `json:"duration"`
		Notes       string    `json:"notes"`
		Latitude    *float64  `json:"latitude"`
		Longitude   *float64  `json:"longitude"`
		LocationName string   `json:"location_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	hID, _ := domain.FromString(req.HostID)
	tID, _ := domain.FromString(req.TypeID)

	booking, err := h.bookingUseCase.RequestBooking(r.Context(), userID, hID, tID, req.ScheduledAt, req.Duration, req.Notes, req.Latitude, req.Longitude, req.LocationName)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, booking)
}

func (h *Handler) AcceptBooking(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookingID, _ := domain.FromString(vars["id"])
	hostID := h.getUserID(r)

	err := h.bookingUseCase.AcceptBooking(r.Context(), hostID, bookingID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Booking accepted"})
}

func (h *Handler) RejectBooking(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookingID, _ := domain.FromString(vars["id"])
	hostID := h.getUserID(r)
	
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	err := h.bookingUseCase.RejectBooking(r.Context(), hostID, bookingID, req.Reason)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Booking rejected"})
}
