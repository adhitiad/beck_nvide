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

type ModerationHandler struct {
	moderationUC domain.ModerationUseCase
	logger       *zap.Logger
}

func NewModerationHandler(moderationUC domain.ModerationUseCase, logger *zap.Logger) *ModerationHandler {
	return &ModerationHandler{
		moderationUC: moderationUC,
		logger:       logger,
	}
}

func RegisterModerationRoutes(r *mux.Router, h *ModerationHandler) {
	// Rules CRUD
	r.HandleFunc("/moderation/rules", h.CreateRule).Methods("POST")
	r.HandleFunc("/moderation/rules", h.ListRules).Methods("GET")
	r.HandleFunc("/moderation/rules/{id}", h.UpdateRule).Methods("PUT")

	// Wordlist CRUD
	r.HandleFunc("/moderation/wordlist", h.AddWord).Methods("POST")
	r.HandleFunc("/moderation/wordlist", h.GetWordlist).Methods("GET")
	r.HandleFunc("/moderation/wordlist", h.DeleteWord).Methods("DELETE")

	// Appeals
	r.HandleFunc("/moderation/appeals", h.SubmitAppeal).Methods("POST")

	// Admin Override & Logs
	r.HandleFunc("/moderation/override", h.ManualOverride).Methods("POST")
	r.HandleFunc("/moderation/logs", h.ListLogs).Methods("GET")
	r.HandleFunc("/moderation/active-bans", h.GetActiveBans).Methods("GET")

	// Image Moderation review panel
	r.HandleFunc("/moderation/images/pending", h.GetPendingImages).Methods("GET")
	r.HandleFunc("/moderation/images/{id}/approve", h.ApproveImage).Methods("POST")
	r.HandleFunc("/moderation/images/{id}/reject", h.RejectImage).Methods("POST")
}

func (h *ModerationHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req domain.ModerationRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	err := h.moderationUC.CreateRule(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, req)
}

func (h *ModerationHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.moderationUC.ListRules(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, rules)
}

func (h *ModerationHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Missing rule id path parameter")
		return
	}

	var req domain.ModerationRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	req.ID = domain.UUID(idStr)
	err := h.moderationUC.UpdateRule(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, req)
}

func (h *ModerationHandler) AddWord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word          string `json:"word"`
		SeverityLevel int    `json:"severity_level"`
		Language      string `json:"language"`
		IsRegex       bool   `json:"is_regex"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	if req.Word == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_WORD", "Word parameter is required")
		return
	}

	err := h.moderationUC.AddWord(r.Context(), req.Word, req.SeverityLevel, req.Language, req.IsRegex)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{"status": "success", "word": req.Word})
}

func (h *ModerationHandler) GetWordlist(w http.ResponseWriter, r *http.Request) {
	list, err := h.moderationUC.GetWordlist(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, list)
}

func (h *ModerationHandler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word string `json:"word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	err := h.moderationUC.DeleteWord(r.Context(), req.Word)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ModerationHandler) SubmitAppeal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LogID  string `json:"log_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	err := h.moderationUC.SubmitAppeal(r.Context(), domain.UUID(req.LogID), req.Reason)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "APPEAL_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "appeal_submitted"})
}

func (h *ModerationHandler) ManualOverride(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing admin auth context")
		return
	}

	var req struct {
		UserID     string `json:"user_id"`
		ActionType string `json:"action_type"` // unmute, unban, mute, ban_perm
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
		return
	}

	err := h.moderationUC.ManualOverride(r.Context(), adminID, domain.UUID(req.UserID), req.ActionType, req.Reason)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "override_applied"})
}

func (h *ModerationHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	var userUID *domain.UUID
	if uStr := q.Get("user_id"); uStr != "" {
		tmp := domain.UUID(uStr)
		userUID = &tmp
	}

	var streamUID *domain.UUID
	if sStr := q.Get("stream_id"); sStr != "" {
		tmp := domain.UUID(sStr)
		streamUID = &tmp
	}

	var action *string
	if act := q.Get("action"); act != "" {
		action = &act
	}

	logs, err := h.moderationUC.ListLogs(r.Context(), userUID, streamUID, action, limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, logs)
}

func (h *ModerationHandler) GetActiveBans(w http.ResponseWriter, r *http.Request) {
	bans, err := h.moderationUC.GetActiveBans(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, bans)
}

func (h *ModerationHandler) GetPendingImages(w http.ResponseWriter, r *http.Request) {
	pending, err := h.moderationUC.GetPendingImages(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, pending)
}

func (h *ModerationHandler) ApproveImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Missing job id path parameter")
		return
	}

	err := h.moderationUC.ApproveImage(r.Context(), domain.UUID(idStr))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *ModerationHandler) RejectImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Missing job id path parameter")
		return
	}

	err := h.moderationUC.RejectImage(r.Context(), domain.UUID(idStr))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// Helpers
func (h *ModerationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *ModerationHandler) writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": msg,
	})
}
