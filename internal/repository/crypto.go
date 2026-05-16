package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"nvide-live/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type cryptoRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewCryptoRepository(db *pgxpool.Pool, logger *zap.Logger) domain.CryptoRepository {
	return &cryptoRepository{db: db, logger: logger}
}

// Master Wallet
func (r *cryptoRepository) GetMasterWalletByChain(ctx context.Context, chain string) (*domain.CryptoMasterWallet, error) {
	query := `SELECT id, chain, public_key, encrypted_private_key, derivation_path, balance, status, created_at, updated_at 
	          FROM crypto_master_wallets WHERE chain = $1 AND status = 'active' LIMIT 1`
	
	var mw domain.CryptoMasterWallet
	err := r.db.QueryRow(ctx, query, chain).Scan(
		&mw.ID, &mw.Chain, &mw.PublicKey, &mw.EncryptedPrivateKey, &mw.DerivationPath, 
		&mw.Balance, &mw.Status, &mw.CreatedAt, &mw.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("master wallet for chain %s not found", chain)
		}
		return nil, err
	}
	return &mw, nil
}

func (r *cryptoRepository) UpdateMasterWalletBalance(ctx context.Context, id domain.UUID, balance float64) error {
	query := `UPDATE crypto_master_wallets SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, balance, id)
	return err
}

// Deposit Address
func (r *cryptoRepository) GetDepositAddress(ctx context.Context, userID domain.UUID, chain string) (*domain.CryptoDepositAddress, error) {
	query := `SELECT id, user_id, chain, address, derivation_index, master_wallet_id, is_active, created_at 
	          FROM crypto_deposit_addresses WHERE user_id = $1 AND chain = $2 AND is_active = true`
	
	var addr domain.CryptoDepositAddress
	err := r.db.QueryRow(ctx, query, userID, chain).Scan(
		&addr.ID, &addr.UserID, &addr.Chain, &addr.Address, &addr.DerivationIndex, 
		&addr.MasterWalletID, &addr.IsActive, &addr.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &addr, nil
}

func (r *cryptoRepository) CreateDepositAddress(ctx context.Context, addr *domain.CryptoDepositAddress) error {
	query := `INSERT INTO crypto_deposit_addresses (id, user_id, chain, address, derivation_index, master_wallet_id, is_active, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query, addr.ID, addr.UserID, addr.Chain, addr.Address, addr.DerivationIndex, addr.MasterWalletID, addr.IsActive, addr.CreatedAt)
	return err
}

func (r *cryptoRepository) GetLastDerivationIndex(ctx context.Context, masterWalletID domain.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(derivation_index), -1) FROM crypto_deposit_addresses WHERE master_wallet_id = $1`
	var lastIndex int
	err := r.db.QueryRow(ctx, query, masterWalletID).Scan(&lastIndex)
	return lastIndex, err
}

// Transactions
func (r *cryptoRepository) CreateTransaction(ctx context.Context, tx *domain.CryptoTransaction) error {
	metadata, _ := json.Marshal(tx.Metadata)
	query := `INSERT INTO crypto_transactions (id, user_id, type, chain, asset, amount_crypto, amount_idr, exchange_rate, tx_hash, from_address, to_address, confirmations, required_confirmations, status, fee_crypto, fee_idr, metadata, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`
	_, err := r.db.Exec(ctx, query, 
		tx.ID, tx.UserID, tx.Type, tx.Chain, tx.Asset, tx.AmountCrypto, tx.AmountIDR, tx.ExchangeRate, 
		tx.TxHash, tx.FromAddress, tx.ToAddress, tx.Confirmations, tx.RequiredConfirmations, 
		tx.Status, tx.FeeCrypto, tx.FeeIDR, metadata, tx.CreatedAt, tx.UpdatedAt,
	)
	return err
}


func (r *cryptoRepository) UpdateTransactionStatus(ctx context.Context, id domain.UUID, status string, confirmations int) error {
	query := `UPDATE crypto_transactions SET status = $1, confirmations = $2, updated_at = NOW()`
	if status == domain.CryptoStatusSuccess {
		query += `, completed_at = NOW()`
	}
	query += ` WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, confirmations, id)
	return err
}

