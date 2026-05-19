package repository

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type bannedUserRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewBannedUserRepository creates new banned user repository
func NewBannedUserRepository(db *pgxpool.Pool, logger *zap.Logger) domain.BannedUserRepository {
	return &bannedUserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *bannedUserRepository) BanUser(ctx context.Context, banned *domain.BannedUser) error {
	query := `
		INSERT INTO banned_users (id, user_id, reason, banned_at, is_permanent, can_appeal, device_fingerprint, ip_address)
		VALUES ($1, $2, $3, NOW(), $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE 
		SET reason = EXCLUDED.reason, 
		    banned_at = NOW(), 
		    is_permanent = EXCLUDED.is_permanent, 
		    can_appeal = EXCLUDED.can_appeal,
		    device_fingerprint = EXCLUDED.device_fingerprint,
		    ip_address = EXCLUDED.ip_address
	`

	var (
		fingerprint sql.NullString
		ipAddress   sql.NullString
	)

	if banned.DeviceFingerprint != nil {
		fingerprint = sql.NullString{String: *banned.DeviceFingerprint, Valid: true}
	}
	if banned.IPAddress != nil {
		ipAddress = sql.NullString{String: *banned.IPAddress, Valid: true}
	}

	_, err := r.db.Exec(ctx, query,
		banned.ID,
		banned.UserID,
		banned.Reason,
		banned.IsPermanent,
		banned.CanAppeal,
		fingerprint,
		ipAddress,
	)

	if err != nil {
		r.logger.Error("Failed to insert into banned_users", zap.Error(err), zap.String("user_id", banned.UserID.String()))
		return err
	}

	return nil
}

func (r *bannedUserRepository) UnbanUser(ctx context.Context, userID domain.UUID) error {
	query := `DELETE FROM banned_users WHERE user_id = $1`
	res, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to unban user", zap.Error(err), zap.String("user_id", userID.String()))
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *bannedUserRepository) IsBanned(ctx context.Context, userID domain.UUID) (bool, *domain.BannedUser, error) {
	query := `
		SELECT id, user_id, reason, banned_at, is_permanent, can_appeal, device_fingerprint, ip_address
		FROM banned_users
		WHERE user_id = $1
	`
	b := &domain.BannedUser{}
	var (
		fingerprint sql.NullString
		ipAddress   sql.NullString
	)

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&b.ID, &b.UserID, &b.Reason, &b.BannedAt, &b.IsPermanent, &b.CanAppeal, &fingerprint, &ipAddress,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		r.logger.Error("Failed to check if user is banned", zap.Error(err), zap.String("user_id", userID.String()))
		return false, nil, err
	}

	if fingerprint.Valid {
		b.DeviceFingerprint = &fingerprint.String
	}
	if ipAddress.Valid {
		b.IPAddress = &ipAddress.String
	}

	return true, b, nil
}

func (r *bannedUserRepository) IsDeviceBanned(ctx context.Context, fingerprint string) (bool, *domain.BannedUser, error) {
	if fingerprint == "" {
		return false, nil, nil
	}
	query := `
		SELECT id, user_id, reason, banned_at, is_permanent, can_appeal, device_fingerprint, ip_address
		FROM banned_users
		WHERE device_fingerprint = $1
		LIMIT 1
	`
	b := &domain.BannedUser{}
	var (
		fingerprintNull sql.NullString
		ipAddress       sql.NullString
	)

	err := r.db.QueryRow(ctx, query, fingerprint).Scan(
		&b.ID, &b.UserID, &b.Reason, &b.BannedAt, &b.IsPermanent, &b.CanAppeal, &fingerprintNull, &ipAddress,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		r.logger.Error("Failed to check if device is banned", zap.Error(err), zap.String("fingerprint", fingerprint))
		return false, nil, err
	}

	if fingerprintNull.Valid {
		b.DeviceFingerprint = &fingerprintNull.String
	}
	if ipAddress.Valid {
		b.IPAddress = &ipAddress.String
	}

	return true, b, nil
}

func (r *bannedUserRepository) IsIPBanned(ctx context.Context, ip string) (bool, *domain.BannedUser, error) {
	if ip == "" {
		return false, nil, nil
	}
	query := `
		SELECT id, user_id, reason, banned_at, is_permanent, can_appeal, device_fingerprint, ip_address
		FROM banned_users
		WHERE ip_address = $1
		LIMIT 1
	`
	b := &domain.BannedUser{}
	var (
		fingerprintNull sql.NullString
		ipAddress       sql.NullString
	)

	err := r.db.QueryRow(ctx, query, ip).Scan(
		&b.ID, &b.UserID, &b.Reason, &b.BannedAt, &b.IsPermanent, &b.CanAppeal, &fingerprintNull, &ipAddress,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		r.logger.Error("Failed to check if IP is banned", zap.Error(err), zap.String("ip", ip))
		return false, nil, err
	}

	if fingerprintNull.Valid {
		b.DeviceFingerprint = &fingerprintNull.String
	}
	if ipAddress.Valid {
		b.IPAddress = &ipAddress.String
	}

	return true, b, nil
}

func (r *bannedUserRepository) ListBanned(ctx context.Context, limit, offset int) ([]*domain.BannedUser, error) {
	query := `
		SELECT id, user_id, reason, banned_at, is_permanent, can_appeal, device_fingerprint, ip_address
		FROM banned_users
		ORDER BY banned_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list banned users", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	list := make([]*domain.BannedUser, 0)
	for rows.Next() {
		b := &domain.BannedUser{}
		var (
			fingerprint sql.NullString
			ipAddress   sql.NullString
		)
		err := rows.Scan(
			&b.ID, &b.UserID, &b.Reason, &b.BannedAt, &b.IsPermanent, &b.CanAppeal, &fingerprint, &ipAddress,
		)
		if err != nil {
			return nil, err
		}
		if fingerprint.Valid {
			b.DeviceFingerprint = &fingerprint.String
		}
		if ipAddress.Valid {
			b.IPAddress = &ipAddress.String
		}
		list = append(list, b)
	}

	return list, nil
}
