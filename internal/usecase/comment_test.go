package usecase

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

// MockCommentRepository
type mockCommentRepository struct {
	CreateFunc             func(ctx context.Context, comment *domain.Comment) error
	GetByIDFunc            func(ctx context.Context, id domain.UUID) (*domain.Comment, error)
	GetByContentIDFunc     func(ctx context.Context, contentID domain.UUID, contentType string, limit, offset int) ([]*domain.Comment, error)
	GetRepliesFunc         func(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.Comment, error)
	GetByUserFunc          func(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Comment, error)
	UpdateFunc             func(ctx context.Context, comment *domain.Comment) error
	DeleteFunc             func(ctx context.Context, id domain.UUID) error
	IncrementLikeCountFunc func(ctx context.Context, id domain.UUID) error
	CountByContentFunc     func(ctx context.Context, contentID domain.UUID, contentType string) (int, error)
}

func (m *mockCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	return m.CreateFunc(ctx, comment)
}
func (m *mockCommentRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Comment, error) {
	return m.GetByIDFunc(ctx, id)
}
func (m *mockCommentRepository) GetByContentID(ctx context.Context, contentID domain.UUID, contentType string, limit, offset int) ([]*domain.Comment, error) {
	return m.GetByContentIDFunc(ctx, contentID, contentType, limit, offset)
}
func (m *mockCommentRepository) GetReplies(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	return m.GetRepliesFunc(ctx, parentID, limit, offset)
}
func (m *mockCommentRepository) GetByUser(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	return m.GetByUserFunc(ctx, userID, limit, offset)
}
func (m *mockCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	return m.UpdateFunc(ctx, comment)
}
func (m *mockCommentRepository) Delete(ctx context.Context, id domain.UUID) error {
	return m.DeleteFunc(ctx, id)
}
func (m *mockCommentRepository) IncrementLikeCount(ctx context.Context, id domain.UUID) error {
	return m.IncrementLikeCountFunc(ctx, id)
}
func (m *mockCommentRepository) CountByContent(ctx context.Context, contentID domain.UUID, contentType string) (int, error) {
	return m.CountByContentFunc(ctx, contentID, contentType)
}

// MockCommentLikeRepository
type mockCommentLikeRepository struct {
	CreateFunc           func(ctx context.Context, like *domain.CommentLike) error
	DeleteFunc           func(ctx context.Context, userID, commentID domain.UUID) error
	HasLikedFunc         func(ctx context.Context, userID, commentID domain.UUID) (bool, error)
	CountByCommentIDFunc func(ctx context.Context, commentID domain.UUID) (int, error)
}

func (m *mockCommentLikeRepository) Create(ctx context.Context, like *domain.CommentLike) error {
	return m.CreateFunc(ctx, like)
}
func (m *mockCommentLikeRepository) Delete(ctx context.Context, userID, commentID domain.UUID) error {
	return m.DeleteFunc(ctx, userID, commentID)
}
func (m *mockCommentLikeRepository) HasLiked(ctx context.Context, userID, commentID domain.UUID) (bool, error) {
	return m.HasLikedFunc(ctx, userID, commentID)
}
func (m *mockCommentLikeRepository) CountByCommentID(ctx context.Context, commentID domain.UUID) (int, error) {
	return m.CountByCommentIDFunc(ctx, commentID)
}

func TestCommentUseCase_CreateComment(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateCommentRequest
		mock    func(*mockCommentRepository)
		wantErr bool
	}{
		{
			name: "Success",
			req: &CreateCommentRequest{
				ContentID:   domain.NewUUID(),
				ContentType: "stream",
				Content:     "Hello stream",
			},
			mock: func(r *mockCommentRepository) {
				r.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DB Error",
			req: &CreateCommentRequest{
				ContentID:   domain.NewUUID(),
				ContentType: "stream",
				Content:     "Hello stream",
			},
			mock: func(r *mockCommentRepository) {
				r.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
					return errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			commentRepo := &mockCommentRepository{}
			if tt.mock != nil {
				tt.mock(commentRepo)
			}
			uc := NewCommentUseCase(commentRepo, nil, nil, logger)

			userID := domain.NewUUID()
			_, err := uc.CreateComment(context.Background(), userID, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateComment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommentUseCase_LikeComment(t *testing.T) {
	tests := []struct {
		name     string
		mockLike func(*mockCommentLikeRepository)
		mockComm func(*mockCommentRepository)
		wantErr  bool
	}{
		{
			name: "Success",
			mockLike: func(r *mockCommentLikeRepository) {
				r.HasLikedFunc = func(ctx context.Context, userID, commentID domain.UUID) (bool, error) {
					return false, nil
				}
				r.CreateFunc = func(ctx context.Context, like *domain.CommentLike) error {
					return nil
				}
			},
			mockComm: func(r *mockCommentRepository) {
				r.IncrementLikeCountFunc = func(ctx context.Context, id domain.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "Already Liked",
			mockLike: func(r *mockCommentLikeRepository) {
				r.HasLikedFunc = func(ctx context.Context, userID, commentID domain.UUID) (bool, error) {
					return true, nil
				}
			},
			mockComm: func(r *mockCommentRepository) {},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			likeRepo := &mockCommentLikeRepository{}
			if tt.mockLike != nil {
				tt.mockLike(likeRepo)
			}
			commentRepo := &mockCommentRepository{}
			if tt.mockComm != nil {
				tt.mockComm(commentRepo)
			}

			uc := NewCommentUseCase(commentRepo, likeRepo, nil, logger)

			userID := domain.NewUUID()
			commentID := domain.NewUUID()
			err := uc.LikeComment(context.Background(), userID, commentID)

			if (err != nil) != tt.wantErr {
				t.Errorf("LikeComment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
