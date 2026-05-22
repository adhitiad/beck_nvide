package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type ShortVideoUseCase struct {
	videoRepo domain.ShortVideoRepository
	tagRepo   domain.StreamTagRepository
	redis     *redis.Client
	logger    *zap.Logger
}

func NewShortVideoUseCase(
	videoRepo domain.ShortVideoRepository,
	tagRepo domain.StreamTagRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *ShortVideoUseCase {
	return &ShortVideoUseCase{
		videoRepo: videoRepo,
		tagRepo:   tagRepo,
		redis:     redis,
		logger:    logger,
	}
}

// Upload creates a new short video
func (uc *ShortVideoUseCase) Upload(ctx context.Context, userID domain.UUID, input domain.CreateShortVideoInput) (*domain.ShortVideo, error) {
	if input.Duration > 60 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "video duration cannot exceed 60 seconds", nil)
	}
	if input.VideoURL == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "video URL is required", nil)
	}

	video := &domain.ShortVideo{
		ID:           domain.NewUUID(),
		UserID:       userID,
		VideoURL:     input.VideoURL,
		ThumbnailURL: input.ThumbnailURL,
		Caption:      input.Caption,
		Duration:     input.Duration,
		Status:       domain.ShortVideoStatusActive,
		Tags:         input.Tags,
	}

	if err := uc.videoRepo.Create(ctx, video); err != nil {
		return nil, err
	}

	uc.logger.Info("Short video uploaded",
		zap.String("video_id", string(video.ID)),
		zap.String("user_id", string(userID)),
	)

	return video, nil
}

// GetByID returns a short video by ID
func (uc *ShortVideoUseCase) GetByID(ctx context.Context, videoID domain.UUID) (*domain.ShortVideo, error) {
	return uc.videoRepo.GetByID(ctx, videoID)
}

// GetFeed returns the short video feed for a user
func (uc *ShortVideoUseCase) GetFeed(ctx context.Context, viewerID domain.UUID, limit, offset int) ([]*domain.ShortVideo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return uc.videoRepo.GetFeed(ctx, viewerID, limit, offset)
}

// GetUserVideos returns videos by a specific user
func (uc *ShortVideoUseCase) GetUserVideos(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.ShortVideo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return uc.videoRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetTrending returns trending short videos
func (uc *ShortVideoUseCase) GetTrending(ctx context.Context, limit, offset int) ([]*domain.ShortVideo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return uc.videoRepo.GetTrending(ctx, limit, offset)
}

// RecordView increments view count
func (uc *ShortVideoUseCase) RecordView(ctx context.Context, videoID domain.UUID) error {
	return uc.videoRepo.IncrementViewCount(ctx, videoID)
}

// Like likes a short video
func (uc *ShortVideoUseCase) Like(ctx context.Context, userID, videoID domain.UUID) error {
	like := &domain.ShortVideoLike{
		ID:      domain.NewUUID(),
		VideoID: videoID,
		UserID:  userID,
	}
	return uc.videoRepo.Like(ctx, like)
}

// Unlike unlikes a short video
func (uc *ShortVideoUseCase) Unlike(ctx context.Context, userID, videoID domain.UUID) error {
	return uc.videoRepo.Unlike(ctx, videoID, userID)
}

// Comment adds a comment to a short video
func (uc *ShortVideoUseCase) Comment(ctx context.Context, userID, videoID domain.UUID, content string, parentID *domain.UUID) (*domain.ShortVideoComment, error) {
	if content == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "comment content cannot be empty", nil)
	}

	comment := &domain.ShortVideoComment{
		ID:       domain.NewUUID(),
		VideoID:  videoID,
		UserID:   userID,
		Content:  content,
		ParentID: parentID,
	}

	if err := uc.videoRepo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

// GetComments returns comments for a video
func (uc *ShortVideoUseCase) GetComments(ctx context.Context, videoID domain.UUID, limit, offset int) ([]*domain.ShortVideoComment, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return uc.videoRepo.GetComments(ctx, videoID, limit, offset)
}

// Share records a share event
func (uc *ShortVideoUseCase) Share(ctx context.Context, videoID domain.UUID) error {
	return uc.videoRepo.IncrementShareCount(ctx, videoID)
}

// DeleteVideo soft-deletes a video (owner only)
func (uc *ShortVideoUseCase) DeleteVideo(ctx context.Context, userID, videoID domain.UUID) error {
	video, err := uc.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "video not found", err)
	}
	if video.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "you can only delete your own videos", nil)
	}
	return uc.videoRepo.Delete(ctx, videoID)
}
