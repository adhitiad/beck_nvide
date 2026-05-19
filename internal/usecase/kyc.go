package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/config"
)

var (
	ErrInvalidRegion = errors.New("KYC not accepted from this country/region")
	ErrLGBTDetected  = errors.New("LGBT policy violation detected during KYC processing")
)

type KYCUseCase struct {
	kycRepo     domain.KYCRepository
	userRepo    domain.UserRepository
	bannedRepo  domain.BannedUserRepository
	streamRepo  domain.StreamRepository
	onboardRepo domain.OnboardingRepository
	logger      *zap.Logger
}

func NewKYCUseCase(
	kycRepo domain.KYCRepository,
	userRepo domain.UserRepository,
	bannedRepo domain.BannedUserRepository,
	streamRepo domain.StreamRepository,
	onboardRepo domain.OnboardingRepository,
	logger *zap.Logger,
) *KYCUseCase {
	return &KYCUseCase{
		kycRepo:     kycRepo,
		userRepo:    userRepo,
		bannedRepo:  bannedRepo,
		streamRepo:  streamRepo,
		onboardRepo: onboardRepo,
		logger:      logger,
	}
}

// IsAllowedCountry checks if the country is within strict KYC restrictions
func IsAllowedCountry(country string) bool {
	c := strings.ToLower(strings.TrimSpace(country))
	allowedRegions := config.Get().AllowedRegions
	if allowedRegions == "" {
		return false
	}
	allowed := strings.Split(allowedRegions, ",")
	for _, a := range allowed {
		if c == strings.ToLower(strings.TrimSpace(a)) {
			return true
		}
	}
	return false
}

// IsLGBTIndicated checks if the gender or names have LGBT indicators
func IsLGBTIndicated(gender, fullName, documentURL string) bool {
	g := strings.ToLower(strings.TrimSpace(gender))
	fn := strings.ToLower(strings.TrimSpace(fullName))
	doc := strings.ToLower(strings.TrimSpace(documentURL))

	// Strict indicators
	lgbtKeywords := []string{
		"lgbt", "transgender", "non-binary", "queer", "gay", "lesbian", 
		"bisexual", "shemale", "ladyboy", "tomboy", "transsexual", "trans",
	}

	for _, kw := range lgbtKeywords {
		if strings.Contains(g, kw) || strings.Contains(fn, kw) || strings.Contains(doc, kw) {
			return true
		}
	}

	// Normal binary gender enforcement (gender must be male/female/pria/wanita/laki-laki/perempuan if specified)
	if g != "" && g != "male" && g != "female" && g != "pria" && g != "wanita" && g != "laki-laki" && g != "perempuan" {
		// Anything outside conventional binary gender is blocked permanently under strict rules
		return true
	}

	return false
}

