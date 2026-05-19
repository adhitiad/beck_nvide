package domain

import (
	"context"
	"time"
)

// StreamClip mewakili rekaman klip sorotan (highlight) otomatis dari stream
type StreamClip struct {
	ID        UUID      `json:"id"`
	StreamID  UUID      `json:"stream_id"`
	Title     string    `json:"title"`
	ClipURL   string    `json:"clip_url"`
	Duration  int       `json:"duration"` // dalam detik
	Score     float64   `json:"score"`    // Skor lonjakan interaksi yang memicu klip
	CreatedAt time.Time `json:"created_at"`
}

// ClipRepository mendefinisikan operasi DB untuk manajemen klip AI
type ClipRepository interface {
	Create(ctx context.Context, clip *StreamClip) error
	GetByID(ctx context.Context, id UUID) (*StreamClip, error)
	ListByStream(ctx context.Context, streamID UUID) ([]*StreamClip, error)
	ListTrending(ctx context.Context, limit, offset int) ([]*StreamClip, error)
	Delete(ctx context.Context, id UUID) error
}

// ClipUseCaseInterface mendefinisikan logika bisnis untuk deteksi sorotan dan pembuatan klip otomatis
type ClipUseCaseInterface interface {
	// RegisterInteractionEvent mendaftarkan aktivitas real-time untuk memantau lonjakan interaksi
	RegisterInteractionEvent(ctx context.Context, streamID UUID, eventType string, weight float64) error
	// CheckHighlightSpike menganalisis apakah terjadi lonjakan interaksi untuk memicu pemotongan klip otomatis
	CheckHighlightSpike(ctx context.Context, streamID UUID) (bool, float64, error)
	// GenerateClip memotong rekaman HLS stream pada rentang waktu tertentu secara asinkron
	GenerateClip(ctx context.Context, streamID UUID, startTimeOffsetSec, durationSec int, title string, triggerScore float64) (*StreamClip, error)
	// GetStreamClips mendapatkan daftar klip untuk stream tertentu
	GetStreamClips(ctx context.Context, streamID UUID) ([]*StreamClip, error)
	// GetTrendingClips mendapatkan daftar klip paling tren secara global
	GetTrendingClips(ctx context.Context, limit, offset int) ([]*StreamClip, error)
}
