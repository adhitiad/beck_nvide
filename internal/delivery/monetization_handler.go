package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// ExtraMonetizationHandler handles extra monetization features API requests
type ExtraMonetizationHandler struct {
	useCase domain.MonetizationUseCase
	logger  *zap.Logger
}

// NewExtraMonetizationHandler creates a new instance of ExtraMonetizationHandler
func NewExtraMonetizationHandler(
	useCase domain.MonetizationUseCase,
	logger *zap.Logger,
) *ExtraMonetizationHandler {
	return &ExtraMonetizationHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *ExtraMonetizationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ExtraMonetizationHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// CreatePaidRoom handles POST /rooms (Host creates paid room)
func (h *ExtraMonetizationHandler) CreatePaidRoom(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	var req struct {
		Name        string `json:"name"`
		EntryFeeIDR int64  `json:"entry_fee_idr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	room, err := h.useCase.CreatePaidRoom(r.Context(), hostID, req.Name, req.EntryFeeIDR)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, room)
}

// JoinPaidRoom handles POST /rooms/{id}/join (User pays and joins paid room)
func (h *ExtraMonetizationHandler) JoinPaidRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	roomIDStr := vars["id"]
	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID room tidak valid")
		return
	}

	room, err := h.useCase.JoinPaidRoom(r.Context(), userID, roomID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "TRANSACTION_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Berhasil membayar dan bergabung ke private room berbayar",
		"room":    room,
	})
}

// RegisterHostDevice handles POST /hosts/me/devices (Host registers interactive toys device)
func (h *ExtraMonetizationHandler) RegisterHostDevice(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	var req struct {
		DeviceName string `json:"device_name"`
		DeviceID   string `json:"device_id"`
		APIToken   string `json:"api_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	device, err := h.useCase.RegisterHostDevice(r.Context(), hostID, req.DeviceName, req.DeviceID, req.APIToken)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, device)
}

// GetHostDevices handles GET /hosts/me/devices (Host retrieves their devices)
func (h *ExtraMonetizationHandler) GetHostDevices(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	devices, err := h.useCase.GetHostDevices(r.Context(), hostID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil daftar perangkat")
		return
	}

	h.writeJSON(w, http.StatusOK, devices)
}

// ControlToys handles POST /streams/{id}/toys/control (Viewer tips + controls host's smart toys)
func (h *ExtraMonetizationHandler) ControlToys(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	streamIDStr := vars["id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID stream tidak valid")
		return
	}

	var req struct {
		Command         string `json:"command"` // e.g. "Vibrate:2", "Vibrate:5"
		DurationSeconds int    `json:"duration_seconds"`
		TipsAmount      int64  `json:"tips_amount"` // Optional tips associated with the control action
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	if req.Command == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_COMMAND", "Perintah kontrol tidak boleh kosong")
		return
	}
	if req.DurationSeconds <= 0 {
		req.DurationSeconds = 5 // default 5 seconds
	}

	result, err := h.useCase.ControlToys(r.Context(), userID, streamID, req.Command, req.DurationSeconds, req.TipsAmount)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "CONTROL_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": result,
	})
}

// SubmitShowRequest handles POST /streams/{id}/requests (Viewer submits custom show request with tips)
func (h *ExtraMonetizationHandler) SubmitShowRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	streamIDStr := vars["id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID stream tidak valid")
		return
	}

	var req struct {
		Description string `json:"description"`
		TipsAmount  int64  `json:"tips_amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	showReq, err := h.useCase.SubmitShowRequest(r.Context(), userID, streamID, req.Description, req.TipsAmount)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "SUBMIT_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, showReq)
}

// AcceptShowRequest handles PUT /requests/{id}/accept (Host accepts custom show request)
func (h *ExtraMonetizationHandler) AcceptShowRequest(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	reqIDStr := vars["id"]
	reqID, err := domain.FromString(reqIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID request tidak valid")
		return
	}

	err = h.useCase.AcceptShowRequest(r.Context(), hostID, reqID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "ACCEPT_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Request show berhasil diterima dan tips dicairkan ke wallet Anda",
	})
}

// RejectShowRequest handles PUT /requests/{id}/reject (Host rejects custom show request)
func (h *ExtraMonetizationHandler) RejectShowRequest(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	reqIDStr := vars["id"]
	reqID, err := domain.FromString(reqIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID request tidak valid")
		return
	}

	err = h.useCase.RejectShowRequest(r.Context(), hostID, reqID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Request show berhasil ditolak",
	})
}

// SendAIChatMessage handles POST /ai/chat (Viewer chats with creator's offline AI chatbot)
func (h *ExtraMonetizationHandler) SendAIChatMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	var req struct {
		HostID  string `json:"host_id"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	hostID, err := domain.FromString(req.HostID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "ID host tidak valid")
		return
	}

	message, err := h.useCase.SendAIChatMessage(r.Context(), userID, hostID, req.Content)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "CHAT_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, message)
}

// GetAIChatHistory handles GET /ai/chat/history (Viewer retrieves companion chat history)
func (h *ExtraMonetizationHandler) GetAIChatHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	hostIDStr := r.URL.Query().Get("host_id")
	if hostIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_HOST_ID", "Parameter host_id diperlukan")
		return
	}

	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "ID host tidak valid")
		return
	}

	limit := 50
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.useCase.GetAIChatHistory(r.Context(), userID, hostID, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil riwayat chat AI companion")
		return
	}

	h.writeJSON(w, http.StatusOK, history)
}
