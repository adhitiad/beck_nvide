package domain

import (
	"context"
	"time"
)

// VODAccessKey mewakili token akses jangka pendek untuk memutar VOD
type VODAccessKey struct {
	ID          UUID      `json:"id"`
	VODID       UUID      `json:"vod_id"`
	UserID      UUID      `json:"user_id"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// VODDRMKey menyimpan kunci enkripsi AES-128 untuk VOD
type VODDRMKey struct {
	VODID     UUID      `json:"vod_id"`
	KeyValue  []byte    `json:"key_value"`
	CreatedAt time.Time `json:"created_at"`
}

// DRMRepository mendefinisikan operasi database untuk hak akses dan kunci DRM
type DRMRepository interface {
	SaveAccessKey(ctx context.Context, key *VODAccessKey) error
	GetAccessKey(ctx context.Context, token string) (*VODAccessKey, error)
	SaveDRMKey(ctx context.Context, vodID UUID, keyValue []byte) error
	GetDRMKey(ctx context.Context, vodID UUID) ([]byte, error)
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// DRMUseCaseInterface mendefinisikan kontrak logika bisnis DRM
type DRMUseCaseInterface interface {
	GeneratePlaybackToken(ctx context.Context, userID, vodID UUID) (*VODAccessKey, string, error)
	ValidateToken(ctx context.Context, token string) (*VODAccessKey, error)
	GetVODDRMKey(ctx context.Context, vodID UUID) ([]byte, error)
	GenerateDRMKeysForVOD(ctx context.Context, vodID UUID) ([]byte, error)
	WatermarkSegment(ctx context.Context, segmentPath string, userID UUID) (string, error) // Returns path to watermarked segment
}
