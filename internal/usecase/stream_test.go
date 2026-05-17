package usecase

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/mux"
)

type mockMuxClient struct {
	CreateLiveStreamFunc func() (*mux.LiveStreamResponse, error)
}

func (m *mockMuxClient) CreateLiveStream() (*mux.LiveStreamResponse, error) {
	if m.CreateLiveStreamFunc != nil {
		return m.CreateLiveStreamFunc()
	}
	res := &mux.LiveStreamResponse{}
	res.Data.ID = "test-stream-id"
	res.Data.StreamKey = "test-stream-key"
	res.Data.PlaybackIDs = []struct {
		ID     string `json:"id"`
		Policy string `json:"policy"`
	}{{ID: "test-playback-id", Policy: "public"}}
	return res, nil
}

func (m *mockMuxClient) GetPlaybackURL(playbackID string) string {
	return "https://stream.mux.com/" + playbackID + ".m3u8"
}

type mockStreamRepo struct {
	GetLiveByHostFunc func(ctx context.Context, hostID domain.UUID) (*domain.Stream, error)
	CreateFunc        func(ctx context.Context, stream *domain.Stream) error
}

func (m *mockStreamRepo) Create(ctx context.Context, stream *domain.Stream) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, stream)
	}
	return nil
}
func (m *mockStreamRepo) Update(ctx context.Context, stream *domain.Stream) error { return nil }
func (m *mockStreamRepo) GetByID(ctx context.Context, id domain.UUID) (*domain.Stream, error) {
	return nil, nil
}
func (m *mockStreamRepo) GetByRoomID(ctx context.Context, roomID domain.UUID) (*domain.Stream, error) {
	return nil, nil
}
func (m *mockStreamRepo) GetLiveByHost(ctx context.Context, hostID domain.UUID) (*domain.Stream, error) {
	if m.GetLiveByHostFunc != nil {
		return m.GetLiveByHostFunc(ctx, hostID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockStreamRepo) ListLive(ctx context.Context, limit, offset int) ([]*domain.Stream, error) {
	return nil, nil
}

type mockStreamSessionRepo struct{}

func (m *mockStreamSessionRepo) Create(ctx context.Context, session *domain.StreamSession) error {
	return nil
}
func (m *mockStreamSessionRepo) Update(ctx context.Context, session *domain.StreamSession) error {
	return nil
}
func (m *mockStreamSessionRepo) GetActiveSession(ctx context.Context, streamID, viewerID domain.UUID) (*domain.StreamSession, error) {
	return nil, nil
}

func TestStreamUseCase_CreateStream(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hostID := domain.NewUUID()

	repo := &mockStreamRepo{
		GetLiveByHostFunc: func(ctx context.Context, hostID domain.UUID) (*domain.Stream, error) {
			// No existing stream
			return nil, domain.ErrNotFound
		},
	}

	uc := NewStreamUseCase(repo, &mockStreamSessionRepo{}, nil, nil, nil, logger)
	uc.SetMuxClient(&mockMuxClient{})

	stream, err := uc.CreateStream(context.Background(), hostID, domain.CreateStreamInput{
		Title:        "Title",
		Description:  "Desc",
		ThumbnailURL: "url",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stream.Title != "Title" {
		t.Errorf("Expected title 'Title', got '%s'", stream.Title)
	}
}
