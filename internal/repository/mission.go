package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type missionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewMissionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.MissionRepository {
	return &missionRepository{db: db, logger: logger}
}

func (r *missionRepository) ListActiveMissions(ctx context.Context, roleTarget string) ([]*domain.DailyMission, error) {
	query := `SELECT id, title, description, type, target_value, reward_type, reward_value,
		reward_item_id, role_target, is_active, created_at
		FROM daily_missions WHERE is_active = true AND (role_target = 'all' OR role_target = $1)`
	rows, err := r.db.Query(ctx, query, roleTarget)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DailyMission
	for rows.Next() {
		var m domain.DailyMission
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.Type, &m.TargetValue,
			&m.RewardType, &m.RewardValue, &m.RewardItemID, &m.RoleTarget, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}
	return list, nil
}

func (r *missionRepository) GetMissionByID(ctx context.Context, id domain.UUID) (*domain.DailyMission, error) {
	query := `SELECT id, title, description, type, target_value, reward_type, reward_value,
		reward_item_id, role_target, is_active, created_at FROM daily_missions WHERE id=$1`
	var m domain.DailyMission
	err := r.db.QueryRow(ctx, query, id).Scan(&m.ID, &m.Title, &m.Description, &m.Type,
		&m.TargetValue, &m.RewardType, &m.RewardValue, &m.RewardItemID, &m.RoleTarget,
		&m.IsActive, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *missionRepository) GetOrCreateUserMissions(ctx context.Context, userID domain.UUID, date time.Time) ([]*domain.UserMission, error) {
	// Insert missions for today if not exists, then return them all
	dateStr := date.Format("2006-01-02")
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_missions (id, user_id, mission_id, progress, is_completed, is_claimed, mission_date, created_at)
		SELECT uuid_generate_v7(), $1, dm.id, 0, false, false, $2::date, NOW()
		FROM daily_missions dm
		WHERE dm.is_active = true
		ON CONFLICT (user_id, mission_id, mission_date) DO NOTHING`, userID, dateStr)
	if err != nil {
		return nil, err
	}

	query := `SELECT um.id, um.user_id, um.mission_id, um.progress, um.is_completed, um.is_claimed,
		um.mission_date, um.claimed_at, um.created_at,
		dm.id, dm.title, dm.description, dm.type, dm.target_value, dm.reward_type, dm.reward_value,
		dm.reward_item_id, dm.role_target, dm.is_active, dm.created_at
		FROM user_missions um
		JOIN daily_missions dm ON um.mission_id = dm.id
		WHERE um.user_id = $1 AND um.mission_date = $2::date
		ORDER BY dm.created_at`
	rows, err := r.db.Query(ctx, query, userID, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserMission
	for rows.Next() {
		var um domain.UserMission
		var dm domain.DailyMission
		if err := rows.Scan(&um.ID, &um.UserID, &um.MissionID, &um.Progress, &um.IsCompleted,
			&um.IsClaimed, &um.MissionDate, &um.ClaimedAt, &um.CreatedAt,
			&dm.ID, &dm.Title, &dm.Description, &dm.Type, &dm.TargetValue, &dm.RewardType,
			&dm.RewardValue, &dm.RewardItemID, &dm.RoleTarget, &dm.IsActive, &dm.CreatedAt); err != nil {
			return nil, err
		}
		um.Mission = &dm
		list = append(list, &um)
	}
	return list, nil
}

func (r *missionRepository) GetUserMission(ctx context.Context, userID, missionID domain.UUID, date time.Time) (*domain.UserMission, error) {
	query := `SELECT id, user_id, mission_id, progress, is_completed, is_claimed, mission_date, claimed_at, created_at
		FROM user_missions WHERE user_id=$1 AND mission_id=$2 AND mission_date=$3::date`
	var um domain.UserMission
	err := r.db.QueryRow(ctx, query, userID, missionID, date.Format("2006-01-02")).
		Scan(&um.ID, &um.UserID, &um.MissionID, &um.Progress, &um.IsCompleted,
			&um.IsClaimed, &um.MissionDate, &um.ClaimedAt, &um.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &um, nil
}

func (r *missionRepository) UpdateProgress(ctx context.Context, id domain.UUID, progress int, completed bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_missions SET progress=$1, is_completed=$2 WHERE id=$3`,
		progress, completed, id)
	return err
}