func (r *cryptoRepository) ListPendingTransactions(ctx context.Context) ([]*domain.CryptoTransaction, error) {
	query := `SELECT id, user_id, type, chain, asset, amount_crypto, amount_idr, exchange_rate, tx_hash, from_address, to_address, confirmations, required_confirmations, status, created_at, updated_at 
	          FROM crypto_transactions WHERE status IN ('pending', 'confirming')`
	
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.CryptoTransaction
	for rows.Next() {
		var tx domain.CryptoTransaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.Type, &tx.Chain, &tx.Asset, &tx.AmountCrypto, &tx.AmountIDR, 
			&tx.ExchangeRate, &tx.TxHash, &tx.FromAddress, &tx.ToAddress, &tx.Confirmations, 
			&tx.RequiredConfirmations, &tx.Status, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, &tx)
	}
	return txs, nil
}

// Exchange Rate
func (r *cryptoRepository) GetExchangeRate(ctx context.Context, asset, currency string) (*domain.CryptoExchangeRate, error) {
	query := `SELECT id, asset, currency, rate, source, fetched_at FROM crypto_exchange_rates WHERE asset = $1 AND currency = $2`
	var rate domain.CryptoExchangeRate
	err := r.db.QueryRow(ctx, query, asset, currency).Scan(&rate.ID, &rate.Asset, &rate.Currency, &rate.Rate, &rate.Source, &rate.FetchedAt)
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *cryptoRepository) UpdateExchangeRate(ctx context.Context, rate *domain.CryptoExchangeRate) error {
	query := `INSERT INTO crypto_exchange_rates (id, asset, currency, rate, source, fetched_at) 
	          VALUES ($1, $2, $3, $4, $5, $6) 
	          ON CONFLICT (asset, currency) DO UPDATE SET rate = $4, source = $5, fetched_at = $6`
	_, err := r.db.Exec(ctx, query, rate.ID, rate.Asset, rate.Currency, rate.Rate, rate.Source, rate.FetchedAt)
	return err
}

// Whitelist
func (r *cryptoRepository) GetWhitelist(ctx context.Context, userID domain.UUID, chain string) ([]*domain.CryptoWithdrawalWhitelist, error) {
	query := `SELECT id, user_id, chain, address, label, is_verified, created_at FROM crypto_withdrawal_whitelist WHERE user_id = $1 AND chain = $2`
	rows, err := r.db.Query(ctx, query, userID, chain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.CryptoWithdrawalWhitelist
	for rows.Next() {
		var e domain.CryptoWithdrawalWhitelist
		rows.Scan(&e.ID, &e.UserID, &e.Chain, &e.Address, &e.Label, &e.IsVerified, &e.CreatedAt)
		list = append(list, &e)
	}
	return list, nil
}

func (r *cryptoRepository) AddToWhitelist(ctx context.Context, entry *domain.CryptoWithdrawalWhitelist) error {
	query := `INSERT INTO crypto_withdrawal_whitelist (id, user_id, chain, address, label, is_verified, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query, entry.ID, entry.UserID, entry.Chain, entry.Address, entry.Label, entry.IsVerified, entry.CreatedAt)
	return err
}

func (r *cryptoRepository) DeleteFromWhitelist(ctx context.Context, userID domain.UUID, id domain.UUID) error {
	query := `DELETE FROM crypto_withdrawal_whitelist WHERE user_id = $1 AND id = $2`
	_, err := r.db.Exec(ctx, query, userID, id)
	return err
}

func (r *cryptoRepository) GetTransactionByHash(ctx context.Context, hash string) (*domain.CryptoTransaction, error) {
	query := `SELECT id, user_id, type, chain, asset, amount_crypto, amount_idr, exchange_rate, tx_hash, from_address, to_address, confirmations, required_confirmations, status, created_at, updated_at 
	          FROM crypto_transactions WHERE tx_hash = $1`
	
	var tx domain.CryptoTransaction
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&tx.ID, &tx.UserID, &tx.Type, &tx.Chain, &tx.Asset, &tx.AmountCrypto, &tx.AmountIDR, 
		&tx.ExchangeRate, &tx.TxHash, &tx.FromAddress, &tx.ToAddress, &tx.Confirmations, 
		&tx.RequiredConfirmations, &tx.Status, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}
