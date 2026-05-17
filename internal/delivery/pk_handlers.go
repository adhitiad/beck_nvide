package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

type PKBattleHandler struct {
	pkUseCase *usecase.PKBattleUseCase
	logger    *zap.Logger
}

func NewPKBattleHandler(pkUseCase *usecase.PKBattleUseCase, logger *zap.Logger) *PKBattleHandler {
	return &PKBattleHandler{
		pkUseCase: pkUseCase,
		logger:    logger,
	}
}

type InvitePKRequest struct {
	TargetHostID string `json:"target_host_id"`
}

func (h *PKBattleHandler) InvitePK(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req InvitePKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	targetHostID, err := domain.FromString(req.TargetHostID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "Invalid target host ID")
		return
	}

	pk, err := h.pkUseCase.InvitePKBattle(r.Context(), userID, targetHostID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, pk)
}

func (h *PKBattleHandler) AcceptPK(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	pkID, err := domain.FromString(vars["pk_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PK_ID", "Invalid PK ID")
		return
	}

	pk, err := h.pkUseCase.AcceptPKBattle(r.Context(), userID, pkID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, pk)
}

func (h *PKBattleHandler) RejectPK(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	pkID, err := domain.FromString(vars["pk_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PK_ID", "Invalid PK ID")
		return
	}

	err = h.pkUseCase.RejectPKBattle(r.Context(), userID, pkID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "PK Battle invitation rejected successfully"})
}

func (h *PKBattleHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pkID, err := domain.FromString(vars["pk_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PK_ID", "Invalid PK ID")
		return
	}

	status, err := h.pkUseCase.GetPKStatus(r.Context(), pkID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

func (h *PKBattleHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *PKBattleHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}
