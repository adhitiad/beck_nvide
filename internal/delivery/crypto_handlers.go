package delivery

import (
	"encoding/json"
	"net/http"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
	"nvide-live/internal/worker"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type CryptoHandler struct {
	cryptoUC *usecase.CryptoUseCase
	monitor  *worker.CryptoMonitor
	logger   *zap.Logger
}

func NewCryptoHandler(cryptoUC *usecase.CryptoUseCase, monitor *worker.CryptoMonitor, logger *zap.Logger) *CryptoHandler {
	return &CryptoHandler{cryptoUC: cryptoUC, monitor: monitor, logger: logger}
}

func (h *CryptoHandler) GetDepositAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id_uuid").(domain.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chain := r.URL.Query().Get("chain")
	if chain == "" {
		h.writeError(w, http.StatusBadRequest, "chain is required")
		return
	}

	addr, err := h.cryptoUC.GetOrCreateDepositAddress(r.Context(), userID, chain)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, addr)
}

func (h *CryptoHandler) GetExchangeRates(w http.ResponseWriter, r *http.Request) {
	assets := []string{"SOL", "BTC", "USDT"}
	rates := make(map[string]float64)

	for _, asset := range assets {
		rate, _ := h.cryptoUC.GetExchangeRate(r.Context(), asset)
		rates[asset] = rate
	}

	h.writeJSON(w, http.StatusOK, rates)
}

func (h *CryptoHandler) RequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id_uuid").(domain.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Chain     string  `json:"chain"`
		Asset     string  `json:"asset"`
		Address   string  `json:"address"`
		Amount    float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tx, err := h.cryptoUC.RequestWithdrawal(r.Context(), userID, req.Chain, req.Asset, req.Address, req.Amount)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusAccepted, tx)
}

func (h *CryptoHandler) CryptoWebhook(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"] // e.g. "helius", "blockcypher"
	
	var payload interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if err := h.monitor.HandleWebhook(r.Context(), provider, payload); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *CryptoHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CryptoHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
