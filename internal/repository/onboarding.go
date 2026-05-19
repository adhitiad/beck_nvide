package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type onboardingRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewOnboardingRepository creates new onboarding repository
func NewOnboardingRepository(db *pgxpool.Pool, logger *zap.Logger) domain.OnboardingRepository {
	return &onboardingRepository{
		db:     db,
		logger: logger,
	}
}

func (r *onboardingRepository) GetProgress(ctx context.Context, userID domain.UUID) (*domain.OnboardingProgress, error) {
	query := `
		SELECT user_id, role_type, steps_completed, is_completed, updated_at
		FROM onboarding_progress
		WHERE user_id = $1
	`
	progress := &domain.OnboardingProgress{}
	var stepsRaw []byte

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&progress.UserID, &progress.RoleType, &stepsRaw, &progress.IsCompleted, &progress.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get onboarding progress", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	if len(stepsRaw) > 0 {
		var steps []string
		if err := json.Unmarshal(stepsRaw, &steps); err == nil {
			progress.StepsCompleted = steps
		}
	}

	return progress, nil
}

func (r *onboardingRepository) SaveProgress(ctx context.Context, progress *domain.OnboardingProgress) error {
	query := `
		INSERT INTO onboarding_progress (user_id, role_type, steps_completed, is_completed, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET steps_completed = EXCLUDED.steps_completed,
		    is_completed = EXCLUDED.is_completed,
		    updated_at = NOW()
	`
	stepsRaw, err := json.Marshal(progress.StepsCompleted)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query,
		progress.UserID,
		progress.RoleType,
		stepsRaw,
		progress.IsCompleted,
	)

	if err != nil {
		r.logger.Error("Failed to save onboarding progress", zap.Error(err), zap.String("user_id", progress.UserID.String()))
		return err
	}

	return nil
}

func (r *onboardingRepository) CompleteStep(ctx context.Context, userID domain.UUID, stepKey string) error {
	progress, err := r.GetProgress(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			// Cannot complete step if not initialized, do nothing
			return nil
		}
		return err
	}

	// Check if already completed
	for _, step := range progress.StepsCompleted {
		if step == stepKey {
			return nil
		}
	}

	progress.StepsCompleted = append(progress.StepsCompleted, stepKey)
	progress.UpdatedAt = time.Now()

	// Update is_completed based on role requirements
	progress.IsCompleted = r.checkCompletion(progress.RoleType, progress.StepsCompleted)

	return r.SaveProgress(ctx, progress)
}

func (r *onboardingRepository) InitProgress(ctx context.Context, userID domain.UUID, roleType string) (*domain.OnboardingProgress, error) {
	progress := &domain.OnboardingProgress{
		UserID:         userID,
		RoleType:       roleType,
		StepsCompleted: make([]string, 0),
		IsCompleted:    false,
		UpdatedAt:      time.Now(),
	}

	err := r.SaveProgress(ctx, progress)
	if err != nil {
		return nil, err
	}

	return progress, nil
}

func (r *onboardingRepository) checkCompletion(role string, completed []string) bool {
	requiredSteps := make([]string, 0)
	switch role {
	case "user":
		requiredSteps = []string{"profile", "email_verified", "language_preferences"}
	case "host":
		requiredSteps = []string{"profile", "kyc_approved", "payment_method", "stream_key", "content_guidelines"}
	case "agency":
		requiredSteps = []string{"business_profile", "agency_kyc_approved", "hosts_added", "commission_set"}
	}

	if len(requiredSteps) == 0 {
		return false
	}

	for _, req := range requiredSteps {
		found := false
		for _, comp := range completed {
			if comp == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
