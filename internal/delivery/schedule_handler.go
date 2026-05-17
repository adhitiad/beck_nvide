package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
)

func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req domain.LiveSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Format body request tidak valid")
		return
	}

	err := h.liveScheduleUseCase.CreateSchedule(r.Context(), userID, &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, req)
}

func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	scheduleIDStr := vars["id"]
	scheduleID, err := domain.FromString(scheduleIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Schedule ID tidak valid")
		return
	}

	var req domain.LiveSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Format body request tidak valid")
		return
	}

	err = h.liveScheduleUseCase.UpdateSchedule(r.Context(), userID, scheduleID, &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Jadwal siaran berhasil diperbarui"})
}

func (h *Handler) CancelSchedule(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	scheduleIDStr := vars["id"]
	scheduleID, err := domain.FromString(scheduleIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Schedule ID tidak valid")
		return
	}

	err = h.liveScheduleUseCase.CancelSchedule(r.Context(), userID, scheduleID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Jadwal siaran berhasil dibatalkan"})
}

func (h *Handler) CancelOccurrence(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	occIDStr := vars["occ_id"]
	occID, err := domain.FromString(occIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Occurrence ID tidak valid")
		return
	}

	err = h.liveScheduleUseCase.CancelOccurrence(r.Context(), userID, occID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Sesi siaran terjadwal tunggal berhasil dibatalkan"})
}

func (h *Handler) SubscribeReminder(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	scheduleIDStr := vars["id"]
	scheduleID, err := domain.FromString(scheduleIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Schedule ID tidak valid")
		return
	}

	var req domain.UserScheduleReminder
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Remind24h = true
		req.Remind1h = true
		req.Remind15m = true
		req.RemindLiveStart = true
		req.PushEnabled = true
	}

	err = h.liveScheduleUseCase.SubscribeReminder(r.Context(), userID, scheduleID, &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "SUBSCRIBE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Berhasil mengaktifkan pengingat untuk jadwal siaran ini"})
}

func (h *Handler) UnsubscribeReminder(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	scheduleIDStr := vars["id"]
	scheduleID, err := domain.FromString(scheduleIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Schedule ID tidak valid")
		return
	}

	err = h.liveScheduleUseCase.UnsubscribeReminder(r.Context(), userID, scheduleID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "UNSUBSCRIBE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Berhasil menonaktifkan pengingat untuk jadwal siaran ini"})
}

func (h *Handler) ListMyReminders(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	list, err := h.liveScheduleUseCase.ListMyReminders(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	type itemDTO struct {
		*domain.LiveScheduleOccurrence
		CountdownSeconds int64 `json:"countdown_seconds"`
	}

	result := make([]itemDTO, len(list))
	for i, occ := range list {
		remaining := int64(time.Until(occ.OccurrenceStartAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		result[i] = itemDTO{
			LiveScheduleOccurrence: occ,
			CountdownSeconds:       remaining,
		}
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetNextSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostIDStr := vars["id"]
	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Host ID tidak valid")
		return
	}

	occ, err := h.liveScheduleUseCase.GetNextSchedule(r.Context(), hostID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	if occ == nil {
		h.writeJSON(w, http.StatusOK, map[string]interface{}{"next_occurrence": nil})
		return
	}

	remaining := int64(time.Until(occ.OccurrenceStartAt).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"next_occurrence":   occ,
		"countdown_seconds": remaining,
	})
}

func (h *Handler) GetUpcomingFeed(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	query := r.URL.Query()
	category := query.Get("category")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit := 10
	offset := 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	list, err := h.liveScheduleUseCase.GetUpcomingFeed(r.Context(), userID, category, limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, list)
}

func (h *Handler) GetTrendingSchedules(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	list, err := h.liveScheduleUseCase.GetTrendingSchedules(r.Context(), limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, list)
}

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	list, err := h.liveScheduleUseCase.GetAnalytics(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, list)
}

func (h *Handler) PledgeGift(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	waitRoomIDStr := vars["id"]
	waitRoomID, err := domain.FromString(waitRoomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Wait Room ID tidak valid")
		return
	}

	type pledgeReq struct {
		GiftCode string `json:"gift_code"`
		Quantity int    `json:"quantity"`
	}

	var req pledgeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GiftCode == "" || req.Quantity <= 0 {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Gift code dan quantity valid wajib diisi")
		return
	}

	err = h.liveScheduleUseCase.SaveWaitRoomPledge(r.Context(), waitRoomID, userID, req.GiftCode, req.Quantity)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "PLEDGE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Komitmen Gift (Pledge) berhasil dikirim!"})
}

func (h *Handler) ServeWaitRoomWS(w http.ResponseWriter, r *http.Request) {
	if h.waitRoomHub != nil {
		h.waitRoomHub.ServeWaitRoomWS(w, r)
	} else {
		http.Error(w, "Wait Room WebSocket Hub is not initialized", http.StatusServiceUnavailable)
	}
}
