package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type OnboardingUseCase struct {
	onboardRepo domain.OnboardingRepository
	userRepo    domain.UserRepository
	logger      *zap.Logger
}

func NewOnboardingUseCase(
	onboardRepo domain.OnboardingRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *OnboardingUseCase {
	return &OnboardingUseCase{
		onboardRepo: onboardRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// GetChecklist returns the role-specific onboarding checklist for a user
func (uc *OnboardingUseCase) GetChecklist(ctx context.Context, userID domain.UUID, roleType string) (*domain.OnboardingChecklist, error) {
	// Try to get existing progress
	progress, err := uc.onboardRepo.GetProgress(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			// Initialize new progress
			progress, err = uc.onboardRepo.InitProgress(ctx, userID, roleType)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Double check user entity database values to auto-complete steps if already done in users table
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err == nil {
		// If email is verified, autocomplete email_verified
		if user.IsVerified {
			_ = uc.onboardRepo.CompleteStep(ctx, userID, "email_verified")
		}
		// If profile has username and avatar, autocomplete profile
		if user.Username != "" && user.AvatarURL != nil && *user.AvatarURL != "" {
			_ = uc.onboardRepo.CompleteStep(ctx, userID, "profile")
		}
	}

	// Refetch progress
	progress, _ = uc.onboardRepo.GetProgress(ctx, userID)

	checklist := &domain.OnboardingChecklist{
		RoleType:    progress.RoleType,
		IsCompleted: progress.IsCompleted,
		Steps:       make([]domain.OnboardingStep, 0),
	}

	// Build steps based on role
	var steps []domain.OnboardingStep
	switch progress.RoleType {
	case "user":
		steps = []domain.OnboardingStep{
			{Key: "profile", Title: "Lengkapi Profil", Description: "Tambahkan foto profil dan biodata Anda."},
			{Key: "email_verified", Title: "Verifikasi Email", Description: "Verifikasi alamat email Anda demi keamanan akun."},
			{Key: "language_preferences", Title: "Atur Preferensi Bahasa", Description: "Pilih bahasa utama yang ingin Anda gunakan."},
		}
	case "host":
		steps = []domain.OnboardingStep{
			{Key: "profile", Title: "Lengkapi Profil", Description: "Lengkapi biodata dan informasi profil siaran Anda."},
			{Key: "kyc_approved", Title: "Upload KYC", Description: "Unggah KTP/Paspor dan selfie liveness untuk verifikasi identitas host."},
			{Key: "payment_method", Title: "Atur Metode Pembayaran", Description: "Tambahkan wallet kripto atau rekening bank untuk withdraw pendapatan."},
			{Key: "stream_key", Title: "Konfigurasi Stream Key", Description: "Unduh stream key untuk software streaming Anda (OBS/Streamlabs)."},
			{Key: "content_guidelines", Title: "Atur Pedoman Konten Pribadi", Description: "Tentukan aturan dan batas konten di ruang obrolan pribadi Anda."},
		}
	case "agency":
		steps = []domain.OnboardingStep{
			{Key: "business_profile", Title: "Lengkapi Profil Bisnis", Description: "Lengkapi nama agensi, deskripsi, dan alamat kantor fisik."},
			{Key: "agency_kyc_approved", Title: "Upload KYC Bisnis", Description: "Unggah Akta Pendirian, NPWP Badan, dan NIB/SIUP agensi Anda."},
			{Key: "hosts_added", Title: "Tambahkan Host", Description: "Undang minimal satu host untuk masuk ke dalam manajemen agensi Anda."},
			{Key: "commission_set", Title: "Atur Komisi", Description: "Tentukan persentase pembagian hasil/komisi untuk host Anda."},
		}
	}

	// Check which steps are completed
	completedCount := 0
	for i, step := range steps {
		isComp := false
		for _, comp := range progress.StepsCompleted {
			if comp == step.Key {
				isComp = true
				break
			}
		}
		steps[i].IsCompleted = isComp
		if isComp {
			completedCount++
		}
	}

	checklist.Steps = steps
	if len(steps) > 0 {
		checklist.ProgressPct = (completedCount * 100) / len(steps)
	}

	return checklist, nil
}

// CompleteStep manual completion trigger for onboarding steps
func (uc *OnboardingUseCase) CompleteStep(ctx context.Context, userID domain.UUID, stepKey string) error {
	return uc.onboardRepo.CompleteStep(ctx, userID, stepKey)
}
