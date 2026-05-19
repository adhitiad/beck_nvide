package domain

import (
	"context"
	"time"
)

// CreatorToken merepresentasikan token kustom yang diterbitkan oleh host/kreator
type CreatorToken struct {
	ID          UUID      `json:"id"`
	HostID      UUID      `json:"host_id"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	TotalSupply int64     `json:"total_supply"`
	MaxSupply   int64     `json:"max_supply"`
	BasePrice   int64     `json:"base_price"` // dalam IDR
	Slope       int64     `json:"slope"`      // konstanta kenaikan harga bonding curve
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserToken merepresentasikan saldo token kreator yang dimiliki oleh pengguna
type UserToken struct {
	ID        UUID      `json:"id"`
	UserID    UUID      `json:"user_id"`
	TokenID   UUID      `json:"token_id"`
	Balance   int64     `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relasi virtual
	Token *CreatorToken `json:"token,omitempty"`
}

// CreatorTokenRepository mendefinisikan kontrak akses data untuk Token Kreator
type CreatorTokenRepository interface {
	CreateToken(ctx context.Context, token *CreatorToken) error
	GetTokenByHostID(ctx context.Context, hostID UUID) (*CreatorToken, error)
	GetTokenByID(ctx context.Context, id UUID) (*CreatorToken, error)
	GetTokenBySymbol(ctx context.Context, symbol string) (*CreatorToken, error)
	GetUserToken(ctx context.Context, userID, tokenID UUID) (*UserToken, error)
	ListUserTokens(ctx context.Context, userID UUID) ([]*UserToken, error)
	UpdateTokenSupply(ctx context.Context, tokenID UUID, newSupply int64) error
	UpdateUserTokenBalance(ctx context.Context, userID, tokenID UUID, amount int64) error
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CreatorTokenUseCaseInterface mendefinisikan kontrak logika bisnis untuk Token Kreator
type CreatorTokenUseCaseInterface interface {
	IssueToken(ctx context.Context, hostID UUID, name, symbol string, maxSupply, basePrice, slope int64) (*CreatorToken, error)
	GetTokenInfo(ctx context.Context, hostID UUID) (*CreatorToken, error)
	BuyToken(ctx context.Context, userID, tokenID UUID, amount int64) (*UserToken, error)
	GetUserBalances(ctx context.Context, userID UUID) ([]*UserToken, error)
}
