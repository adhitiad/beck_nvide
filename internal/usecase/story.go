package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// StoryUseCase handles story business logic
type StoryUseCase struct {
	storyRepo     domain.StoryRepository
	storyViewRepo domain.StoryViewRepository
	userRepo      domain.UserRepository
	logger        *zap.Logger
}

// NewStoryUseCase creates new story usecase
func NewStoryUseCase(
	storyRepo domain.StoryRepository,
	storyViewRepo domain.StoryViewRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *StoryUseCase {
	return &StoryUseCase{
		storyRepo:     storyRepo,
		storyViewRepo: storyViewRepo,
		userRepo:      userRepo,
		logger:        logger,
	}
}

// CreateStoryRequest represents story creation request
type CreateStoryRequest struct {
	Content   string `json:"content"`
	MediaType string `json:"media_type"` // "text", "image", "video"
}

// CreateStory creates a new story
func (uc *StoryUseCase) CreateStory(ctx context.Context, userID domain.UUID, req *CreateStoryRequest) (*domain.Story, error) {
	story := &domain.Story{
		ID:        domain.NewUUID(),
		UserID:    userID,
		Content:   req.Content,
		MediaType: req.MediaType,
		ExpiresAt: time.Now().Add(168 * time.Hour), // 1 week expiry
	}

	if err := uc.storyRepo.Create(ctx, story); err != nil {
		return nil, err
	}

	uc.logger.Info("Story created", zap.String("story_id", story.ID.String()), zap.String("user_id", userID.String()))
	return story, nil
}

// GetStory gets a story by ID
func (uc *StoryUseCase) GetStory(ctx context.Context, id domain.UUID) (*domain.Story, error) {
	story, err := uc.storyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load user
	user, err := uc.userRepo.GetByID(ctx, story.UserID)
	if err == nil {
		story.User = user
	}

	return story, nil
}

// GetUserStories gets stories by user ID
func (uc *StoryUseCase) GetUserStories(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Story, error) {
	return uc.storyRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetFeedStories gets stories from followed users
func (uc *StoryUseCase) GetFeedStories(ctx context.Context, userID domain.UUID, limit int) ([]*domain.Story, error) {
	// For now, get all active stories
	return uc.storyRepo.GetActiveStories(ctx, userID)
}

// ViewStory records a story view
func (uc *StoryUseCase) ViewStory(ctx context.Context, userID, storyID domain.UUID) error {
	// Check if already viewed
	hasViewed, err := uc.storyViewRepo.HasViewed(ctx, userID, storyID)
	if err != nil {
		return err
	}

	if !hasViewed {
		view := &domain.StoryView{
			ID:      domain.NewUUID(),
			StoryID: storyID,
			UserID:  userID,
		}
		if err := uc.storyViewRepo.Create(ctx, view); err != nil {
			return err
		}
		// Increment view count
		uc.storyRepo.IncrementViewCount(ctx, storyID)
	}

	return nil
}

// DeleteStory deletes a story
func (uc *StoryUseCase) DeleteStory(ctx context.Context, id domain.UUID) error {
	return uc.storyRepo.Delete(ctx, id)
}
