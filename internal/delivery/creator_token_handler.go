package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// CreatorTokenHandler menangani request HTTP untuk Creator Tokens
type CreatorTokenHandler struct {
	useCase domain.CreatorTokenUseCaseInterface
	logger  *zap.Logger
}

// NewCreatorTokenHandler membuat instance baru dari CreatorTokenHandler
func NewCreatorTokenHandler(useCase domain.CreatorTokenUseCaseInterface, logger *zap.Logger) *CreatorTokenHandler {
	return &CreatorTokenHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *CreatorTokenHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CreatorTokenHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// IssueTokenRequest mewakili payload untuk menerbitkan token baru
type IssueTokenRequest struct {
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	MaxSupply int64  `json:"max_supply"`
	BasePrice int64  `json:"base_price"`
	Slope     int64  `json:"slope"`
}

// IssueToken handles POST /creators/{host_id}/tokens
func (h *CreatorTokenHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	// Dapatkan host_id dari URL
	vars := mux.Vars(r)
	hostIDStr := vars["host_id"]
	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "ID Host tidak valid")
		return
	}

	// Otentikasi: pastikan user yang login adalah host yang bersangkutan
	authUserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || authUserID != hostID {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses untuk menerbitkan token host ini")
		return
	}

	var req IssueTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Body request tidak valid")
		return
	}

	token, err := h.useCase.IssueToken(r.Context(), hostID, req.Name, req.Symbol, req.MaxSupply, req.BasePrice, req.Slope)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, token)
}

// GetTokenInfo handles GET /creators/{host_id}/tokens
func (h *CreatorTokenHandler) GetTokenInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostIDStr := vars["host_id"]
	hostID, err := domain.FromString(hostIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "ID Host tidak valid")
		return
	}

	token, err := h.useCase.GetTokenInfo(r.Context(), hostID)
	if err != nil {
		if err == domain.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Host belum menerbitkan token kustom")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, token)
}

// BuyTokenRequest mewakili payload untuk membeli token
type BuyTokenRequest struct {
	TokenID string `json:"token_id"`
	Amount  int64  `json:"amount"`
}

// BuyToken handles POST /tokens/buy
func (h *CreatorTokenHandler) BuyToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	var req BuyTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Body request tidak valid")
		return
	}

	tokenID, err := domain.FromString(req.TokenID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_TOKEN_ID", "ID Token tidak valid")
		return
	}

	userToken, err := h.useCase.BuyToken(r.Context(), userID, tokenID, req.Amount)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, userToken)
}

// GetUserBalances handles GET /users/{id}/tokens
func (h *CreatorTokenHandler) GetUserBalances(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := domain.FromString(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "ID User tidak valid")
		return
	}

	// Keamanan tambahan: user hanya boleh melihat saldo mereka sendiri
	authUserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || authUserID != userID {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak diizinkan mengakses informasi saldo pengguna lain")
		return
	}

	balances, err := h.useCase.GetUserBalances(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, balances)
}
