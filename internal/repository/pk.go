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
	score_a, score_b, started_at, punishment_start, ended_at, created_at,
	current_round, total_rounds, round_duration, winner_reward, config
`

func scanPKBattle(row pgx.Row, pk *domain.PKBattle) error {
	return row.Scan(
		&pk.ID, &pk.StreamAID, &pk.StreamBID, &pk.HostAID, &pk.HostBID, &pk.Status, &pk.WinnerID,
		&pk.ScoreA, &pk.ScoreB, &pk.StartedAt, &pk.PunishmentStart, &pk.EndedAt, &pk.CreatedAt,
		&pk.CurrentRound, &pk.TotalRounds, &pk.RoundDuration, &pk.WinnerReward, &pk.Config,
	)
}

func (r *pkBattleRepository) Create(ctx context.Context, pk *domain.PKBattle) error {
	query := `
		INSERT INTO pk_battles (
			id, stream_a_id, stream_b_id, host_a_id, host_b_id, status, winner_id,
			score_a, score_b, started_at, punishment_start, ended_at,
			current_round, total_rounds, round_duration, winner_reward, config, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query,
		pk.ID, pk.StreamAID, pk.StreamBID, pk.HostAID, pk.HostBID, pk.Status, pk.WinnerID,
		pk.ScoreA, pk.ScoreB, pk.StartedAt, pk.PunishmentStart, pk.EndedAt,
		pk.CurrentRound, pk.TotalRounds, pk.RoundDuration, pk.WinnerReward, pk.Config,
	).Scan(&pk.CreatedAt)
}

func (r *pkBattleRepository) Update(ctx context.Context, pk *domain.PKBattle) error {
	query := `
		UPDATE pk_battles
		SET status = $1, winner_id = $2, score_a = $3, score_b = $4,
			started_at = $5, punishment_start = $6, ended_at = $7,
			current_round = $8, winner_reward = $9, config = $10
		WHERE id = $11
	`
	_, err := r.db.Exec(ctx, query,
		pk.Status, pk.WinnerID, pk.ScoreA, pk.ScoreB,
		pk.StartedAt, pk.PunishmentStart, pk.EndedAt,
		pk.CurrentRound, pk.WinnerReward, pk.Config, pk.ID,
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

func (r *pkBattleRepository) CreateRound(ctx context.Context, round *domain.PKBattleRound) error {
	query := `INSERT INTO pk_battle_rounds (id, pk_id, round_number, score_a, score_b, winner_id,
		started_at, ended_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, round.ID, round.PKID, round.RoundNumber, round.ScoreA,
		round.ScoreB, round.WinnerID, round.StartedAt, round.EndedAt).Scan(&round.CreatedAt)
}

func (r *pkBattleRepository) UpdateRound(ctx context.Context, round *domain.PKBattleRound) error {
	query := `UPDATE pk_battle_rounds SET score_a=$1, score_b=$2, winner_id=$3, ended_at=$4
		WHERE id=$5`
	_, err := r.db.Exec(ctx, query, round.ScoreA, round.ScoreB, round.WinnerID, round.EndedAt, round.ID)
	return err
}

func (r *pkBattleRepository) GetRound(ctx context.Context, pkID domain.UUID, roundNumber int) (*domain.PKBattleRound, error) {
	query := `SELECT id, pk_id, round_number, score_a, score_b, winner_id, started_at, ended_at, created_at
		FROM pk_battle_rounds WHERE pk_id=$1 AND round_number=$2`
	var round domain.PKBattleRound
	err := r.db.QueryRow(ctx, query, pkID, roundNumber).Scan(&round.ID, &round.PKID,
		&round.RoundNumber, &round.ScoreA, &round.ScoreB, &round.WinnerID,
		&round.StartedAt, &round.EndedAt, &round.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &round, nil
}

func (r *pkBattleRepository) ListRounds(ctx context.Context, pkID domain.UUID) ([]*domain.PKBattleRound, error) {
	query := `SELECT id, pk_id, round_number, score_a, score_b, winner_id, started_at, ended_at, created_at
		FROM pk_battle_rounds WHERE pk_id=$1 ORDER BY round_number`
	rows, err := r.db.Query(ctx, query, pkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.PKBattleRound
	for rows.Next() {
		var round domain.PKBattleRound
		if err := rows.Scan(&round.ID, &round.PKID, &round.RoundNumber, &round.ScoreA,
			&round.ScoreB, &round.WinnerID, &round.StartedAt, &round.EndedAt, &round.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &round)
	}
	return list, nil
}

func (r *pkBattleRepository) GetHistory(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.PKBattle, error) {
	query := `SELECT ` + selectPKBattleFields + `
		FROM pk_battles WHERE (host_a_id=$1 OR host_b_id=$1) AND status='ended'
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, hostID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.PKBattle
	for rows.Next() {
		var pk domain.PKBattle
		if err := scanPKBattle(rows, &pk); err != nil {
			return nil, err
		}
		list = append(list, &pk)
	}
	return list, nil
}
