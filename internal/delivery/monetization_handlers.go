package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
	"nvide-live/pkg/duitku"
)

type MonetizationHandler struct {
	walletUC  *usecase.WalletUseCase
	giftUC    *usecase.GiftUseCase
	agencyUC  *usecase.AgencyUseCase
	paymentUC *usecase.PaymentUseCase
	withdrawalUC domain.WithdrawalUsecase
	logger    *zap.Logger
}

func NewMonetizationHandler(
	walletUC *usecase.WalletUseCase,
	giftUC *usecase.GiftUseCase,
	agencyUC *usecase.AgencyUseCase,
	paymentUC *usecase.PaymentUseCase,
	withdrawalUC domain.WithdrawalUsecase,
	logger *zap.Logger,
) *MonetizationHandler {
	return &MonetizationHandler{
		walletUC:  walletUC,
		giftUC:    giftUC,
		agencyUC:  agencyUC,
		paymentUC: paymentUC,
		withdrawalUC: withdrawalUC,
		logger:    logger,
	}
}

func (h *MonetizationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *MonetizationHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{"error_code": code, "message": message})
}

func (h *MonetizationHandler) getUserID(r *http.Request) (domain.UUID, bool) {
	return middleware.GetUserIDFromContext(r.Context())
}

// ===== WALLET =====

func (h *MonetizationHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	wallet, err := h.walletUC.GetBalance(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, wallet)
}

func (h *MonetizationHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	txType := r.URL.Query().Get("type")
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	txs, err := h.walletUC.GetTransactionHistory(r.Context(), userID, txType, limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, txs)
}

// ===== GIFTS =====

func (h *MonetizationHandler) GetGiftCatalog(w http.ResponseWriter, r *http.Request) {
	gifts, err := h.giftUC.GetGiftCatalog(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, gifts)
}

type SendGiftRequest struct {
	ReceiverID     string `json:"receiver_id"`
	RoomID         string `json:"room_id"`         // Untuk Stream Chat
	ConversationID string `json:"conversation_id"` // Untuk Private Chat
	GiftID         string `json:"gift_id"`
	Quantity       int    `json:"quantity"`
}

func (h *MonetizationHandler) SendGift(w http.ResponseWriter, r *http.Request) {
	senderID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req SendGiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	receiverID, err := domain.FromString(req.ReceiverID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_RECEIVER", "Invalid receiver ID")
		return
	}
	giftID, err := domain.FromString(req.GiftID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_GIFT", "Invalid gift ID")
		return
	}

	// Case 1: Private Chat Gift
	if req.ConversationID != "" {
		convID, err := domain.FromString(req.ConversationID)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION", "Invalid conversation ID")
			return
		}
		gtx, err := h.giftUC.SendPrivateGift(r.Context(), senderID, convID, giftID, req.Quantity)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "GIFT_ERROR", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, gtx)
		return
	}

	// Case 2: Live Stream Gift (Room Chat)
	var roomID *domain.UUID
	if req.RoomID != "" {
		rid, err := domain.FromString(req.RoomID)
		if err == nil {
			roomID = &rid
		}
	}

	gtx, err := h.giftUC.SendGift(r.Context(), senderID, receiverID, roomID, giftID, req.Quantity)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "GIFT_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, gtx)
}

func (h *MonetizationHandler) GetGiftLeaderboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID, err := domain.FromString(vars["stream_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}
	lb, err := h.giftUC.GetLeaderboard(r.Context(), streamID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, lb)
}

// ===== HOST & AGENCY =====

type ApplyHostRequest struct {
	Bio               string `json:"bio"`
	IDCardURL         string `json:"id_card_url"`
	BankName          string `json:"bank_name"`
	BankAccountName   string `json:"bank_account_name"`
	BankAccountNumber string `json:"bank_account_number"`
}

func (h *MonetizationHandler) ApplyHost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req ApplyHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	app, err := h.agencyUC.ApplyHost(r.Context(), userID, req.Bio, req.IDCardURL, req.BankName, req.BankAccountName, req.BankAccountNumber)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "APPLICATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, app)
}

