package domain

import (
	"context"
	"time"
)

const (
	PayoutTypeBankTransfer = "bank_transfer"
	PayoutTypeEwallet      = "ewallet"
	PayoutTypeCrypto       = "crypto"
)

const (
	EwalletGopay     = "gopay"
	EwalletOVO       = "ovo"
	EwalletDANA      = "dana"
	EwalletShopeePay = "shopeepay"
	EwalletLinkAja   = "linkaja"
)

const (
	CryptoNetworkSolana = "solana"
	CryptoNetworkBTC    = "bitcoin"
	CryptoNetworkBSC    = "bsc"
)

type PayoutMethod struct {
	ID                   UUID      `json:"id" db:"id"`
	UserID               UUID      `json:"user_id" db:"user_id"`
	Type                 string    `json:"type" db:"type"`
	IsPrimary            bool      `json:"is_primary" db:"is_primary"`
	BankName             *string   `json:"bank_name,omitempty" db:"bank_name"`
	AccountNumber        *string   `json:"account_number,omitempty" db:"account_number"`
	AccountHolderName    *string   `json:"account_holder_name,omitempty" db:"account_holder_name"`
	EwalletProvider      *string   `json:"ewallet_provider,omitempty" db:"ewallet_provider"`
	EwalletPhoneNumber   *string   `json:"ewallet_phone_number,omitempty" db:"ewallet_phone_number"`
	IsVerified           bool      `json:"is_verified" db:"is_verified"`
	MicroDepositRequired bool      `json:"micro_deposit_required" db:"micro_deposit_required"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

type CryptoPayoutAddress struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Network   string    `json:"network" db:"network"`
	Address   string    `json:"address" db:"address"`
	Label     *string   `json:"label,omitempty" db:"label"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type PayoutMethodRepository interface {
	ListByUserID(ctx context.Context, userID UUID) ([]*PayoutMethod, error)
	GetByID(ctx context.Context, id UUID) (*PayoutMethod, error)
	GetByIDAndUserID(ctx context.Context, id, userID UUID) (*PayoutMethod, error)
	GetPrimaryByUserID(ctx context.Context, userID UUID) (*PayoutMethod, error)
	CountByUserIDAndType(ctx context.Context, userID UUID, payoutType string) (int, error)
	Create(ctx context.Context, method *PayoutMethod) error
	Update(ctx context.Context, method *PayoutMethod) error
	Delete(ctx context.Context, id, userID UUID) error
	UnsetPrimaryByUserID(ctx context.Context, userID UUID) error
	SetPrimary(ctx context.Context, id, userID UUID) error
}

type CryptoPayoutAddressRepository interface {
	ListByUserID(ctx context.Context, userID UUID) ([]*CryptoPayoutAddress, error)
	GetByIDAndUserID(ctx context.Context, id, userID UUID) (*CryptoPayoutAddress, error)
	CountByUserID(ctx context.Context, userID UUID) (int, error)
	Create(ctx context.Context, address *CryptoPayoutAddress) error
	Delete(ctx context.Context, id, userID UUID) error
}

type PayoutUsecase interface {
	ListPayoutMethods(ctx context.Context, userID UUID) ([]*PayoutMethod, error)
	CreatePayoutMethod(ctx context.Context, userID UUID, req *CreatePayoutMethodRequest) (*PayoutMethod, error)
	UpdatePayoutMethod(ctx context.Context, userID, methodID UUID, req *UpdatePayoutMethodRequest) (*PayoutMethod, error)
	DeletePayoutMethod(ctx context.Context, userID, methodID UUID) error
	SetPrimaryPayoutMethod(ctx context.Context, userID, methodID UUID) error

	ListCryptoPayoutAddresses(ctx context.Context, userID UUID) ([]*CryptoPayoutAddress, error)
	CreateCryptoPayoutAddress(ctx context.Context, userID UUID, req *CreateCryptoPayoutAddressRequest) (*CryptoPayoutAddress, error)
	DeleteCryptoPayoutAddress(ctx context.Context, userID, id UUID) error

	ResolveWithdrawalTarget(ctx context.Context, userID UUID) (string, map[string]interface{}, error)
}

type CreatePayoutMethodRequest struct {
	Type               string  `json:"type"`
	IsPrimary          bool    `json:"is_primary"`
	BankName           *string `json:"bank_name"`
	AccountNumber      *string `json:"account_number"`
	AccountHolderName  *string `json:"account_holder_name"`
	EwalletProvider    *string `json:"provider"`
	EwalletPhoneNumber *string `json:"phone_number"`
}

type UpdatePayoutMethodRequest struct {
	IsPrimary          *bool   `json:"is_primary"`
	BankName           *string `json:"bank_name"`
	AccountNumber      *string `json:"account_number"`
	AccountHolderName  *string `json:"account_holder_name"`
	EwalletProvider    *string `json:"provider"`
	EwalletPhoneNumber *string `json:"phone_number"`
}

type CreateCryptoPayoutAddressRequest struct {
	Network string  `json:"network"`
	Address string  `json:"address"`
	Label   *string `json:"label"`
}
