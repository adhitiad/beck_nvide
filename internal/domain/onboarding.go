package domain

import (
	"context"
	"time"
)

// OnboardingProgress tracks role-specific onboarding checklists
type OnboardingProgress struct {
	UserID         UUID      `json:"user_id" db:"user_id"`
	RoleType       string    `json:"role_type" db:"role_type"`       // 'user', 'host', 'agency'
	StepsCompleted []string  `json:"steps_completed" db:"steps_completed"` // e.g. ["profile", "email_verified"]
	IsCompleted    bool      `json:"is_completed" db:"is_completed"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// OnboardingStep defines a single onboarding step item
type OnboardingStep struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsCompleted bool   `json:"is_completed"`
}

// OnboardingChecklist represents the checklist response for the user
type OnboardingChecklist struct {
	RoleType    string            `json:"role_type"`
	IsCompleted bool              `json:"is_completed"`
	Steps       []OnboardingStep  `json:"steps"`
	ProgressPct int               `json:"progress_percentage"`
}

// OnboardingRepository defines data access methods for onboarding progress
type OnboardingRepository interface {
	GetProgress(ctx context.Context, userID UUID) (*OnboardingProgress, error)
	SaveProgress(ctx context.Context, progress *OnboardingProgress) error
	CompleteStep(ctx context.Context, userID UUID, stepKey string) error
	InitProgress(ctx context.Context, userID UUID, roleType string) (*OnboardingProgress, error)
}
