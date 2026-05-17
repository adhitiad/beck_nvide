package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type pkBattleRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPKBattleRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PKBattleRepository {
	return &pkBattleRepository{
		db:     db,
		logger: logger,
	}
}

const selectPKBattleFields = `
	id, stream_a_id, stream_b_id, host_a_id, host_b_id, status, winner_id,
	score_a, score_b, started_at, punishment_start, ended_at, created_at
`

func scanPKBattle(row pgx.Row, pk *domain.PKBattle) error {
	return row.Scan(
		&pk.ID, &pk.StreamAID, &pk.StreamBID, &pk.HostAID, &pk.HostBID, &pk.Status, &pk.WinnerID,
		&pk.ScoreA, &pk.ScoreB, &pk.StartedAt, &pk.PunishmentStart, &pk.EndedAt, &pk.CreatedAt,
	)
}

func (r *pkBattleRepository) Create(ctx context.Context, pk *domain.PKBattle) error {
	query := `
		INSERT INTO pk_battles (
			id, stream_a_id, stream_b_id, host_a_id, host_b_id, status, winner_id,
			score_a, score_b, started_at, punishment_start, ended_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query,
		pk.ID, pk.StreamAID, pk.StreamBID, pk.HostAID, pk.HostBID, pk.Status, pk.WinnerID,
		pk.ScoreA, pk.ScoreB, pk.StartedAt, pk.PunishmentStart, pk.EndedAt,
	).Scan(&pk.CreatedAt)
}

func (r *pkBattleRepository) Update(ctx context.Context, pk *domain.PKBattle) error {
	query := `
		UPDATE pk_battles
		SET status = $1, winner_id = $2, score_a = $3, score_b = $4,
			started_at = $5, punishment_start = $6, ended_at = $7
		WHERE id = $8
	`
	_, err := r.db.Exec(ctx, query,
		pk.Status, pk.WinnerID, pk.ScoreA, pk.ScoreB,
		pk.StartedAt, pk.PunishmentStart, pk.EndedAt, pk.ID,
	)
	return err
}

func (r *pkBattleRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.PKBattle, error) {
	query := `SELECT ` + selectPKBattleFields + ` FROM pk_battles WHERE id = $1`
	var pk domain.PKBattle
	err := scanPKBattle(r.db.QueryRow(ctx, query, id), &pk)
	if err != nil {
		return nil, err
	}
	return &pk, nil
}

func (r *pkBattleRepository) GetActiveByHost(ctx context.Context, hostID domain.UUID) (*domain.PKBattle, error) {
	query := `
		SELECT ` + selectPKBattleFields + `
		FROM pk_battles
		WHERE (host_a_id = $1 OR host_b_id = $1) AND status IN ('invite', 'active', 'punishment')
		ORDER BY created_at DESC
		LIMIT 1
	`
	var pk domain.PKBattle
	err := scanPKBattle(r.db.QueryRow(ctx, query, hostID), &pk)
	if err != nil {
		return nil, err
	}
	return &pk, nil
}
