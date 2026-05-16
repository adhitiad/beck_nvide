package usecase

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/broker"
	"nvide-live/pkg/mux"
	"nvide-live/pkg/redis"
)

type StreamUseCase struct {
	streamRepo    domain.StreamRepository
	sessionRepo   domain.StreamSessionRepository
	redisClient   *redis.Client
	muxClient     *mux.Client
	broker        broker.Broker
	logger        *zap.Logger
}

func NewStreamUseCase(
	streamRepo domain.StreamRepository,
	sessionRepo domain.StreamSessionRepository,
	redisClient *redis.Client,
	broker broker.Broker,
	logger *zap.Logger,
) *StreamUseCase {
	return &StreamUseCase{
		streamRepo:  streamRepo,
		sessionRepo: sessionRepo,
		redisClient: redisClient,
		muxClient:   mux.NewClient(),
		broker:      broker,
		logger:      logger,
	}
}

// CreateStream creates a new stream
func (uc *StreamUseCase) CreateStream(ctx context.Context, hostID domain.UUID, title, description, thumbnailURL string) (*domain.Stream, error) {
	// Enforce 1 live stream per host
	existing, err := uc.streamRepo.GetLiveByHost(ctx, hostID)
	if err != nil && err != domain.ErrNotFound && err.Error() != "no rows in result set" {
		return nil, err
	}
	if existing != nil {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "host already has a live stream", nil)
	}

	roomID := domain.NewUUID()
	stream := &domain.Stream{
		ID:           domain.NewUUID(),
		HostID:       hostID,
		Title:        title,
		Description:  description,
		ThumbnailURL: thumbnailURL,
		Status:       domain.StreamStatusPreparing,
		RoomID:       roomID,
	}

	// Integrate with Mux
	muxStream, err := uc.muxClient.CreateLiveStream()
	if err != nil {
		uc.logger.Error("Failed to create Mux live stream", zap.Error(err))
		// Fallback or error? Let's error since user wants Mux
		return nil, err
	}

	stream.StreamKey = muxStream.Data.StreamKey
	if len(muxStream.Data.PlaybackIDs) > 0 {
		stream.PlaybackID = muxStream.Data.PlaybackIDs[0].ID
	}

	if err := uc.streamRepo.Create(ctx, stream); err != nil {
		return nil, err
	}

	return stream, nil
}

// StartStream marks a stream as live
func (uc *StreamUseCase) StartStream(ctx context.Context, streamID domain.UUID) error {
	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if stream.Status != domain.StreamStatusPreparing {
		return domain.NewDomainError(domain.ErrCodeValidation, "stream is not in preparing state", nil)
	}

	now := time.Now()
	stream.Status = domain.StreamStatusLive
	stream.StartedAt = &now

	if err := uc.streamRepo.Update(ctx, stream); err != nil {
		return err
	}

	// Initialize viewer count in Redis
	key := fmt.Sprintf("stream:viewer_count:%s", stream.RoomID.String())
	uc.redisClient.GetClient().Set(ctx, key, 0, 24*time.Hour)

	// Update stream status in Redis
	statusKey := fmt.Sprintf("stream:status:%s", stream.RoomID.String())
	uc.redisClient.GetClient().Set(ctx, statusKey, domain.StreamStatusLive, 24*time.Hour)

	// Broadcast stream start event
	// TODO: Broadcast to followers
	uc.logger.Info("Stream started", zap.String("stream_id", streamID.String()))

	return nil
}

