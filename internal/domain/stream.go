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

	// Bigo/Mango Room Config & Modes
	RoomMode            string  `json:"room_mode"`
	RoomPasswordHash   string  `json:"-"`
	EntryFeeIDR        float64 `json:"entry_fee_idr"`
	MinLevelToEnter    int     `json:"min_level_to_enter"`
	Category           string  `json:"category"`
	Tags               string  `json:"tags"`
	MaxResolution      string  `json:"max_resolution"`
	IsScreenShare      bool    `json:"is_screen_share"`
	IsCoHostEnabled    bool    `json:"is_co_host_enabled"`
	MaxCoHosts         int     `json:"max_co_hosts"`
	ViewerCount        int     `json:"viewer_count"`
	TotalGiftValueIDR  float64 `json:"total_gift_value_idr"`
	LikeCount          int     `json:"like_count"`
	ShareCount         int     `json:"share_count"`
	CurrentPKID        *UUID   `json:"current_pk_id,omitempty"`
	IsPKEligible       bool    `json:"is_pk_eligible"`
	ChatMode           string  `json:"chat_mode"`
	ChatSlowModeSeconds int    `json:"chat_slow_mode_seconds"`
	CountryCode        string  `json:"country_code"`
	Language           string  `json:"language"`

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

type CreateStreamInput struct {
	Title               string
	Description         string
	ThumbnailURL        string
	RoomMode            string
	RoomPassword        string
	EntryFeeIDR         float64
	MinLevelToEnter     int
	Category            string
	Tags                string
	MaxResolution       string
	IsScreenShare       bool
	IsCoHostEnabled     bool
	MaxCoHosts          int
	ChatMode            string
	ChatSlowModeSeconds int
	CountryCode         string
	Language            string
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
