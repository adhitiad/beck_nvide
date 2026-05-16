package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type permissionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPermissionRepository creates new permission repository
func NewPermissionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PermissionRepository {
	return &permissionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *permissionRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Permission, error) {
	query := `SELECT id, resource, action, name, description, created_at FROM permissions WHERE id = $1`
	p := &domain.Permission{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Resource, &p.Action, &p.Name, &p.Description, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *permissionRepository) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	query := `SELECT id, resource, action, name, description, created_at FROM permissions WHERE name = $1`
	p := &domain.Permission{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Resource, &p.Action, &p.Name, &p.Description, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *permissionRepository) List(ctx context.Context) ([]domain.Permission, error) {
	query := `SELECT id, resource, action, name, description, created_at FROM permissions ORDER BY resource, action`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]domain.Permission, 0)
	for rows.Next() {
		var p domain.Permission
		err := rows.Scan(
			&p.ID, &p.Resource, &p.Action, &p.Name, &p.Description, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func (r *permissionRepository) ListByResource(ctx context.Context, resource string) ([]domain.Permission, error) {
	query := `SELECT id, resource, action, name, description, created_at FROM permissions WHERE resource = $1 ORDER BY action`
	rows, err := r.db.Query(ctx, query, resource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]domain.Permission, 0)
	for rows.Next() {
		var p domain.Permission
		err := rows.Scan(
			&p.ID, &p.Resource, &p.Action, &p.Name, &p.Description, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func (r *permissionRepository) BulkCreate(ctx context.Context, permissions []domain.Permission) error {
	// This is for seeding, not for runtime
	for _, p := range permissions {
		query := `
			INSERT INTO permissions (id, resource, action, name, description, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (name) DO NOTHING
		`
		_, err := r.db.Exec(ctx, query, p.ID, p.Resource, p.Action, p.Name, p.Description)
		if err != nil {
			return err
		}
	}
	return nil
}
