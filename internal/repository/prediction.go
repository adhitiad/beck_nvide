package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type predictionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPredictionRepository membuat instance baru dari PredictionRepository
func NewPredictionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PredictionRepository {
	return &predictionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *predictionRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *predictionRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ctxWithTx := WithTx(ctx, tx)
	if err := fn(ctxWithTx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *predictionRepository) Create(ctx context.Context, p *domain.Prediction) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO predictions (id, stream_id, question, status, total_yes_pool, total_no_pool, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	return exec.QueryRow(ctx, query, p.ID, p.StreamID, p.Question, p.Status, p.TotalYesPool, p.TotalNoPool).Scan(&p.CreatedAt)
}

func (r *predictionRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Prediction, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, question, status, resolved_outcome, total_yes_pool, total_no_pool, created_at, resolved_at 
		FROM predictions WHERE id = $1`
	var p domain.Prediction
	err := exec.QueryRow(ctx, query, id).Scan(&p.ID, &p.StreamID, &p.Question, &p.Status, &p.ResolvedOutcome, &p.TotalYesPool, &p.TotalNoPool, &p.CreatedAt, &p.ResolvedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *predictionRepository) GetActiveByStreamID(ctx context.Context, streamID domain.UUID) ([]*domain.Prediction, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, question, status, resolved_outcome, total_yes_pool, total_no_pool, created_at, resolved_at 
		FROM predictions WHERE stream_id = $1 AND status = 'active' ORDER BY created_at DESC`
	rows, err := exec.Query(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Prediction
	for rows.Next() {
		var p domain.Prediction
		err := rows.Scan(&p.ID, &p.StreamID, &p.Question, &p.Status, &p.ResolvedOutcome, &p.TotalYesPool, &p.TotalNoPool, &p.CreatedAt, &p.ResolvedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

func (r *predictionRepository) CreateBet(ctx context.Context, bet *domain.PredictionBet) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO prediction_bets (id, prediction_id, user_id, outcome, amount, currency_type, creator_token_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) RETURNING created_at`
	return exec.QueryRow(ctx, query, bet.ID, bet.PredictionID, bet.UserID, bet.Outcome, bet.Amount, bet.CurrencyType, bet.CreatorTokenID).Scan(&bet.CreatedAt)
}

func (r *predictionRepository) GetBetsByPredictionID(ctx context.Context, predictionID domain.UUID) ([]*domain.PredictionBet, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, prediction_id, user_id, outcome, amount, currency_type, creator_token_id, created_at 
		FROM prediction_bets WHERE prediction_id = $1`
	rows, err := exec.Query(ctx, query, predictionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []*domain.PredictionBet
	for rows.Next() {
		var b domain.PredictionBet
		err := rows.Scan(&b.ID, &b.PredictionID, &b.UserID, &b.Outcome, &b.Amount, &b.CurrencyType, &b.CreatorTokenID, &b.CreatedAt)
		if err != nil {
			return nil, err
		}
		bets = append(bets, &b)
	}
	return bets, nil
}

func (r *predictionRepository) UpdatePools(ctx context.Context, id domain.UUID, yesAmount, noAmount int64) error {
	exec := r.getExecutor(ctx)
	// Execute row lock on prediction first
	var dummy string
	err := exec.QueryRow(ctx, "SELECT status FROM predictions WHERE id = $1 FOR UPDATE", id).Scan(&dummy)
	if err != nil {
		return err
	}

	query := `UPDATE predictions SET total_yes_pool = total_yes_pool + $1, total_no_pool = total_no_pool + $2 WHERE id = $3`
	_, err = exec.Exec(ctx, query, yesAmount, noAmount, id)
	return err
}

func (r *predictionRepository) ResolvePrediction(ctx context.Context, id domain.UUID, outcome string) error {
	exec := r.getExecutor(ctx)
	query := `UPDATE predictions SET status = 'resolved', resolved_outcome = $1, resolved_at = NOW() WHERE id = $2`
	_, err := exec.Exec(ctx, query, outcome, id)
	return err
}