// EndStream ends a live stream
func (uc *StreamUseCase) EndStream(ctx context.Context, streamID domain.UUID) error {
	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if stream.Status != domain.StreamStatusLive {
		return domain.NewDomainError(domain.ErrCodeValidation, "stream is not live", nil)
	}

	now := time.Now()
	stream.Status = domain.StreamStatusEnded
	stream.EndedAt = &now

	if stream.StartedAt != nil {
		stream.TotalDuration = int(now.Sub(*stream.StartedAt).Seconds())
	}

	// Fetch peak viewers from redis if we were tracking it, or just use current count
	// Usually peak is updated during viewer join, let's just get the final count for now
	key := fmt.Sprintf("stream:viewer_count:%s", stream.RoomID.String())
	countStr, _ := uc.redisClient.GetClient().Get(ctx, key).Result()
	var viewerCount int
	fmt.Sscanf(countStr, "%d", &viewerCount)
	// We might want to track peak viewer separately, but for now fallback to current
	if viewerCount > stream.ViewerPeak {
		stream.ViewerPeak = viewerCount
	}

	if err := uc.streamRepo.Update(ctx, stream); err != nil {
		return err
	}

	// Cleanup Redis
	uc.redisClient.GetClient().Del(ctx, key)
	statusKey := fmt.Sprintf("stream:status:%s", stream.RoomID.String())
	uc.redisClient.GetClient().Del(ctx, statusKey)

	uc.logger.Info("Stream ended", zap.String("stream_id", streamID.String()))

	return nil
}

// GetLiveStreams returns a list of live streams
func (uc *StreamUseCase) GetLiveStreams(ctx context.Context, limit, offset int) ([]*domain.Stream, error) {
	streams, err := uc.streamRepo.ListLive(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Attach viewer counts from Redis
	for _, stream := range streams {
		key := fmt.Sprintf("stream:viewer_count:%s", stream.RoomID.String())
		countStr, _ := uc.redisClient.GetClient().Get(ctx, key).Result()
		var count int
		fmt.Sscanf(countStr, "%d", &count)
		stream.ViewerPeak = count // temporary store current count in ViewerPeak or another field
	}

	return streams, nil
}

// GetStreamByID returns stream details by ID
func (uc *StreamUseCase) GetStreamByID(ctx context.Context, streamID domain.UUID) (*domain.Stream, error) {
	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	// Attach current viewer count
	key := fmt.Sprintf("stream:viewer_count:%s", stream.RoomID.String())
	countStr, _ := uc.redisClient.GetClient().Get(ctx, key).Result()
	var count int
	fmt.Sscanf(countStr, "%d", &count)
	stream.Viewers = count
	
	if stream.PlaybackID != "" {
		stream.MuxPlaybackURL = uc.muxClient.GetPlaybackURL(stream.PlaybackID)
	}

	return stream, nil
}

// JoinStream is called when a viewer joins the stream
func (uc *StreamUseCase) JoinStream(ctx context.Context, roomID, viewerID domain.UUID, ipAddress string) error {
	stream, err := uc.streamRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return err
	}

	if stream.Status != domain.StreamStatusLive {
		return domain.NewDomainError(domain.ErrCodeValidation, "stream is not live", nil)
	}

	session := &domain.StreamSession{
		ID:        domain.NewUUID(),
		StreamID:  stream.ID,
		ViewerID:  viewerID,
		IPAddress: ipAddress,
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return err
	}

	// Increment Redis viewer count
	key := fmt.Sprintf("stream:viewer_count:%s", roomID.String())
	uc.redisClient.GetClient().Incr(ctx, key)

	return nil
}

// LeaveStream is called when a viewer leaves the stream
func (uc *StreamUseCase) LeaveStream(ctx context.Context, roomID, viewerID domain.UUID) error {
	stream, err := uc.streamRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return err
	}

	session, err := uc.sessionRepo.GetActiveSession(ctx, stream.ID, viewerID)
	if err != nil {
		return err // Not found active session
	}

	now := time.Now()
	session.LeftAt = &now
	session.Duration = int(now.Sub(session.JoinedAt).Seconds())

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return err
	}

	// Decrement Redis viewer count
	key := fmt.Sprintf("stream:viewer_count:%s", roomID.String())
	uc.redisClient.GetClient().Decr(ctx, key)

	return nil
}
