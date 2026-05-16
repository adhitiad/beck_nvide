package usecase

import (
	"context"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// CommentUseCase handles comment business logic
type CommentUseCase struct {
	commentRepo     domain.CommentRepository
	commentLikeRepo domain.CommentLikeRepository
	userRepo        domain.UserRepository
	logger          *zap.Logger
}

// NewCommentUseCase creates new comment usecase
func NewCommentUseCase(
	commentRepo domain.CommentRepository,
	commentLikeRepo domain.CommentLikeRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *CommentUseCase {
	return &CommentUseCase{
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		userRepo:        userRepo,
		logger:          logger,
	}
}

// CreateCommentRequest represents comment creation request
type CreateCommentRequest struct {
	ContentID   domain.UUID  `json:"content_id"`
	ContentType string       `json:"content_type"` // "stream", "vod", "story"
	ParentID    *domain.UUID `json:"parent_id,omitempty"`
	Content     string       `json:"content"`
}

// CreateComment creates a new comment
func (uc *CommentUseCase) CreateComment(ctx context.Context, userID domain.UUID, req *CreateCommentRequest) (*domain.Comment, error) {
	comment := &domain.Comment{
		ID:          domain.NewUUID(),
		UserID:      userID,
		ContentID:   req.ContentID,
		ContentType: req.ContentType,
		ParentID:    req.ParentID,
		Content:     req.Content,
	}

	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	uc.logger.Info("Comment created", zap.String("comment_id", comment.ID.String()), zap.String("user_id", userID.String()))
	return comment, nil
}

// GetComments gets comments for content
func (uc *CommentUseCase) GetComments(ctx context.Context, contentID domain.UUID, contentType string, limit, offset int) ([]*domain.Comment, error) {
	return uc.commentRepo.GetByContentID(ctx, contentID, contentType, limit, offset)
}

// GetComment gets a comment by ID
func (uc *CommentUseCase) GetComment(ctx context.Context, id domain.UUID) (*domain.Comment, error) {
	return uc.commentRepo.GetByID(ctx, id)
}

// GetReplies gets replies to a comment
func (uc *CommentUseCase) GetReplies(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	return uc.commentRepo.GetReplies(ctx, parentID, limit, offset)
}

// UpdateComment updates a comment
func (uc *CommentUseCase) UpdateComment(ctx context.Context, id domain.UUID, content string) error {
	comment, err := uc.commentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	comment.Content = content
	return uc.commentRepo.Update(ctx, comment)
}

// DeleteComment deletes a comment
func (uc *CommentUseCase) DeleteComment(ctx context.Context, id domain.UUID) error {
	return uc.commentRepo.Delete(ctx, id)
}

// LikeComment likes a comment
func (uc *CommentUseCase) LikeComment(ctx context.Context, userID, commentID domain.UUID) error {
	hasLiked, err := uc.commentLikeRepo.HasLiked(ctx, userID, commentID)
	if err != nil {
		return err
	}

	if hasLiked {
		return domain.NewDomainError(domain.ErrCodeConflict, "already liked", nil)
	}

	like := &domain.CommentLike{
		ID:        domain.NewUUID(),
		UserID:    userID,
		CommentID: commentID,
	}

	if err := uc.commentLikeRepo.Create(ctx, like); err != nil {
		return err
	}

	// Increment like count
	uc.commentRepo.IncrementLikeCount(ctx, commentID)
	return nil
}

// UnlikeComment unlikes a comment
func (uc *CommentUseCase) UnlikeComment(ctx context.Context, userID, commentID domain.UUID) error {
	if err := uc.commentLikeRepo.Delete(ctx, userID, commentID); err != nil {
		return err
	}
	// Note: We don't decrement like count to avoid negative numbers
	return nil
}

// HasLiked checks if user has liked a comment
func (uc *CommentUseCase) HasLiked(ctx context.Context, userID, commentID domain.UUID) (bool, error) {
	return uc.commentLikeRepo.HasLiked(ctx, userID, commentID)
}
