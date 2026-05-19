package usecase

import (
	"context"
	"testing"
	"time"

	"nvide-live/internal/domain"
)

type mockOnboardRepo struct {
	progress map[domain.UUID]*domain.OnboardingProgress
}

func newMockOnboardRepo() *mockOnboardRepo {
	return &mockOnboardRepo{
		progress: make(map[domain.UUID]*domain.OnboardingProgress),
	}
}

func (m *mockOnboardRepo) GetProgress(ctx context.Context, userID domain.UUID) (*domain.OnboardingProgress, error) {
	p, ok := m.progress[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (m *mockOnboardRepo) SaveProgress(ctx context.Context, progress *domain.OnboardingProgress) error {
	m.progress[progress.UserID] = progress
	return nil
}

func (m *mockOnboardRepo) CompleteStep(ctx context.Context, userID domain.UUID, stepKey string) error {
	p, ok := m.progress[userID]
	if !ok {
		return nil
	}
	for _, s := range p.StepsCompleted {
		if s == stepKey {
			return nil
		}
	}
	p.StepsCompleted = append(p.StepsCompleted, stepKey)
	p.UpdatedAt = time.Now()
	return m.SaveProgress(ctx, p)
}

func (m *mockOnboardRepo) InitProgress(ctx context.Context, userID domain.UUID, roleType string) (*domain.OnboardingProgress, error) {
	p := &domain.OnboardingProgress{
		UserID:         userID,
		RoleType:       roleType,
		StepsCompleted: make([]string, 0),
		IsCompleted:    false,
		UpdatedAt:      time.Now(),
	}
	m.progress[userID] = p
	return p, nil
}

type mockUserRepo struct {
	domain.UserRepository
}

func (m *mockUserRepo) GetByID(ctx context.Context, id domain.UUID) (*domain.User, error) {
	return &domain.User{
		ID:         id,
		Username:   "testuser",
		IsVerified: true,
	}, nil
}

func TestOnboardingUseCase_GetChecklist(t *testing.T) {
	repo := newMockOnboardRepo()
	userRepo := &mockUserRepo{}
	uc := NewOnboardingUseCase(repo, userRepo, nil)

	userID := domain.NewUUID()

	// 1. Get checklist for User
	t.Run("User checklist", func(t *testing.T) {
		checklist, err := uc.GetChecklist(context.Background(), userID, "user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checklist.RoleType != "user" {
			t.Errorf("expected roleType 'user', got %s", checklist.RoleType)
		}
		if len(checklist.Steps) != 3 {
			t.Errorf("expected 3 steps, got %d", len(checklist.Steps))
		}
		if checklist.Steps[0].Key != "profile" || checklist.Steps[0].Title != "Lengkapi Profil" {
			t.Errorf("invalid first step: %+v", checklist.Steps[0])
		}
	})

	// 2. Get checklist for Host
	t.Run("Host checklist", func(t *testing.T) {
		hostID := domain.NewUUID()
		checklist, err := uc.GetChecklist(context.Background(), hostID, "host")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checklist.RoleType != "host" {
			t.Errorf("expected roleType 'host', got %s", checklist.RoleType)
		}
		if len(checklist.Steps) != 5 {
			t.Errorf("expected 5 steps, got %d", len(checklist.Steps))
		}
		if checklist.Steps[1].Key != "kyc_approved" || checklist.Steps[1].Title != "Upload KYC" {
			t.Errorf("invalid second step: %+v", checklist.Steps[1])
		}
	})

	// 3. Get checklist for Agency
	t.Run("Agency checklist", func(t *testing.T) {
		agencyID := domain.NewUUID()
		checklist, err := uc.GetChecklist(context.Background(), agencyID, "agency")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checklist.RoleType != "agency" {
			t.Errorf("expected roleType 'agency', got %s", checklist.RoleType)
		}
		if len(checklist.Steps) != 4 {
			t.Errorf("expected 4 steps, got %d", len(checklist.Steps))
		}
		if checklist.Steps[3].Key != "commission_set" || checklist.Steps[3].Title != "Atur Komisi" {
			t.Errorf("invalid fourth step: %+v", checklist.Steps[3])
		}
	})
}
