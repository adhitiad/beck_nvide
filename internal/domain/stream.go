package domain

import (
	"context"
	"time"
)

// Stream status constants
const (
	StreamStatusPreparing = "preparing"
	StreamStatusLive      = "live"
	StreamStatusEnded     = "ended"
	StreamStatusArchived  = "archived"
)

// Stream represents a live stream
type Stream struct {
	ID            UUID      `json:"id"`
	HostID        UUID      `json:"host_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ThumbnailURL  string    `json:"thumbnail_url"`
	Status        string    `json:"status"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	ViewerPeak    int       `json:"viewer_peak"`
	TotalDuration int       `json:"total_duration"`
	RoomID        UUID      `json:"room_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Mux Fields
	StreamKey    string `json:"stream_key,omitempty"`    // For RTMP push
	PlaybackID   string `json:"playback_id,omitempty"`   // For HLS playback
	MuxAssetID   string `json:"mux_asset_id,omitempty"`

	// Virtual fields
	Viewers      int    `json:"viewers" gorm:"-"`        // Current viewers from Redis
	MuxPlaybackURL string `json:"mux_playback_url,omitempty" gorm:"-"`

	// Relations
	Host *User `json:"host,omitempty"`
}

// StreamSession represents a viewer's session in a stream
type StreamSession struct {
	ID        UUID       `json:"id"`
	StreamID  UUID       `json:"stream_id"`
	ViewerID  UUID       `json:"viewer_id"`
	JoinedAt  time.Time  `json:"joined_at"`
	LeftAt    *time.Time `json:"left_at"`
	Duration  int        `json:"duration"`
	IPAddress string     `json:"ip_address"`
}

// StreamSignaling represents a WebRTC signaling message stored in DB (optional fallback)
type StreamSignaling struct {
	ID         UUID      `json:"id"`
	StreamID   UUID      `json:"stream_id"`
	PeerID     UUID      `json:"peer_id"`
	SignalType string    `json:"signal_type"`
	Data       string    `json:"data"`
	CreatedAt  time.Time `json:"created_at"`
}

// StreamRepository defines operations for streams
type StreamRepository interface {
	Create(ctx context.Context, stream *Stream) error
	Update(ctx context.Context, stream *Stream) error
	GetByID(ctx context.Context, id UUID) (*Stream, error)
	GetByRoomID(ctx context.Context, roomID UUID) (*Stream, error)
	GetLiveByHost(ctx context.Context, hostID UUID) (*Stream, error)
	ListLive(ctx context.Context, limit, offset int) ([]*Stream, error)
}

// StreamSessionRepository defines operations for stream sessions
type StreamSessionRepository interface {
	Create(ctx context.Context, session *StreamSession) error
	Update(ctx context.Context, session *StreamSession) error
	GetActiveSession(ctx context.Context, streamID, viewerID UUID) (*StreamSession, error)
}

// StreamSignalingRepository defines operations for signaling (if needed)
type StreamSignalingRepository interface {
	Create(ctx context.Context, sig *StreamSignaling) error
	GetByStreamAndPeer(ctx context.Context, streamID, peerID UUID) ([]*StreamSignaling, error)
}