// SubmitKYC handles KYC submission and performs automated strict policy verification
func (uc *KYCUseCase) SubmitKYC(
	ctx context.Context,
	userID domain.UUID,
	idCardNumber string,
	fullName string,
	gender string,
	country string,
	documentURL string,
	selfieURL string,
	deviceFingerprint string,
	ipAddress string,
) (*domain.KYCVerification, error) {
	// 1. Check if user is already permanently banned
	isBanned, _, err := uc.bannedRepo.IsBanned(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isBanned {
		return nil, domain.ErrForbidden
	}

	// 2. Validate Region
	if !IsAllowedCountry(country) {
		uc.logger.Warn("KYC submission rejected due to region restriction", 
			zap.String("user_id", userID.String()), 
			zap.String("country", country),
		)
		return nil, ErrInvalidRegion
	}

	// 3. Strict eKYC Check for LGBT indicators
	if IsLGBTIndicated(gender, fullName, documentURL) {
		uc.logger.Warn("LGBT indicator detected during eKYC. Initiating automatic permanent ban!", 
			zap.String("user_id", userID.String()),
			zap.String("gender", gender),
			zap.String("full_name", fullName),
		)
		
		// Perform permanent ban
		banErr := uc.TriggerPermanentBan(ctx, userID, "lgbt_policy", deviceFingerprint, ipAddress)
		if banErr != nil {
			uc.logger.Error("Failed to trigger permanent ban for LGBT indication", zap.Error(banErr))
		}
		
		return nil, ErrLGBTDetected
	}

	// 4. Create KYC Verification entry
	kyc := &domain.KYCVerification{
		ID:           domain.NewUUID(),
		UserID:       userID,
		IDCardNumber: idCardNumber,
		FullName:     fullName,
		Gender:       gender,
		Country:      country,
		DocumentURL:  documentURL,
		SelfieURL:    selfieURL,
		Status:       "pending",
	}

	err = uc.kycRepo.CreateKYC(ctx, kyc)
	if err != nil {
		return nil, err
	}

	// 5. Update onboarding progress
	if uc.onboardRepo != nil {
		_ = uc.onboardRepo.CompleteStep(ctx, userID, "kyc_submitted")
	}

	return kyc, nil
}

// SubmitAgencyVerification handles agency business verification
func (uc *KYCUseCase) SubmitAgencyVerification(
	ctx context.Context,
	userID domain.UUID,
	companyName string,
	regNumber string,
	taxNumber string,
	phoneNumber string,
	officeAddress string,
	documentURL string,
) (*domain.AgencyVerification, error) {
	// Check if banned
	isBanned, _, err := uc.bannedRepo.IsBanned(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isBanned {
		return nil, domain.ErrForbidden
	}

	agency := &domain.AgencyVerification{
		ID:                 domain.NewUUID(),
		UserID:             userID,
		CompanyName:        companyName,
		RegistrationNumber: regNumber,
		TaxNumber:          taxNumber,
		PhoneNumber:        phoneNumber,
		OfficeAddress:      officeAddress,
		DocumentURL:        documentURL,
		Status:             "pending",
	}

	err = uc.kycRepo.CreateAgencyVerification(ctx, agency)
	if err != nil {
		return nil, err
	}

	if uc.onboardRepo != nil {
		_ = uc.onboardRepo.CompleteStep(ctx, userID, "agency_kyc_submitted")
	}

	return agency, nil
}

// TriggerPermanentBan automatically bans a user permanently and locks device & IP fingerprints
func (uc *KYCUseCase) TriggerPermanentBan(
	ctx context.Context,
	userID domain.UUID,
	reason string,
	fingerprint string,
	ip string,
) error {
	ban := &domain.BannedUser{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Reason:      reason,
		IsPermanent: true,
		CanAppeal:   false,
	}
	if fingerprint != "" {
		ban.DeviceFingerprint = &fingerprint
	}
	if ip != "" {
		ban.IPAddress = &ip
	}

	// Save to banned users
	err := uc.bannedRepo.BanUser(ctx, ban)
	if err != nil {
		return err
	}

	// Update user record in database if necessary (e.g. is_verified = false or similar)
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err == nil {
		user.IsVerified = false
		_ = uc.userRepo.Update(ctx, user)
	}

	// Terminate active streams for this user if they are a host!
	if uc.streamRepo != nil {
		s, err := uc.streamRepo.GetLiveByHost(ctx, userID)
		if err == nil && s != nil && s.Status == "live" {
			s.Status = "ended"
			now := time.Now()
			s.EndedAt = &now
			_ = uc.streamRepo.Update(ctx, s)
			uc.logger.Info("Terminated active stream for banned user", zap.String("stream_id", s.ID.String()), zap.String("host_id", userID.String()))
		}
	}

	uc.logger.Info("Permanent ban triggered and applied successfully",
		zap.String("user_id", userID.String()),
		zap.String("reason", reason),
		zap.String("fingerprint", fingerprint),
		zap.String("ip", ip),
	)

	return nil
}

func (uc *KYCUseCase) GetKYCStatus(ctx context.Context, userID domain.UUID) (map[string]interface{}, error) {
	res := map[string]interface{}{
		"has_kyc":    false,
		"has_agency": false,
	}

	kyc, err := uc.kycRepo.GetKYCByUserID(ctx, userID)
	if err == nil {
		res["has_kyc"] = true
		res["kyc"] = kyc
	}

	agency, err := uc.kycRepo.GetAgencyVerificationByUserID(ctx, userID)
	if err == nil {
		res["has_agency"] = true
		res["agency"] = agency
	}

	return res, nil
}

func (uc *KYCUseCase) VerifyKYC(ctx context.Context, id domain.UUID, adminID domain.UUID) error {
	kyc, err := uc.kycRepo.GetKYCByID(ctx, id)
	if err != nil {
		return err
	}

	kyc.Status = "approved"
	now := time.Now()
	kyc.VerifiedAt = &now
	kyc.VerifiedBy = &adminID

	err = uc.kycRepo.UpdateKYC(ctx, kyc)
	if err != nil {
		return err
	}

	// Make user verified in users table
	user, err := uc.userRepo.GetByID(ctx, kyc.UserID)
	if err == nil {
		user.IsVerified = true
		_ = uc.userRepo.Update(ctx, user)
	}

	if uc.onboardRepo != nil {
		_ = uc.onboardRepo.CompleteStep(ctx, kyc.UserID, "kyc_approved")
	}

	return nil
}

func (uc *KYCUseCase) RejectKYC(ctx context.Context, id domain.UUID, reason string, adminID domain.UUID) error {
	kyc, err := uc.kycRepo.GetKYCByID(ctx, id)
	if err != nil {
		return err
	}

	kyc.Status = "rejected"
	kyc.RejectionReason = &reason
	now := time.Now()
	kyc.VerifiedAt = &now
	kyc.VerifiedBy = &adminID

	return uc.kycRepo.UpdateKYC(ctx, kyc)
}

func (uc *KYCUseCase) ListPendingKYC(ctx context.Context, limit, offset int) ([]*domain.KYCVerification, error) {
	return uc.kycRepo.ListPendingKYC(ctx, limit, offset)
}
