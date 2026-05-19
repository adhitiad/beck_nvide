package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

type DashboardHandler struct {
	useCase domain.DashboardUseCase
	logger  *zap.Logger
}

// NewDashboardHandler membuat instance baru dari DashboardHandler
func NewDashboardHandler(useCase domain.DashboardUseCase, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *DashboardHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *DashboardHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func (h *DashboardHandler) writeErrorLocalized(ctx context.Context, w http.ResponseWriter, status int, code, translationKey string, defaultMessage string) {
	msg := middleware.T(ctx, translationKey)
	if msg == translationKey {
		msg = defaultMessage
	}
	h.writeError(w, status, code, msg)
}

// === ADMIN DASHBOARD ENDPOINTS ===

func (h *DashboardHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.useCase.GetAdminStats(r.Context())
	if err != nil {
		h.logger.Error("GetAdminStats: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil statistik admin")
		return
	}
	h.writeJSON(w, http.StatusOK, stats)
}

func (h *DashboardHandler) GetAdminRevenue(w http.ResponseWriter, r *http.Request) {
	rev, err := h.useCase.GetAdminRevenue(r.Context())
	if err != nil {
		h.logger.Error("GetAdminRevenue: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil rincian pendapatan")
		return
	}
	h.writeJSON(w, http.StatusOK, rev)
}

func (h *DashboardHandler) GetAdminGraph(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "daily"
	}

	graph, err := h.useCase.GetAdminGraph(r.Context(), period)
	if err != nil {
		h.logger.Error("GetAdminGraph: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data grafik")
		return
	}
	h.writeJSON(w, http.StatusOK, graph)
}

func (h *DashboardHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	filter := domain.UserListFilter{
		Role:   role,
		Status: status,
		Search: search,
		Page:   page,
		Limit:  limit,
	}

	users, total, err := h.useCase.ListUsers(r.Context(), filter)
	if err != nil {
		h.logger.Error("ListUsers: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil list user")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *DashboardHandler) ListPendingKYC(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	pending, err := h.useCase.ListPendingKYC(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("ListPendingKYC: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil list KYC pending")
		return
	}

	h.writeJSON(w, http.StatusOK, pending)
}

func (h *DashboardHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	reports, err := h.useCase.ListReports(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("ListReports: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil list laporan")
		return
	}

	h.writeJSON(w, http.StatusOK, reports)
}

func (h *DashboardHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "Admin tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID target tidak valid")
		return
	}

	var req struct {
		Reason      string `json:"reason"`
		IsPermanent bool   `json:"is_permanent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Alasan pemblokiran harus diisi")
		return
	}

	err = h.useCase.BanUser(r.Context(), adminID, targetID, req.Reason, req.IsPermanent)
	if err != nil {
		h.logger.Error("BanUser: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal memblokir user")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "User berhasil diblokir",
	})
}

func (h *DashboardHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID target tidak valid")
		return
	}

	err = h.useCase.UnbanUser(r.Context(), targetID)
	if err != nil {
		h.logger.Error("UnbanUser: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengaktifkan kembali user")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "User berhasil diaktifkan kembali",
	})
}

func (h *DashboardHandler) TerminateStream(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Admin tidak terautentikasi")
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
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Alasan penghentian harus diisi")
		return
	}

	err = h.useCase.TerminateStream(r.Context(), adminID, streamID, req.Reason)
	if err != nil {
		h.logger.Error("TerminateStream: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Siaran live berhasil dihentikan",
	})
}

func (h *DashboardHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Admin tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	commentIDStr := vars["id"]
	commentID, err := domain.FromString(commentIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID komentar tidak valid")
		return
	}

	err = h.useCase.DeleteComment(r.Context(), adminID, commentID)
	if err != nil {
		h.logger.Error("DeleteComment: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal menghapus komentar")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Komentar berhasil dihapus",
	})
}

func (h *DashboardHandler) SubmitReport(w http.ResponseWriter, r *http.Request) {
	reporterID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "User tidak terautentikasi")
		return
	}

	var req struct {
		ReportedUserID *domain.UUID `json:"reported_user_id,omitempty"`
		StreamID       *domain.UUID `json:"stream_id,omitempty"`
		ChatMessageID  *domain.UUID `json:"chat_message_id,omitempty"`
		ReportType     string       `json:"report_type"`
		Reason         string       `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "INVALID_REQUEST", "invalid_request", "Format request JSON tidak valid")
		return
	}

	if req.ReportType == "" || req.Reason == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Tipe laporan dan alasan harus diisi")
		return
	}

	report, err := h.useCase.SubmitReport(r.Context(), reporterID, req.ReportedUserID, req.StreamID, req.ChatMessageID, req.ReportType, req.Reason)
	if err != nil {
		h.logger.Error("SubmitReport: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal menyimpan laporan")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Laporan berhasil dikirim",
		"report":  report,
	})
}

