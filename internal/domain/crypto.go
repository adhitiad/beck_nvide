package domain

import (
	"context"
	"time"
)

// Crypto constants
const (
	ChainSOL       = "SOL"
	ChainBTC       = "BTC"
	ChainUSDT_ERC20 = "USDT_ERC20"
	ChainUSDT_TRC20 = "USDT_TRC20"
	ChainUSDT_BEP20 = "USDT_BEP20"

	CryptoStatusPending   = "pending"
	CryptoStatusConfirming = "confirming"
	CryptoStatusSuccess    = "success"
	CryptoStatusFailed     = "failed"
	CryptoStatusCancelled  = "cancelled"
)

// CryptoMasterWallet represents a hot wallet owned by the platform
type CryptoMasterWallet struct {
	ID                   UUID      `json:"id"`
	Chain                string    `json:"chain"`
	PublicKey            string    `json:"public_key"`
	EncryptedPrivateKey  string    `json:"-"`
	DerivationPath       string    `json:"derivation_path"`
	Balance              float64   `json:"balance"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CryptoDepositAddress represents a unique address generated for a user
type CryptoDepositAddress struct {
	ID              UUID      `json:"id"`
	UserID          UUID      `json:"user_id"`
	Chain           string    `json:"chain"`
	Address         string    `json:"address"`
	DerivationIndex int       `json:"derivation_index"`
	MasterWalletID  UUID      `json:"master_wallet_id"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

// CryptoTransaction represents a crypto deposit or withdrawal
type CryptoTransaction struct {
	ID                    UUID       `json:"id"`
	UserID                UUID       `json:"user_id"`
	Type                  string     `json:"type"` // deposit, withdrawal
	Chain                 string     `json:"chain"`
	Asset                 string     `json:"asset"`
	AmountCrypto          float64    `json:"amount_crypto"`
	AmountIDR             float64    `json:"amount_idr"`
	ExchangeRate          float64    `json:"exchange_rate"`
	TxHash                string     `json:"tx_hash"`
	FromAddress           string     `json:"from_address"`
	ToAddress             string     `json:"to_address"`
	Confirmations         int        `json:"confirmations"`
	RequiredConfirmations int        `json:"required_confirmations"`
	Status                string     `json:"status"`
	FeeCrypto             float64    `json:"fee_crypto"`
	FeeIDR                float64    `json:"fee_idr"`
	Metadata              Metadata   `json:"metadata"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
}

// CryptoExchangeRate represents the market rate for a crypto asset
type CryptoExchangeRate struct {
	ID        UUID      `json:"id"`
	Asset     string    `json:"asset"`
	Currency  string    `json:"currency"`
	Rate      float64   `json:"rate"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}

// CryptoWithdrawalWhitelist represents a user's verified withdrawal address
type CryptoWithdrawalWhitelist struct {
	ID         UUID      `json:"id"`
	UserID     UUID      `json:"user_id"`
	Chain      string    `json:"chain"`
	Address    string    `json:"address"`
	Label      string    `json:"label"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
}

// CryptoRepository interfaces
type CryptoRepository interface {
	// Master Wallet
	GetMasterWalletByChain(ctx context.Context, chain string) (*CryptoMasterWallet, error)
	UpdateMasterWalletBalance(ctx context.Context, id UUID, balance float64) error

	// Deposit Address
	GetDepositAddress(ctx context.Context, userID UUID, chain string) (*CryptoDepositAddress, error)
	CreateDepositAddress(ctx context.Context, addr *CryptoDepositAddress) error
	GetLastDerivationIndex(ctx context.Context, masterWalletID UUID) (int, error)

	// Transactions
	CreateTransaction(ctx context.Context, tx *CryptoTransaction) error
	GetTransactionByHash(ctx context.Context, hash string) (*CryptoTransaction, error)
	UpdateTransactionStatus(ctx context.Context, id UUID, status string, confirmations int) error
	ListPendingTransactions(ctx context.Context) ([]*CryptoTransaction, error)

	// Exchange Rate
	GetExchangeRate(ctx context.Context, asset, currency string) (*CryptoExchangeRate, error)
	UpdateExchangeRate(ctx context.Context, rate *CryptoExchangeRate) error

	// Whitelist
	GetWhitelist(ctx context.Context, userID UUID, chain string) ([]*CryptoWithdrawalWhitelist, error)
	AddToWhitelist(ctx context.Context, entry *CryptoWithdrawalWhitelist) error
	DeleteFromWhitelist(ctx context.Context, userID UUID, id UUID) error
}
