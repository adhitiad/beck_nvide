package repository

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type roleRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewRoleRepository creates new role repository
func NewRoleRepository(db *pgxpool.Pool, logger *zap.Logger) domain.RoleRepository {
	return &roleRepository{
		db:     db,
		logger: logger,
	}
}

func (r *roleRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Role, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM roles WHERE id = $1`
	role := &domain.Role{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM roles WHERE name = $1`
	role := &domain.Role{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *roleRepository) List(ctx context.Context) ([]*domain.Role, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM roles ORDER BY name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]*domain.Role, 0)
	for rows.Next() {
		role := &domain.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *roleRepository) GetPermissionsByRoleID(ctx context.Context, roleID domain.UUID) ([]domain.Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.name, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := r.db.Query(ctx, query, roleID)
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

func (r *roleRepository) GetPermissionsByRoleName(ctx context.Context, roleName string) ([]domain.Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.name, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles ro ON rp.role_id = ro.id
		WHERE ro.name = $1
		ORDER BY p.resource, p.action
	`

	rows, err := r.db.Query(ctx, query, roleName)
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

func (r *roleRepository) AssignPermission(ctx context.Context, roleID, permissionID domain.UUID) error {
	query := `INSERT INTO role_permissions (id, role_id, permission_id) VALUES (uuid_generate_v7(), $1, $2)`
	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	return err
}

func (r *roleRepository) RevokePermission(ctx context.Context, roleID, permissionID domain.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	return err
}
