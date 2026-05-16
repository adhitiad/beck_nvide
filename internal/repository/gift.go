package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type giftRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewGiftRepository(db *pgxpool.Pool, logger *zap.Logger) domain.GiftRepository {
	return &giftRepository{db: db, logger: logger}
}

func (r *giftRepository) Create(ctx context.Context, gift *domain.Gift) error {
	query := `INSERT INTO gifts (id, name, icon_url, price, currency, animation_url, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, gift.ID, gift.Name, gift.IconURL, gift.Price, gift.Currency, gift.AnimationURL, gift.IsActive).Scan(&gift.CreatedAt)
}

func (r *giftRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Gift, error) {
	query := `SELECT id, name, icon_url, price, currency, animation_url, is_active, created_at FROM gifts WHERE id = $1`
	var g domain.Gift
	err := r.db.QueryRow(ctx, query, id).Scan(&g.ID, &g.Name, &g.IconURL, &g.Price, &g.Currency, &g.AnimationURL, &g.IsActive, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *giftRepository) ListActive(ctx context.Context) ([]*domain.Gift, error) {
	query := `SELECT id, name, icon_url, price, currency, animation_url, is_active, created_at FROM gifts WHERE is_active = true ORDER BY price ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*domain.Gift
	for rows.Next() {
		var g domain.Gift
		if err := rows.Scan(&g.ID, &g.Name, &g.IconURL, &g.Price, &g.Currency, &g.AnimationURL, &g.IsActive, &g.CreatedAt); err != nil {
			return nil, err
		}
		gifts = append(gifts, &g)
	}
	return gifts, nil
}

func (r *giftRepository) Update(ctx context.Context, gift *domain.Gift) error {
	query := `UPDATE gifts SET name = $1, icon_url = $2, price = $3, animation_url = $4, is_active = $5 WHERE id = $6`
	_, err := r.db.Exec(ctx, query, gift.Name, gift.IconURL, gift.Price, gift.AnimationURL, gift.IsActive, gift.ID)
	return err
}

// GiftTransaction repository
type giftTransactionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewGiftTransactionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.GiftTransactionRepository {
	return &giftTransactionRepository{db: db, logger: logger}
}

func (r *giftTransactionRepository) Create(ctx context.Context, gtx *domain.GiftTransaction) error {
	query := `INSERT INTO gift_transactions (id, stream_id, sender_id, receiver_id, gift_id, quantity, total_price, agency_id, agency_commission, host_earning, platform_fee, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, gtx.ID, gtx.StreamID, gtx.SenderID, gtx.ReceiverID, gtx.GiftID, gtx.Quantity, gtx.TotalPrice, gtx.AgencyID, gtx.AgencyCommission, gtx.HostEarning, gtx.PlatformFee).Scan(&gtx.CreatedAt)
}

func (r *giftTransactionRepository) ListByStream(ctx context.Context, streamID domain.UUID, limit, offset int) ([]*domain.GiftTransaction, error) {
	query := `SELECT id, stream_id, sender_id, receiver_id, gift_id, quantity, total_price, agency_id, agency_commission, host_earning, platform_fee, created_at
		FROM gift_transactions WHERE stream_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, streamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.GiftTransaction
	for rows.Next() {
		var gt domain.GiftTransaction
		if err := rows.Scan(&gt.ID, &gt.StreamID, &gt.SenderID, &gt.ReceiverID, &gt.GiftID, &gt.Quantity, &gt.TotalPrice, &gt.AgencyID, &gt.AgencyCommission, &gt.HostEarning, &gt.PlatformFee, &gt.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, &gt)
	}
	return txs, nil
}

func (r *giftTransactionRepository) ListBySender(ctx context.Context, senderID domain.UUID, limit, offset int) ([]*domain.GiftTransaction, error) {
	query := `SELECT id, stream_id, sender_id, receiver_id, gift_id, quantity, total_price, agency_id, agency_commission, host_earning, platform_fee, created_at
		FROM gift_transactions WHERE sender_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, senderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.GiftTransaction
	for rows.Next() {
		var gt domain.GiftTransaction
		if err := rows.Scan(&gt.ID, &gt.StreamID, &gt.SenderID, &gt.ReceiverID, &gt.GiftID, &gt.Quantity, &gt.TotalPrice, &gt.AgencyID, &gt.AgencyCommission, &gt.HostEarning, &gt.PlatformFee, &gt.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, &gt)
	}
	return txs, nil
}

// DuitkuPayment repository
type duitkuPaymentRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewDuitkuPaymentRepository(db *pgxpool.Pool, logger *zap.Logger) domain.DuitkuPaymentRepository {
	return &duitkuPaymentRepository{db: db, logger: logger}
}

func (r *duitkuPaymentRepository) Create(ctx context.Context, dp *domain.DuitkuPayment) error {
	query := `INSERT INTO duitku_payments (id, transaction_id, merchant_order_id, duitku_reference, payment_url, va_number, payment_method, status, amount, expiry_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query, dp.ID, dp.TransactionID, dp.MerchantOrderID, dp.DuitkuReference, dp.PaymentURL, dp.VANumber, dp.PaymentMethod, dp.Status, dp.Amount, dp.ExpiryAt).Scan(&dp.CreatedAt, &dp.UpdatedAt)
}

func (r *duitkuPaymentRepository) GetByMerchantOrderID(ctx context.Context, merchantOrderID string) (*domain.DuitkuPayment, error) {
	query := `SELECT id, transaction_id, merchant_order_id, duitku_reference, payment_url, va_number, payment_method, status, amount, expiry_at, created_at, updated_at
		FROM duitku_payments WHERE merchant_order_id = $1`
	var dp domain.DuitkuPayment
	err := r.db.QueryRow(ctx, query, merchantOrderID).Scan(&dp.ID, &dp.TransactionID, &dp.MerchantOrderID, &dp.DuitkuReference, &dp.PaymentURL, &dp.VANumber, &dp.PaymentMethod, &dp.Status, &dp.Amount, &dp.ExpiryAt, &dp.CreatedAt, &dp.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &dp, nil
}

func (r *duitkuPaymentRepository) Update(ctx context.Context, dp *domain.DuitkuPayment) error {
	query := `UPDATE duitku_payments SET duitku_reference = $1, status = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.Exec(ctx, query, dp.DuitkuReference, dp.Status, dp.ID)
	return err
}
