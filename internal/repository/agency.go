package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type agencyRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewAgencyRepository(db *pgxpool.Pool, logger *zap.Logger) domain.AgencyRepository {
	return &agencyRepository{db: db, logger: logger}
}

func (r *agencyRepository) Create(ctx context.Context, agency *domain.Agency) error {
	query := `INSERT INTO agencies (id, owner_id, name, description, logo_url, commission_rate, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query, agency.ID, agency.OwnerID, agency.Name, agency.Description, agency.LogoURL, agency.CommissionRate, agency.Status).Scan(&agency.CreatedAt, &agency.UpdatedAt)
}

func (r *agencyRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Agency, error) {
	query := `SELECT id, owner_id, name, description, logo_url, commission_rate, status, created_at, updated_at FROM agencies WHERE id = $1`
	var a domain.Agency
	err := r.db.QueryRow(ctx, query, id).Scan(&a.ID, &a.OwnerID, &a.Name, &a.Description, &a.LogoURL, &a.CommissionRate, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *agencyRepository) GetByOwnerID(ctx context.Context, ownerID domain.UUID) (*domain.Agency, error) {
	query := `SELECT id, owner_id, name, description, logo_url, commission_rate, status, created_at, updated_at FROM agencies WHERE owner_id = $1`
	var a domain.Agency
	err := r.db.QueryRow(ctx, query, ownerID).Scan(&a.ID, &a.OwnerID, &a.Name, &a.Description, &a.LogoURL, &a.CommissionRate, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *agencyRepository) Update(ctx context.Context, agency *domain.Agency) error {
	query := `UPDATE agencies SET name = $1, description = $2, logo_url = $3, commission_rate = $4, status = $5, updated_at = NOW()
		WHERE id = $6 RETURNING updated_at`
	return r.db.QueryRow(ctx, query, agency.Name, agency.Description, agency.LogoURL, agency.CommissionRate, agency.Status, agency.ID).Scan(&agency.UpdatedAt)
}

func (r *agencyRepository) AddHost(ctx context.Context, ah *domain.AgencyHost) error {
	query := `INSERT INTO agency_hosts (agency_id, host_id, joined_at, status, revenue_share, total_earnings)
		VALUES ($1, $2, NOW(), $3, $4, 0) RETURNING joined_at`
	return r.db.QueryRow(ctx, query, ah.AgencyID, ah.HostID, ah.Status, ah.RevenueShare).Scan(&ah.JoinedAt)
}

func (r *agencyRepository) RemoveHost(ctx context.Context, agencyID, hostID domain.UUID) error {
	query := `UPDATE agency_hosts SET status = 'removed' WHERE agency_id = $1 AND host_id = $2`
	_, err := r.db.Exec(ctx, query, agencyID, hostID)
	return err
}

func (r *agencyRepository) GetHostRelation(ctx context.Context, hostID domain.UUID) (*domain.AgencyHost, error) {
	query := `SELECT agency_id, host_id, joined_at, status, revenue_share, total_earnings
		FROM agency_hosts WHERE host_id = $1 AND status = 'active'`
	var ah domain.AgencyHost
	err := r.db.QueryRow(ctx, query, hostID).Scan(&ah.AgencyID, &ah.HostID, &ah.JoinedAt, &ah.Status, &ah.RevenueShare, &ah.TotalEarnings)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &ah, nil
}

func (r *agencyRepository) ListHosts(ctx context.Context, agencyID domain.UUID) ([]*domain.AgencyHost, error) {
	query := `SELECT ah.agency_id, ah.host_id, ah.joined_at, ah.status, ah.revenue_share, ah.total_earnings
		FROM agency_hosts ah WHERE ah.agency_id = $1 AND ah.status = 'active'`
	rows, err := r.db.Query(ctx, query, agencyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []*domain.AgencyHost
	for rows.Next() {
		var ah domain.AgencyHost
		if err := rows.Scan(&ah.AgencyID, &ah.HostID, &ah.JoinedAt, &ah.Status, &ah.RevenueShare, &ah.TotalEarnings); err != nil {
			return nil, err
		}
		hosts = append(hosts, &ah)
	}
	return hosts, nil
}

func (r *agencyRepository) UpdateHostEarnings(ctx context.Context, agencyID, hostID domain.UUID, amount int64) error {
	query := `UPDATE agency_hosts SET total_earnings = total_earnings + $1 WHERE agency_id = $2 AND host_id = $3`
	_, err := r.db.Exec(ctx, query, amount, agencyID, hostID)
	return err
}

// HostApplicationRepository
type hostApplicationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewHostApplicationRepository(db *pgxpool.Pool, logger *zap.Logger) domain.HostApplicationRepository {
	return &hostApplicationRepository{db: db, logger: logger}
}

func (r *hostApplicationRepository) Create(ctx context.Context, app *domain.HostApplication) error {
	query := `INSERT INTO host_applications (id, user_id, bio, id_card_url, bank_account_name, bank_account_number, bank_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query, app.ID, app.UserID, app.Bio, app.IDCardURL, app.BankAccountName, app.BankAccountNumber, app.BankName, app.Status).Scan(&app.CreatedAt, &app.UpdatedAt)
}

func (r *hostApplicationRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.HostApplication, error) {
	query := `SELECT id, user_id, bio, id_card_url, bank_account_name, bank_account_number, bank_name, status, reviewed_by, reviewed_at, rejection_reason, created_at, updated_at
		FROM host_applications WHERE id = $1`
	var a domain.HostApplication
	err := r.db.QueryRow(ctx, query, id).Scan(&a.ID, &a.UserID, &a.Bio, &a.IDCardURL, &a.BankAccountName, &a.BankAccountNumber, &a.BankName, &a.Status, &a.ReviewedBy, &a.ReviewedAt, &a.RejectionReason, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *hostApplicationRepository) GetByUserID(ctx context.Context, userID domain.UUID) (*domain.HostApplication, error) {
	query := `SELECT id, user_id, bio, id_card_url, bank_account_name, bank_account_number, bank_name, status, reviewed_by, reviewed_at, rejection_reason, created_at, updated_at
		FROM host_applications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`
	var a domain.HostApplication
	err := r.db.QueryRow(ctx, query, userID).Scan(&a.ID, &a.UserID, &a.Bio, &a.IDCardURL, &a.BankAccountName, &a.BankAccountNumber, &a.BankName, &a.Status, &a.ReviewedBy, &a.ReviewedAt, &a.RejectionReason, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *hostApplicationRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.HostApplication, error) {
	query := `SELECT id, user_id, bio, id_card_url, bank_account_name, bank_account_number, bank_name, status, reviewed_by, reviewed_at, rejection_reason, created_at, updated_at
		FROM host_applications WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*domain.HostApplication
	for rows.Next() {
		var a domain.HostApplication
		if err := rows.Scan(&a.ID, &a.UserID, &a.Bio, &a.IDCardURL, &a.BankAccountName, &a.BankAccountNumber, &a.BankName, &a.Status, &a.ReviewedBy, &a.ReviewedAt, &a.RejectionReason, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, &a)
	}
	return apps, nil
}

func (r *hostApplicationRepository) Update(ctx context.Context, app *domain.HostApplication) error {
	query := `UPDATE host_applications SET status = $1, reviewed_by = $2, reviewed_at = $3, rejection_reason = $4, updated_at = NOW() WHERE id = $5`
	_, err := r.db.Exec(ctx, query, app.Status, app.ReviewedBy, app.ReviewedAt, app.RejectionReason, app.ID)
	return err
}
