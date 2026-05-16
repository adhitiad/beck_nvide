package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"nvide-live/internal/domain"
)

func (h *Handler) UpdateMyLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid location data")
		return
	}

	userID := h.getUserID(r)
	err := h.locationUseCase.UpdateUserLocation(r.Context(), userID, req.Latitude, req.Longitude)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Location updated"})
}

func (h *Handler) GetHostLiveLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID, _ := domain.FromString(vars["host_id"])
	
	// Check if user has active booking with this host
	// TODO: Add security check

	loc, err := h.locationUseCase.GetUserLocation(r.Context(), hostID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "LOCATION_NOT_FOUND", "Host is not currently sharing location")
		return
	}

	h.writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) GetBookingMeetingPoint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookingID, _ := domain.FromString(vars["id"])

	loc, err := h.locationUseCase.GetMeetingPoint(r.Context(), bookingID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, loc)
}
