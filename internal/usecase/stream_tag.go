package usecase

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type StreamTagUseCase struct {
	tagRepo domain.StreamTagRepository
	logger  *zap.Logger
}

func NewStreamTagUseCase(tagRepo domain.StreamTagRepository, logger *zap.Logger) *StreamTagUseCase {
	return &StreamTagUseCase{tagRepo: tagRepo, logger: logger}
}

// AutoTagStream automatically assigns tags to a stream based on its title
func (uc *StreamTagUseCase) AutoTagStream(ctx context.Context, streamID domain.UUID, title string) error {
	// Clear existing auto-tags
	_ = uc.tagRepo.ClearStreamTags(ctx, streamID)

	allTags, err := uc.tagRepo.ListAll(ctx)
	if err != nil {
		return err
	}

	titleLower := strings.ToLower(title)
	for _, tag := range allTags {
		for _, keyword := range tag.Keywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				_ = uc.tagRepo.AddTagToStream(ctx, streamID, tag.ID)
				break // Only add the tag once
			}
		}
	}
	return nil
}

// GetStreamTags returns tags for a stream
func (uc *StreamTagUseCase) GetStreamTags(ctx context.Context, streamID domain.UUID) ([]*domain.StreamTag, error) {
	return uc.tagRepo.GetStreamTags(ctx, streamID)
}

// ListAllTags returns all available tags
func (uc *StreamTagUseCase) ListAllTags(ctx context.Context) ([]*domain.StreamTag, error) {
	return uc.tagRepo.ListAll(ctx)
}