func (h *MonetizationHandler) ApproveHostApplication(w http.ResponseWriter, r *http.Request) {
	reviewerID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	vars := mux.Vars(r)
	appID, err := domain.FromString(vars["application_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid application ID")
		return
	}
	if err := h.agencyUC.ApproveHost(r.Context(), appID, reviewerID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Application approved"})
}

type RejectHostRequest struct {
	Reason string `json:"reason"`
}

func (h *MonetizationHandler) RejectHostApplication(w http.ResponseWriter, r *http.Request) {
	reviewerID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	vars := mux.Vars(r)
	appID, err := domain.FromString(vars["application_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid application ID")
		return
	}
	var req RejectHostRequest
	json.NewDecoder(r.Body).Decode(&req)
	if err := h.agencyUC.RejectHost(r.Context(), appID, reviewerID, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Application rejected"})
}

func (h *MonetizationHandler) ListHostApplications(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = domain.ApplicationPending
	}
	apps, err := h.agencyUC.ListHostApplications(r.Context(), status, 20, 0)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, apps)
}

type CreateAgencyRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	LogoURL        string `json:"logo_url"`
	CommissionRate int    `json:"commission_rate"`
}

func (h *MonetizationHandler) CreateAgency(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req CreateAgencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	agency, err := h.agencyUC.CreateAgency(r.Context(), ownerID, req.Name, req.Description, req.LogoURL, req.CommissionRate)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "AGENCY_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, agency)
}

type InviteHostRequest struct {
	HostID       string `json:"host_id"`
	RevenueShare int    `json:"revenue_share"`
}

func (h *MonetizationHandler) InviteHost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agencyID, err := domain.FromString(vars["agency_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_AGENCY_ID", "Invalid agency ID")
		return
	}
	var req InviteHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	hostID, err := domain.FromString(req.HostID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_HOST_ID", "Invalid host ID")
		return
	}
	if err := h.agencyUC.InviteHost(r.Context(), agencyID, hostID, req.RevenueShare); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVITE_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation sent"})
}

func (h *MonetizationHandler) ListAgencyHosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agencyID, err := domain.FromString(vars["agency_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_AGENCY_ID", "Invalid agency ID")
		return
	}
	hosts, err := h.agencyUC.ListAgencyHosts(r.Context(), agencyID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, hosts)
}

// ===== PAYMENT =====

type DepositRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	CustomerName  string `json:"customer_name"`
}

func (h *MonetizationHandler) RequestDeposit(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	dp, err := h.paymentUC.RequestDeposit(r.Context(), userID, req.Amount, req.PaymentMethod, req.Email, req.CustomerName)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "DEPOSIT_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, dp)
}

func (h *MonetizationHandler) DuitkuCallback(w http.ResponseWriter, r *http.Request) {
	var payload duitku.CallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CALLBACK", "Invalid callback payload")
		return
	}
	if err := h.paymentUC.HandleCallback(r.Context(), &payload); err != nil {
		h.logger.Error("Duitku callback processing error", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "CALLBACK_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type WithdrawalRequest struct {
	Amount int64 `json:"amount"`
}

func (h *MonetizationHandler) RequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	tx, err := h.paymentUC.RequestWithdrawal(r.Context(), userID, req.Amount)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "WITHDRAWAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, tx)
}

func (h *MonetizationHandler) ApproveWithdrawal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txID, err := domain.FromString(vars["tx_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_TX_ID", "Invalid transaction ID")
		return
	}
	if err := h.paymentUC.ApproveWithdrawal(r.Context(), txID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Withdrawal approved"})
}

func (h *MonetizationHandler) RejectWithdrawal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txID, err := domain.FromString(vars["tx_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_TX_ID", "Invalid transaction ID")
		return
	}
	if err := h.paymentUC.RejectWithdrawal(r.Context(), txID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Withdrawal rejected"})
}

// ===== NEW WITHDRAWAL FEE ENGINE =====

func (h *MonetizationHandler) GetWithdrawalPreview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	amountStr := r.URL.Query().Get("amount")
	amount, _ := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 {
		h.writeError(w, http.StatusBadRequest, "INVALID_AMOUNT", "Amount must be greater than zero")
		return
	}
	preview, err := h.withdrawalUC.CalculatePreview(r.Context(), userID, amount)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, preview)
}

func (h *MonetizationHandler) SubmitWithdrawal(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req struct {
		Amount        int64                  `json:"amount"`
		PaymentMethod string                 `json:"payment_method"`
		BankInfo      map[string]interface{} `json:"bank_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	wd, err := h.withdrawalUC.RequestWithdrawal(r.Context(), userID, req.Amount, req.PaymentMethod, req.BankInfo)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "WITHDRAWAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, wd)
}