// === HOST DASHBOARD ENDPOINTS ===

func (h *DashboardHandler) GetHostStats(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	stats, err := h.useCase.GetHostStats(r.Context(), hostID)
	if err != nil {
		h.logger.Error("GetHostStats: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil statistik host")
		return
	}
	h.writeJSON(w, http.StatusOK, stats)
}

func (h *DashboardHandler) GetHostRevenue(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "daily"
	}

	rev, err := h.useCase.GetHostRevenue(r.Context(), hostID, period)
	if err != nil {
		h.logger.Error("GetHostRevenue: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil pendapatan host")
		return
	}
	h.writeJSON(w, http.StatusOK, rev)
}

func (h *DashboardHandler) GetHostClips(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	clips, err := h.useCase.GetHostClips(r.Context(), hostID, limit, offset)
	if err != nil {
		h.logger.Error("GetHostClips: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil klip host")
		return
	}
	h.writeJSON(w, http.StatusOK, clips)
}

func (h *DashboardHandler) GetHostStreams(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	streams, err := h.useCase.GetHostStreams(r.Context(), hostID, limit, offset)
	if err != nil {
		h.logger.Error("GetHostStreams: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil siaran host")
		return
	}
	h.writeJSON(w, http.StatusOK, streams)
}

func (h *DashboardHandler) GetHostRequests(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	page := 1
	limit := 10
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	requests, err := h.useCase.GetHostRequests(r.Context(), hostID, limit, offset)
	if err != nil {
		h.logger.Error("GetHostRequests: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data permintaan show")
		return
	}
	h.writeJSON(w, http.StatusOK, requests)
}

func (h *DashboardHandler) UpdateHostSettings(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Host tidak terautentikasi")
		return
	}

	var req domain.HostDashboardSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Format JSON input tidak valid")
		return
	}

	err := h.useCase.UpdateHostSettings(r.Context(), hostID, &req)
	if err != nil {
		h.logger.Error("UpdateHostSettings: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal memperbarui pengaturan host")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Pengaturan host berhasil diperbarui",
	})
}

// === AGENCY DASHBOARD ENDPOINTS ===

func (h *DashboardHandler) GetAgencyStats(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	stats, err := h.useCase.GetAgencyStats(r.Context(), ownerID)
	if err != nil {
		h.logger.Error("GetAgencyStats: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, stats)
}

func (h *DashboardHandler) GetAgencyHosts(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	hosts, err := h.useCase.GetAgencyHosts(r.Context(), ownerID)
	if err != nil {
		h.logger.Error("GetAgencyHosts: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data host agensi")
		return
	}
	h.writeJSON(w, http.StatusOK, hosts)
}

func (h *DashboardHandler) InviteHost(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	var req struct {
		HostUsername string `json:"host_username"`
		RevenueShare int    `json:"revenue_share"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostUsername == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Format input tidak valid atau username kosong")
		return
	}

	if req.RevenueShare < 0 || req.RevenueShare > 100 {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Revenue share harus bernilai antara 0 sampai 100")
		return
	}

	err := h.useCase.InviteHostToAgency(r.Context(), ownerID, req.HostUsername, req.RevenueShare)
	if err != nil {
		h.logger.Error("InviteHost: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Undangan bergabung agensi berhasil dikirim ke host",
	})
}

func (h *DashboardHandler) RemoveHost(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	hostIDStr := vars["id"]
	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID host tidak valid")
		return
	}

	err = h.useCase.RemoveHostFromAgency(r.Context(), ownerID, hostID)
	if err != nil {
		h.logger.Error("RemoveHost: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Host berhasil dihapus dari agensi",
	})
}

func (h *DashboardHandler) GetAgencyRevenue(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "daily"
	}

	rev, err := h.useCase.GetAgencyRevenue(r.Context(), ownerID, period)
	if err != nil {
		h.logger.Error("GetAgencyRevenue: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data rincian pendapatan agensi")
		return
	}
	h.writeJSON(w, http.StatusOK, rev)
}

func (h *DashboardHandler) UpdateAgencySettings(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Pemilik agensi tidak terautentikasi")
		return
	}

	var req domain.AgencyDashboardSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Format input JSON tidak valid")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Nama agensi harus diisi")
		return
	}

	err := h.useCase.UpdateAgencySettings(r.Context(), ownerID, &req)
	if err != nil {
		h.logger.Error("UpdateAgencySettings: handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Pengaturan agensi berhasil diperbarui",
	})
}
