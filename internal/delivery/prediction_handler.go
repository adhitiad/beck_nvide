package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// PredictionHandler menangani request HTTP untuk Prediction Market
type PredictionHandler struct {
	useCase domain.PredictionUseCaseInterface
	logger  *zap.Logger
}

// NewPredictionHandler membuat instance baru dari PredictionHandler
func NewPredictionHandler(useCase domain.PredictionUseCaseInterface, logger *zap.Logger) *PredictionHandler {
	return &PredictionHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *PredictionHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *PredictionHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// CreatePredictionRequest mewakili payload untuk membuat prediksi baru
type CreatePredictionRequest struct {
	Question string `json:"question"`
}

// CreatePrediction handles POST /streams/{id}/predictions
func (h *PredictionHandler) CreatePrediction(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	streamIDStr := vars["id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "ID Stream tidak valid")
		return
	}

	var req CreatePredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Body request tidak valid")
		return
	}

	prediction, err := h.useCase.CreatePrediction(r.Context(), hostID, streamID, req.Question)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, prediction)
}

// GetActivePredictions handles GET /streams/{id}/predictions
func (h *PredictionHandler) GetActivePredictions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamIDStr := vars["id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "ID Stream tidak valid")
		return
	}

	predictions, err := h.useCase.GetActivePredictions(r.Context(), streamID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, predictions)
}

// PlaceBetRequest mewakili payload untuk melakukan taruhan
type PlaceBetRequest struct {
	Outcome        string  `json:"outcome"`          // 'yes', 'no'
	Amount         int64   `json:"amount"`           // jumlah taruhan
	CurrencyType   string  `json:"currency_type"`    // 'wallet', 'token'
	CreatorTokenID *string `json:"creator_token_id"` // opsional jika taruhan memakai token kreator
}

// PlaceBet handles POST /predictions/{id}/bet
func (h *PredictionHandler) PlaceBet(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	predictionIDStr := vars["id"]
	predictionID, err := domain.FromString(predictionIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PREDICTION_ID", "ID Prediksi tidak valid")
		return
	}

	var req PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Body request tidak valid")
		return
	}

	var creatorTokenID *domain.UUID
	if req.CreatorTokenID != nil && *req.CreatorTokenID != "" {
		tid, err := domain.FromString(*req.CreatorTokenID)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_TOKEN_ID", "ID Token Kreator tidak valid")
			return
		}
		creatorTokenID = &tid
	}

	bet, err := h.useCase.PlaceBet(r.Context(), userID, predictionID, req.Outcome, req.Amount, req.CurrencyType, creatorTokenID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, bet)
}

// ResolvePredictionRequest mewakili payload untuk menyelesaikan prediksi
type ResolvePredictionRequest struct {
	Outcome string `json:"outcome"` // 'yes', 'no'
}

// ResolvePrediction handles PUT /predictions/{id}/resolve
func (h *PredictionHandler) ResolvePrediction(w http.ResponseWriter, r *http.Request) {
	hostID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	predictionIDStr := vars["id"]
	predictionID, err := domain.FromString(predictionIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PREDICTION_ID", "ID Prediksi tidak valid")
		return
	}

	var req ResolvePredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Body request tidak valid")
		return
	}

	prediction, err := h.useCase.ResolvePrediction(r.Context(), hostID, predictionID, req.Outcome)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, prediction)
}
