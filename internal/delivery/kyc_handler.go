package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

type KYCHandler struct {
	kycUseCase      *usecase.KYCUseCase
	onboardUseCase  *usecase.OnboardingUseCase
	logger          *zap.Logger
}

func NewKYCHandler(
	kycUseCase *usecase.KYCUseCase,
	onboardUseCase *usecase.OnboardingUseCase,
	logger *zap.Logger,
) *KYCHandler {
	return &KYCHandler{
		kycUseCase:     kycUseCase,
		onboardUseCase: onboardUseCase,
		logger:         logger,
	}
}

func (h *KYCHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *KYCHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func (h *KYCHandler) writeErrorLocalized(ctx context.Context, w http.ResponseWriter, status int, code, translationKey string, defaultMessage string) {
	msg := middleware.T(ctx, translationKey)
	if msg == translationKey {
		msg = defaultMessage
	}
	h.writeError(w, status, code, msg)
}

// SubmitKYC submits new KYC verification for hosts/individuals
func (h *KYCHandler) SubmitKYC(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "User not authenticated")
		return
	}

	var req struct {
		IDCardNumber string `json:"id_card_number"`
		FullName     string `json:"full_name"`
		Gender       string `json:"gender"`
		Country      string `json:"country"`
		DocumentURL  string `json:"document_url"`
		SelfieURL    string `json:"selfie_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "INVALID_REQUEST", "invalid_request", "Failed to parse JSON body")
		return
	}

	// Basic fields validation
	if req.IDCardNumber == "" || req.FullName == "" || req.Gender == "" || req.Country == "" || req.DocumentURL == "" || req.SelfieURL == "" {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "VALIDATION_ERROR", "validation_error", "All KYC fields are strictly required")
		return
	}

	// Extract device fingerprint & IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	kyc, err := h.kycUseCase.SubmitKYC(
		r.Context(),
		userID,
		req.IDCardNumber,
		req.FullName,
		req.Gender,
		req.Country,
		req.DocumentURL,
		req.SelfieURL,
		fingerprint,
		ip,
	)

	if err != nil {
		if err == usecase.ErrInvalidRegion {
			h.writeErrorLocalized(r.Context(), w, http.StatusForbidden, "REGION_RESTRICTED", "region_restricted", err.Error())
			return
		}
		if err == usecase.ErrLGBTDetected {
			h.writeErrorLocalized(r.Context(), w, http.StatusForbidden, "PERMANENT_BANNED", "lgbt_policy_ban", "Your account has been permanently blocked due to a policy violation.")
			return
		}
		h.logger.Error("KYC submit handler failed", zap.Error(err))
		h.writeErrorLocalized(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal_error", "Failed to submit KYC")
		return
	}

	message := middleware.T(r.Context(), "success_kyc_submit")
	if message == "success_kyc_submit" {
		message = "KYC submitted successfully and is currently under review"
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": message,
		"kyc":     kyc,
	})
}

// SubmitAgency submits new business agency verification
func (h *KYCHandler) SubmitAgency(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "User not authenticated")
		return
	}

	var req struct {
		CompanyName        string `json:"company_name"`
		RegistrationNumber string `json:"registration_number"`
		TaxNumber          string `json:"tax_number"`
		PhoneNumber        string `json:"phone_number"`
		OfficeAddress      string `json:"office_address"`
		DocumentURL        string `json:"document_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "INVALID_REQUEST", "invalid_request", "Failed to parse JSON body")
		return
	}

	if req.CompanyName == "" || req.RegistrationNumber == "" || req.TaxNumber == "" || req.PhoneNumber == "" || req.OfficeAddress == "" || req.DocumentURL == "" {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "VALIDATION_ERROR", "validation_error", "All business agency verification fields are required")
		return
	}

	agency, err := h.kycUseCase.SubmitAgencyVerification(
		r.Context(),
		userID,
		req.CompanyName,
		req.RegistrationNumber,
		req.TaxNumber,
		req.PhoneNumber,
		req.OfficeAddress,
		req.DocumentURL,
	)

	if err != nil {
		h.logger.Error("Agency KYC submit handler failed", zap.Error(err))
		h.writeErrorLocalized(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal_error", "Failed to submit business verification")
		return
	}

	message := middleware.T(r.Context(), "success_agency_submit")
	if message == "success_agency_submit" {
		message = "Agency business verification submitted successfully"
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": message,
		"agency":  agency,
	})
}

