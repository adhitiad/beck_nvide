package domain

import (
	"context"
	"time"
)

// Gift represents a gift in the catalog
type Gift struct {
	ID           UUID      `json:"id"`
	Name         string    `json:"name"`
	IconURL      string    `json:"icon_url"`
	Price        int64     `json:"price"` // in IDR
	Currency     string    `json:"currency"`
	AnimationURL string    `json:"animation_url"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// GiftTransaction records a gift send event
type GiftTransaction struct {
	ID               UUID      `json:"id"`
	StreamID         *UUID     `json:"stream_id,omitempty"`
	SenderID         UUID      `json:"sender_id"`
	ReceiverID       UUID      `json:"receiver_id"`
	GiftID           UUID      `json:"gift_id"`
	Quantity         int       `json:"quantity"`
	TotalPrice       int64     `json:"total_price"`
	AgencyID         *UUID     `json:"agency_id,omitempty"`
	AgencyCommission int64     `json:"agency_commission"`
	HostEarning      int64     `json:"host_earning"`
	PlatformFee      int64     `json:"platform_fee"`
	CreatedAt        time.Time `json:"created_at"`

	// Relations
	Gift   *Gift `json:"gift,omitempty"`
	Sender *User `json:"sender,omitempty"`
}

// DuitkuPayment represents a Duitku payment record
type DuitkuPayment struct {
	ID              UUID       `json:"id"`
	TransactionID   UUID       `json:"transaction_id"`
	MerchantOrderID string     `json:"merchant_order_id"`
	DuitkuReference string     `json:"duitku_reference"`
	PaymentURL      string     `json:"payment_url"`
	VANumber        string     `json:"va_number,omitempty"`
	PaymentMethod   string     `json:"payment_method"`
	Status          string     `json:"status"`
	Amount          int64      `json:"amount"`
	ExpiryAt        *time.Time `json:"expiry_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Repositories
type GiftRepository interface {
	Create(ctx context.Context, gift *Gift) error
	GetByID(ctx context.Context, id UUID) (*Gift, error)
	ListActive(ctx context.Context) ([]*Gift, error)
	Update(ctx context.Context, gift *Gift) error
}

type GiftTransactionRepository interface {
	Create(ctx context.Context, gtx *GiftTransaction) error
	ListByStream(ctx context.Context, streamID UUID, limit, offset int) ([]*GiftTransaction, error)
	ListBySender(ctx context.Context, senderID UUID, limit, offset int) ([]*GiftTransaction, error)
}

type DuitkuPaymentRepository interface {
	Create(ctx context.Context, dp *DuitkuPayment) error
	GetByMerchantOrderID(ctx context.Context, merchantOrderID string) (*DuitkuPayment, error)
	Update(ctx context.Context, dp *DuitkuPayment) error
}
