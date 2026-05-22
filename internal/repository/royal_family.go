package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type royalFamilyRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewRoyalFamilyRepository(db *pgxpool.Pool, logger *zap.Logger) domain.RoyalFamilyRepository {
	return &royalFamilyRepository{db: db, logger: logger}
}

func (r *royalFamilyRepository) Create(ctx context.Context, family *domain.RoyalFamily) error {
	query := `INSERT INTO royal_families (id, host_id, name, description, badge_url, level, total_contribution, max_members, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query, family.ID, family.HostID, family.Name, family.Description,
		family.BadgeURL, family.Level, family.TotalContribution, family.MaxMembers).
		Scan(&family.CreatedAt, &family.UpdatedAt)
}

func (r *royalFamilyRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.RoyalFamily, error) {
	query := `SELECT id, host_id, name, description, badge_url, level, total_contribution, max_members, created_at, updated_at
		FROM royal_families WHERE id = $1`
	var f domain.RoyalFamily
	err := r.db.QueryRow(ctx, query, id).Scan(&f.ID, &f.HostID, &f.Name, &f.Description,
		&f.BadgeURL, &f.Level, &f.TotalContribution, &f.MaxMembers, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *royalFamilyRepository) GetByHostID(ctx context.Context, hostID domain.UUID) (*domain.RoyalFamily, error) {
	query := `SELECT id, host_id, name, description, badge_url, level, total_contribution, max_members, created_at, updated_at
		FROM royal_families WHERE host_id = $1`
	var f domain.RoyalFamily
	err := r.db.QueryRow(ctx, query, hostID).Scan(&f.ID, &f.HostID, &f.Name, &f.Description,
		&f.BadgeURL, &f.Level, &f.TotalContribution, &f.MaxMembers, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *royalFamilyRepository) Update(ctx context.Context, family *domain.RoyalFamily) error {
	query := `UPDATE royal_families SET name=$1, description=$2, badge_url=$3, level=$4,
		total_contribution=$5, max_members=$6, updated_at=NOW() WHERE id=$7`
	_, err := r.db.Exec(ctx, query, family.Name, family.Description, family.BadgeURL,
		family.Level, family.TotalContribution, family.MaxMembers, family.ID)
	return err
}

func (r *royalFamilyRepository) AddMember(ctx context.Context, member *domain.RoyalFamilyMember) error {
	query := `INSERT INTO royal_family_members (id, family_id, user_id, role, total_contribution, joined_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING joined_at`
	return r.db.QueryRow(ctx, query, member.ID, member.FamilyID, member.UserID, member.Role,
		member.TotalContribution).Scan(&member.JoinedAt)
}

func (r *royalFamilyRepository) RemoveMember(ctx context.Context, familyID, userID domain.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM royal_family_members WHERE family_id=$1 AND user_id=$2`, familyID, userID)
	return err
}

func (r *royalFamilyRepository) GetMember(ctx context.Context, familyID, userID domain.UUID) (*domain.RoyalFamilyMember, error) {
	query := `SELECT id, family_id, user_id, role, total_contribution, joined_at
		FROM royal_family_members WHERE family_id=$1 AND user_id=$2`
	var m domain.RoyalFamilyMember
	err := r.db.QueryRow(ctx, query, familyID, userID).Scan(&m.ID, &m.FamilyID, &m.UserID,
		&m.Role, &m.TotalContribution, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *royalFamilyRepository) GetUserFamily(ctx context.Context, userID domain.UUID) (*domain.RoyalFamilyMember, error) {
	query := `SELECT id, family_id, user_id, role, total_contribution, joined_at
		FROM royal_family_members WHERE user_id=$1 LIMIT 1`
	var m domain.RoyalFamilyMember
	err := r.db.QueryRow(ctx, query, userID).Scan(&m.ID, &m.FamilyID, &m.UserID,
		&m.Role, &m.TotalContribution, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *royalFamilyRepository) ListMembers(ctx context.Context, familyID domain.UUID, limit, offset int) ([]*domain.RoyalFamilyMember, error) {
	query := `SELECT rfm.id, rfm.family_id, rfm.user_id, rfm.role, rfm.total_contribution, rfm.joined_at,
		u.id, u.username, u.avatar_url
		FROM royal_family_members rfm
		JOIN users u ON rfm.user_id = u.id
		WHERE rfm.family_id=$1 ORDER BY rfm.total_contribution DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, familyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.RoyalFamilyMember
	for rows.Next() {
		var m domain.RoyalFamilyMember
		var u domain.User
		if err := rows.Scan(&m.ID, &m.FamilyID, &m.UserID, &m.Role, &m.TotalContribution, &m.JoinedAt,
			&u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		m.User = &u
		list = append(list, &m)
	}
	return list, nil
}

func (r *royalFamilyRepository) CountMembers(ctx context.Context, familyID domain.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM royal_family_members WHERE family_id=$1`, familyID).Scan(&count)
	return count, err
}

func (r *royalFamilyRepository) UpdateMemberRole(ctx context.Context, familyID, userID domain.UUID, role string) error {
	_, err := r.db.Exec(ctx, `UPDATE royal_family_members SET role=$1 WHERE family_id=$2 AND user_id=$3`, role, familyID, userID)
	return err
}

func (r *royalFamilyRepository) UpdateMemberContribution(ctx context.Context, familyID, userID domain.UUID, amount int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE royal_family_members SET total_contribution = total_contribution + $1 WHERE family_id=$2 AND user_id=$3`,
		amount, familyID, userID)
	return err
}

func (r *royalFamilyRepository) AddContribution(ctx context.Context, contrib *domain.RoyalFamilyContribution) error {
	query := `INSERT INTO royal_family_contributions (id, family_id, user_id, amount, source, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, contrib.ID, contrib.FamilyID, contrib.UserID,
		contrib.Amount, contrib.Source).Scan(&contrib.CreatedAt)
}

func (r *royalFamilyRepository) GetContributionLeaderboard(ctx context.Context, familyID domain.UUID, limit int) ([]*domain.RoyalFamilyMember, error) {
	query := `SELECT rfm.id, rfm.family_id, rfm.user_id, rfm.role, rfm.total_contribution, rfm.joined_at,
		u.id, u.username, u.avatar_url
		FROM royal_family_members rfm
		JOIN users u ON rfm.user_id = u.id
		WHERE rfm.family_id=$1
		ORDER BY rfm.total_contribution DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, familyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.RoyalFamilyMember
	for rows.Next() {
		var m domain.RoyalFamilyMember
		var u domain.User
		if err := rows.Scan(&m.ID, &m.FamilyID, &m.UserID, &m.Role, &m.TotalContribution, &m.JoinedAt,
			&u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		m.User = &u
		list = append(list, &m)
	}
	return list, nil
}

func (r *royalFamilyRepository) GetTopFamilies(ctx context.Context, limit int) ([]*domain.RoyalFamily, error) {
	query := `SELECT id, host_id, name, description, badge_url, level, total_contribution, max_members, created_at, updated_at
		FROM royal_families ORDER BY total_contribution DESC LIMIT $1`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.RoyalFamily
	for rows.Next() {
		var f domain.RoyalFamily
		if err := rows.Scan(&f.ID, &f.HostID, &f.Name, &f.Description, &f.BadgeURL,
			&f.Level, &f.TotalContribution, &f.MaxMembers, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &f)
	}
	return list, nil
}