// GetStatus returns the current KYC & Agency verification status of the user
func (h *KYCHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	status, err := h.kycUseCase.GetKYCStatus(r.Context(), userID)
	if err != nil {
		h.logger.Error("KYC get status handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get verification status")
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

// ApproveKYC approves a pending KYC verification
func (h *KYCHandler) ApproveKYC(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Admin not authenticated")
		return
	}

	vars := mux.Vars(r)
	kycIDStr := vars["id"]
	kycID, err := domain.FromString(kycIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid KYC ID format")
		return
	}

	err = h.kycUseCase.VerifyKYC(r.Context(), kycID, adminID)
	if err != nil {
		if err == domain.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "KYC record not found")
			return
		}
		h.logger.Error("Approve KYC handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to approve KYC")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "KYC verification approved successfully",
	})
}

// RejectKYC rejects a pending KYC verification
func (h *KYCHandler) RejectKYC(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Admin not authenticated")
		return
	}

	vars := mux.Vars(r)
	kycIDStr := vars["id"]
	kycID, err := domain.FromString(kycIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid KYC ID format")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Rejection reason is required")
		return
	}

	err = h.kycUseCase.RejectKYC(r.Context(), kycID, req.Reason, adminID)
	if err != nil {
		if err == domain.ErrNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "KYC record not found")
			return
		}
		h.logger.Error("Reject KYC handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reject KYC")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "KYC verification rejected successfully",
	})
}

// BanUserPermanent locks user permanently
func (h *KYCHandler) BanUserPermanent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := domain.FromString(userIDStr)
	if err != nil {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "INVALID_ID", "invalid_id", "Invalid user ID format")
		return
	}

	var req struct {
		Reason      string `json:"reason"`
		Fingerprint string `json:"device_fingerprint"`
		IP          string `json:"ip_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "VALIDATION_ERROR", "validation_error", "Ban reason is required")
		return
	}

	err = h.kycUseCase.TriggerPermanentBan(r.Context(), userID, req.Reason, req.Fingerprint, req.IP)
	if err != nil {
		h.logger.Error("BanUserPermanent handler failed", zap.Error(err))
		h.writeErrorLocalized(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal_error", "Failed to ban user permanently")
		return
	}

	message := middleware.T(r.Context(), "success_ban")
	if message == "success_ban" {
		message = "User and associated fingerprints have been permanently banned successfully"
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": message,
	})
}

// GetOnboardingChecklist returns role-specific onboarding checklist
func (h *KYCHandler) GetOnboardingChecklist(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "User not authenticated")
		return
	}

	// Read role type from path (e.g. /onboarding/user, /onboarding/host, /onboarding/agency)
	parts := strings.Split(r.URL.Path, "/")
	roleType := "user"
	if len(parts) > 0 {
		roleType = parts[len(parts)-1]
	}

	if roleType != "user" && roleType != "host" && roleType != "agency" {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "INVALID_ROLE", "validation_error", "Invalid onboarding role type requested")
		return
	}

	checklist, err := h.onboardUseCase.GetChecklist(r.Context(), userID, roleType)
	if err != nil {
		h.logger.Error("GetOnboardingChecklist handler failed", zap.Error(err))
		h.writeErrorLocalized(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal_error", "Failed to get onboarding checklist")
		return
	}

	h.writeJSON(w, http.StatusOK, checklist)
}

// CompleteOnboardingStep manually marks an onboarding checklist step completed
func (h *KYCHandler) CompleteOnboardingStep(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeErrorLocalized(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", "User not authenticated")
		return
	}

	var req struct {
		StepKey string `json:"step_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.StepKey == "" {
		h.writeErrorLocalized(r.Context(), w, http.StatusBadRequest, "VALIDATION_ERROR", "validation_error", "step_key is required")
		return
	}

	err := h.onboardUseCase.CompleteStep(r.Context(), userID, req.StepKey)
	if err != nil {
		h.logger.Error("CompleteOnboardingStep handler failed", zap.Error(err))
		h.writeErrorLocalized(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal_error", "Failed to complete step")
		return
	}

	message := middleware.T(r.Context(), "success_onboarding_step")
	if message == "success_onboarding_step" {
		message = "Step marked as completed successfully"
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": message,
	})
}
