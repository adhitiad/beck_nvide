package domain

import (
	"context"
	"time"
)

// Prediction mewakili sebuah pertanyaan pasar prediksi pada sebuah stream
type Prediction struct {
	ID              UUID      `json:"id"`
	StreamID        UUID      `json:"stream_id"`
	Question        string    `json:"question"`
	Status          string    `json:"status"`            // 'active', 'resolved', 'cancelled'
	ResolvedOutcome *string   `json:"resolved_outcome"`  // 'yes', 'no'
	TotalYesPool    int64     `json:"total_yes_pool"`    // Total taruhan 'yes' dalam IDR
	TotalNoPool     int64     `json:"total_no_pool"`     // Total taruhan 'no' dalam IDR
	CreatedAt       time.Time `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// PredictionBet mewakili taruhan seorang pengguna pada sebuah prediksi
type PredictionBet struct {
	ID             UUID      `json:"id"`
	PredictionID   UUID      `json:"prediction_id"`
	UserID         UUID      `json:"user_id"`
	Outcome        string    `json:"outcome"`          // 'yes', 'no'
	Amount         int64     `json:"amount"`           // Jumlah taruhan dalam IDR atau token kreator
	CurrencyType   string    `json:"currency_type"`    // 'wallet', 'token'
	CreatorTokenID *UUID     `json:"creator_token_id"` // Jika menggunakan token kustom host
	CreatedAt      time.Time `json:"created_at"`
}

// PredictionRepository mendefinisikan kontrak akses data untuk pasar prediksi
type PredictionRepository interface {
	Create(ctx context.Context, p *Prediction) error
	GetByID(ctx context.Context, id UUID) (*Prediction, error)
	GetActiveByStreamID(ctx context.Context, streamID UUID) ([]*Prediction, error)
	CreateBet(ctx context.Context, bet *PredictionBet) error
	GetBetsByPredictionID(ctx context.Context, predictionID UUID) ([]*PredictionBet, error)
	UpdatePools(ctx context.Context, id UUID, yesAmount, noAmount int64) error
	ResolvePrediction(ctx context.Context, id UUID, outcome string) error
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PredictionUseCaseInterface mendefinisikan kontrak logika bisnis untuk pasar prediksi
type PredictionUseCaseInterface interface {
	CreatePrediction(ctx context.Context, hostID, streamID UUID, question string) (*Prediction, error)
	GetActivePredictions(ctx context.Context, streamID UUID) ([]*Prediction, error)
	PlaceBet(ctx context.Context, userID, predictionID UUID, outcome string, amount int64, currencyType string, creatorTokenID *UUID) (*PredictionBet, error)
	ResolvePrediction(ctx context.Context, hostID, predictionID UUID, outcome string) (*Prediction, error)
}