func (r *missionRepository) ClaimReward(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_missions SET is_claimed=true, claimed_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *missionRepository) IncrementMissionProgress(ctx context.Context, userID domain.UUID, missionType string, delta int, date time.Time) error {
	query := `UPDATE user_missions um
		SET progress = LEAST(um.progress + $1, dm.target_value),
			is_completed = (um.progress + $1 >= dm.target_value)
		FROM daily_missions dm
		WHERE um.mission_id = dm.id
			AND um.user_id = $2 AND dm.type = $3
			AND um.mission_date = $4::date
			AND um.is_completed = false`
	_, err := r.db.Exec(ctx, query, delta, userID, missionType, date.Format("2006-01-02"))
	return err
}

func (r *missionRepository) AwardBadge(ctx context.Context, badge *domain.UserBadge) error {
	query := `INSERT INTO user_badges (id, user_id, badge_name, badge_icon, achievement_key, description, earned_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, achievement_key) DO NOTHING`
	_, err := r.db.Exec(ctx, query, badge.ID, badge.UserID, badge.BadgeName, badge.BadgeIcon,
		badge.AchievementKey, badge.Description)
	return err
}

func (r *missionRepository) GetUserBadges(ctx context.Context, userID domain.UUID) ([]*domain.UserBadge, error) {
	query := `SELECT id, user_id, badge_name, badge_icon, achievement_key, description, earned_at
		FROM user_badges WHERE user_id=$1 ORDER BY earned_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserBadge
	for rows.Next() {
		var b domain.UserBadge
		if err := rows.Scan(&b.ID, &b.UserID, &b.BadgeName, &b.BadgeIcon, &b.AchievementKey,
			&b.Description, &b.EarnedAt); err != nil {
			return nil, err
		}
		list = append(list, &b)
	}
	return list, nil
}

func (r *missionRepository) HasBadge(ctx context.Context, userID domain.UUID, achievementKey string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_badges WHERE user_id=$1 AND achievement_key=$2)`,
		userID, achievementKey).Scan(&exists)
	return exists, err
}

func (r *missionRepository) CreateSnapshot(ctx context.Context, entry *domain.LeaderboardEntry) error {
	query := `INSERT INTO leaderboard_snapshots (id, type, period, user_id, score, rank, snapshot_date, reward_claimed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, NOW())`
	_, err := r.db.Exec(ctx, query, entry.ID, entry.Type, entry.Period, entry.UserID,
		entry.Score, entry.Rank, entry.SnapshotDate)
	return err
}

func (r *missionRepository) GetLeaderboard(ctx context.Context, lbType, period string, date time.Time, limit int) ([]*domain.LeaderboardEntry, error) {
	query := `SELECT ls.id, ls.type, ls.period, ls.user_id, ls.score, ls.rank, ls.snapshot_date,
		ls.reward_claimed, ls.created_at,
		u.id, u.username, u.avatar_url
		FROM leaderboard_snapshots ls
		JOIN users u ON ls.user_id = u.id
		WHERE ls.type=$1 AND ls.period=$2 AND ls.snapshot_date=$3
		ORDER BY ls.rank ASC LIMIT $4`
	rows, err := r.db.Query(ctx, query, lbType, period, date.Format("2006-01-02"), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LeaderboardEntry
	for rows.Next() {
		var e domain.LeaderboardEntry
		var u domain.User
		if err := rows.Scan(&e.ID, &e.Type, &e.Period, &e.UserID, &e.Score, &e.Rank,
			&e.SnapshotDate, &e.RewardClaimed, &e.CreatedAt,
			&u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		e.User = &u
		list = append(list, &e)
	}
	return list, nil
}

func (r *missionRepository) GetUserRank(ctx context.Context, lbType, period string, userID domain.UUID, date time.Time) (*domain.LeaderboardEntry, error) {
	query := `SELECT id, type, period, user_id, score, rank, snapshot_date, reward_claimed, created_at
		FROM leaderboard_snapshots WHERE type=$1 AND period=$2 AND user_id=$3 AND snapshot_date=$4`
	var e domain.LeaderboardEntry
	err := r.db.QueryRow(ctx, query, lbType, period, userID, date.Format("2006-01-02")).
		Scan(&e.ID, &e.Type, &e.Period, &e.UserID, &e.Score, &e.Rank,
			&e.SnapshotDate, &e.RewardClaimed, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
