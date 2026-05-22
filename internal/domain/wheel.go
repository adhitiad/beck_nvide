package domain

import (
	"context"
	"time"
)

// Wheel prize types
const (
	WheelPrizeCoin        = "coin"
	WheelPrizeGift        = "gift"
	WheelPrizeVoucher     = "voucher"
	WheelPrizeEntryEffect = "entry_effect"
	WheelPrizeNothing     = "nothing"
)

// WheelPrize represents a prize on the wheel of fortune
type WheelPrize struct {
	ID          UUID      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // coin, gift, voucher, entry_effect, nothing
	Value       int64     `json:"value"`
	ItemID      *UUID     `json:"item_id,omitempty"`
	IconURL     string    `json:"icon_url"`
	Probability float64   `json:"probability"` // 0.0 - 1.0
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// WheelSpin represents a record of a user spinning the wheel
type WheelSpin struct {
	ID        UUID      `json:"id"`
	UserID    UUID      `json:"user_id"`
	PrizeID   UUID      `json:"prize_id"`
	Cost      int64     `json:"cost"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	Prize *WheelPrize `json:"prize,omitempty"`
}

// WheelRepository defines the contract for wheel of fortune data access
type WheelRepository interface {
	// Prizes
	CreatePrize(ctx context.Context, prize *WheelPrize) error
	ListActivePrizes(ctx context.Context) ([]*WheelPrize, error)
	GetPrizeByID(ctx context.Context, id UUID) (*WheelPrize, error)
	UpdatePrize(ctx context.Context, prize *WheelPrize) error

	// Spins
	RecordSpin(ctx context.Context, spin *WheelSpin) error
	GetUserSpinsToday(ctx context.Context, userID UUID) (int, error)
	GetUserSpinHistory(ctx context.Context, userID UUID, limit, offset int) ([]*WheelSpin, error)
}
