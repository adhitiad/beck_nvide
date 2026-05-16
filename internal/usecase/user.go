package usecase

import (
	"context"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// UserUseCase handles user business logic
type UserUseCase struct {
	userRepo domain.UserRepository
	logger   *zap.Logger
}

// NewUserUseCase creates new user usecase
func NewUserUseCase(userRepo domain.UserRepository, logger *zap.Logger) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetProfile returns user profile by ID
func (uc *UserUseCase) GetProfile(ctx context.Context, userID domain.UUID) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	user.PasswordHash = ""
	user.VerificationToken = nil
	user.ResetToken = nil

	return user, nil
}

// UpdateProfile updates user profile
func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID domain.UUID, updates map[string]interface{}) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if username, ok := updates["username"].(string); ok && username != "" {
		// Check username uniqueness
		exists, err := uc.userRepo.ExistsByUsername(ctx, username)
		if err != nil {
			return nil, err
		}
		if exists && username != user.Username {
			return nil, domain.NewDomainError(domain.ErrCodeConflict, "username already taken", nil)
		}
		user.Username = username
	}

	if avatarURL, ok := updates["avatar_url"].(string); ok {
		user.AvatarURL = &avatarURL
	}

	// Validate and save
	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	user.VerificationToken = nil
	user.ResetToken = nil

	return user, nil
}

// DeleteAccount deletes user account
func (uc *UserUseCase) DeleteAccount(ctx context.Context, userID domain.UUID) error {
	// In production, you might want soft delete instead
	return uc.userRepo.Delete(ctx, userID)
}
