package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

// UserUseCase handles user business logic
type UserUseCase struct {
	userRepo domain.UserRepository
	redis    *redis.Client
	logger   *zap.Logger
	sf       singleflight.Group
}

// NewUserUseCase creates new user usecase
func NewUserUseCase(userRepo domain.UserRepository, redis *redis.Client, logger *zap.Logger) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
		redis:    redis,
		logger:   logger,
	}
}

// GetProfile returns user profile by ID (Cached for 1 hour with Cache Stampede prevention)
func (uc *UserUseCase) GetProfile(ctx context.Context, userID domain.UUID) (*domain.User, error) {
	cacheKey := fmt.Sprintf("user:profile:%s", userID.String())

	// 1. Try cache first
	if uc.redis != nil {
		cached, err := uc.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var user domain.User
			if err := json.Unmarshal([]byte(cached), &user); err == nil {
				return &user, nil
			}
		}
	}

	// 2. Cache Stampede prevention using Singleflight
	val, err, _ := uc.sf.Do(userID.String(), func() (interface{}, error) {
		user, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}

		// Remove sensitive data
		user.PasswordHash = ""
		user.VerificationToken = nil
		user.ResetToken = nil

		// Save to cache (1 hour TTL)
		if uc.redis != nil {
			data, err := json.Marshal(user)
			if err == nil {
				_ = uc.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
			}
		}

		return user, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*domain.User), nil
}

// UpdateProfile updates user profile and invalidates cache
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

	// Invalidate Cache
	if uc.redis != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
		_ = uc.redis.GetClient().Del(ctx, cacheKey)
	}

	user.PasswordHash = ""
	user.VerificationToken = nil
	user.ResetToken = nil

	return user, nil
}

// DeleteAccount deletes user account and invalidates cache
func (uc *UserUseCase) DeleteAccount(ctx context.Context, userID domain.UUID) error {
	// Invalidate Cache
	if uc.redis != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
		_ = uc.redis.GetClient().Del(ctx, cacheKey)
	}
	return uc.userRepo.Delete(ctx, userID)
}
