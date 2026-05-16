package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Withdrawal statuses
const (
	WithdrawalPending    = "pending"
	WithdrawalApproved   = "approved"
	WithdrawalRejected   = "rejected"
	WithdrawalProcessing = "processing"
	WithdrawalCompleted  = "completed"
	WithdrawalFailed     = "failed"
)

// FeeRule represents a configurable fee
type FeeRule struct {
	ID        UUID      `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	FeeType   string    `json:"fee_type" db:"fee_type"` // "percentage", "fixed"
	Value     float64   `json:"value" db:"value"`
	AppliesTo string    `json:"applies_to" db:"applies_to"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	Priority  int       `json:"priority" db:"priority"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Withdrawal represents a user cashout request
type Withdrawal struct {
	ID              UUID            `json:"id" db:"id"`
	UserID          UUID            `json:"user_id" db:"user_id"`
	AmountRequested int64           `json:"amount_requested" db:"amount_requested"`
	GrossAmount     int64           `json:"gross_amount" db:"gross_amount"`
	FeePlatform     int64           `json:"fee_platform" db:"fee_platform"`
	FeeProcessing   int64           `json:"fee_processing" db:"fee_processing"`
	FeeTax          int64           `json:"fee_tax" db:"fee_tax"`
	FeeAgency       int64           `json:"fee_agency" db:"fee_agency"`
	TotalFee        int64           `json:"total_fee" db:"total_fee"`
	NetAmount       int64           `json:"net_amount" db:"net_amount"`
	AgencyID        *UUID           `json:"agency_id,omitempty" db:"agency_id"`
	Status          string          `json:"status" db:"status"`
	PaymentMethod   string          `json:"payment_method" db:"payment_method"`
	BankAccountInfo json.RawMessage `json:"bank_account_info" db:"bank_account_info"`
	TxReference     string          `json:"tx_reference,omitempty" db:"tx_reference"`
	ApprovedBy      *UUID           `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty" db:"approved_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// WithdrawalFeeAudit represents an immutable fee log
type WithdrawalFeeAudit struct {
	ID             UUID      `json:"id" db:"id"`
	WithdrawalID   UUID      `json:"withdrawal_id" db:"withdrawal_id"`
	FeeName        string    `json:"fee_name" db:"fee_name"`
	FeePercentage  float64   `json:"fee_percentage" db:"fee_percentage"`
	FeeAmount      int64     `json:"fee_amount" db:"fee_amount"`
	CalculatedFrom int64     `json:"calculated_from" db:"calculated_from"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// WithdrawalRepository defines data access for withdrawals
type WithdrawalRepository interface {
	Create(ctx context.Context, w *Withdrawal) error
	GetByID(ctx context.Context, id UUID) (*Withdrawal, error)
	List(ctx context.Context, userID *UUID, status string, limit, offset int) ([]*Withdrawal, error)
	UpdateStatus(ctx context.Context, id UUID, status string, adminID *UUID) error
	
	// Fee Rules
	GetActiveFeeRules(ctx context.Context) ([]*FeeRule, error)
	CreateFeeAudit(ctx context.Context, audit *WithdrawalFeeAudit) error
}

// WithdrawalUsecase defines business logic for withdrawals
type WithdrawalUsecase interface {
	RequestWithdrawal(ctx context.Context, userID UUID, amount int64, method string, bankInfo map[string]interface{}) (*Withdrawal, error)
	ApproveWithdrawal(ctx context.Context, adminID, withdrawalID UUID) error
	RejectWithdrawal(ctx context.Context, adminID, withdrawalID UUID, reason string) error
	GetHistory(ctx context.Context, userID UUID, limit, offset int) ([]*Withdrawal, error)
	CalculatePreview(ctx context.Context, userID UUID, amount int64) (map[string]interface{}, error)
}
