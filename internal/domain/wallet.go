package domain

import (
	"context"
	"time"
)

// Transaction types
const (
	TxTypeDeposit         = "deposit"
	TxTypeWithdrawal      = "withdrawal"
	TxTypeGiftSent        = "gift_sent"
	TxTypeGiftReceived    = "gift_received"
	TxTypeAgencyCommission = "agency_commission"
	TxTypeHostEarning     = "host_earning"
	TxTypePlatformFee     = "platform_fee"
	TxTypePaidChatUnlock  = "paid_chat_unlock"
	TxTypePaidCallTick    = "paid_call_tick"
)

// Transaction statuses
const (
	TxStatusPending   = "pending"
	TxStatusSuccess   = "success"
	TxStatusFailed    = "failed"
	TxStatusCancelled = "cancelled"
	TxStatusRefunded  = "refunded"
)

// Wallet represents a user wallet
type Wallet struct {
	ID            UUID      `json:"id"`
	UserID        UUID      `json:"user_id"`
	Balance       int64     `json:"balance"`        // in IDR
	FrozenBalance int64     `json:"frozen_balance"`
	Currency      string    `json:"currency"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Transaction represents an immutable financial record
type Transaction struct {
	ID            UUID      `json:"id"`
	UserID        UUID      `json:"user_id"`
	Type          string    `json:"type"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	ReferenceID   string    `json:"reference_id"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	Metadata      string    `json:"metadata,omitempty"` // JSON
	CreatedAt     time.Time `json:"created_at"`
}

// Repositories
type WalletRepository interface {
	GetByUserID(ctx context.Context, userID UUID) (*Wallet, error)
	Create(ctx context.Context, wallet *Wallet) error
	// CreditBalance adds to balance within a DB transaction (row-level lock)
	CreditBalance(ctx context.Context, userID UUID, amount int64) error
	// DebitBalance subtracts from balance within a DB transaction (row-level lock)
	DebitBalance(ctx context.Context, userID UUID, amount int64) error
	FreezeBalance(ctx context.Context, userID UUID, amount int64) error
	UnfreezeBalance(ctx context.Context, userID UUID, amount int64) error
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByID(ctx context.Context, id UUID) (*Transaction, error)
	GetByReferenceID(ctx context.Context, refID string) (*Transaction, error)
	ListByUser(ctx context.Context, userID UUID, txType string, limit, offset int) ([]*Transaction, error)
	UpdateStatus(ctx context.Context, id UUID, status string) error
}
