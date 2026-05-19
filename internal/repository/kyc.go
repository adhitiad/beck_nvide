package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type kycRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewKYCRepository creates new KYC repository
func NewKYCRepository(db *pgxpool.Pool, logger *zap.Logger) domain.KYCRepository {
	return &kycRepository{
		db:     db,
		logger: logger,
	}
}

func (r *kycRepository) CreateKYC(ctx context.Context, kyc *domain.KYCVerification) error {
	query := `
		INSERT INTO kyc_verifications (id, user_id, id_card_number, full_name, gender, country, document_url, selfie_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, query,
		kyc.ID,
		kyc.UserID,
		kyc.IDCardNumber,
		kyc.FullName,
		kyc.Gender,
		kyc.Country,
		kyc.DocumentURL,
		kyc.SelfieURL,
		kyc.Status,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		r.logger.Error("Failed to create KYC", zap.Error(err), zap.String("user_id", kyc.UserID.String()))
		return err
	}

	kyc.CreatedAt = createdAt
	kyc.UpdatedAt = updatedAt
	return nil
}

func (r *kycRepository) GetKYCByUserID(ctx context.Context, userID domain.UUID) (*domain.KYCVerification, error) {
	query := `
		SELECT id, user_id, id_card_number, full_name, gender, country, document_url, selfie_url, status, rejection_reason, verified_at, verified_by, created_at, updated_at
		FROM kyc_verifications
		WHERE user_id = $1
	`
	kyc := &domain.KYCVerification{}
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
	)

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&kyc.ID, &kyc.UserID, &kyc.IDCardNumber, &kyc.FullName, &kyc.Gender, &kyc.Country,
		&kyc.DocumentURL, &kyc.SelfieURL, &kyc.Status, &rejectionReason, &verifiedAt, &verifiedBy,
		&kyc.CreatedAt, &kyc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get KYC by user ID", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	if rejectionReason.Valid {
		kyc.RejectionReason = &rejectionReason.String
	}
	if verifiedAt.Valid {
		kyc.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		vBy := domain.UUID(verifiedBy.String)
		kyc.VerifiedBy = &vBy
	}

	return kyc, nil
}

func (r *kycRepository) GetKYCByID(ctx context.Context, id domain.UUID) (*domain.KYCVerification, error) {
	query := `
		SELECT id, user_id, id_card_number, full_name, gender, country, document_url, selfie_url, status, rejection_reason, verified_at, verified_by, created_at, updated_at
		FROM kyc_verifications
		WHERE id = $1
	`
	kyc := &domain.KYCVerification{}
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&kyc.ID, &kyc.UserID, &kyc.IDCardNumber, &kyc.FullName, &kyc.Gender, &kyc.Country,
		&kyc.DocumentURL, &kyc.SelfieURL, &kyc.Status, &rejectionReason, &verifiedAt, &verifiedBy,
		&kyc.CreatedAt, &kyc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get KYC by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if rejectionReason.Valid {
		kyc.RejectionReason = &rejectionReason.String
	}
	if verifiedAt.Valid {
		kyc.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		vBy := domain.UUID(verifiedBy.String)
		kyc.VerifiedBy = &vBy
	}

	return kyc, nil
}

func (r *kycRepository) UpdateKYC(ctx context.Context, kyc *domain.KYCVerification) error {
	query := `
		UPDATE kyc_verifications
		SET status = $1, rejection_reason = $2, verified_at = $3, verified_by = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
		updatedAt      time.Time
	)

	if kyc.RejectionReason != nil {
		rejectionReason = sql.NullString{String: *kyc.RejectionReason, Valid: true}
	}
	if kyc.VerifiedAt != nil {
		verifiedAt = sql.NullTime{Time: *kyc.VerifiedAt, Valid: true}
	}
	if kyc.VerifiedBy != nil {
		verifiedBy = sql.NullString{String: kyc.VerifiedBy.String(), Valid: true}
	}

	err := r.db.QueryRow(ctx, query,
		kyc.Status,
		rejectionReason,
		verifiedAt,
		verifiedBy,
		kyc.ID,
	).Scan(&updatedAt)

	if err != nil {
		r.logger.Error("Failed to update KYC", zap.Error(err), zap.String("id", kyc.ID.String()))
		return err
	}

	kyc.UpdatedAt = updatedAt
	return nil
}

func (r *kycRepository) ListPendingKYC(ctx context.Context, limit, offset int) ([]*domain.KYCVerification, error) {
	query := `
		SELECT id, user_id, id_card_number, full_name, gender, country, document_url, selfie_url, status, created_at, updated_at
		FROM kyc_verifications
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list pending KYC", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	list := make([]*domain.KYCVerification, 0)
	for rows.Next() {
		kyc := &domain.KYCVerification{}
		err := rows.Scan(
			&kyc.ID, &kyc.UserID, &kyc.IDCardNumber, &kyc.FullName, &kyc.Gender, &kyc.Country,
			&kyc.DocumentURL, &kyc.SelfieURL, &kyc.Status, &kyc.CreatedAt, &kyc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, kyc)
	}

	return list, nil
}

func (r *kycRepository) CreateAgencyVerification(ctx context.Context, agency *domain.AgencyVerification) error {
	query := `
		INSERT INTO agency_verifications (id, user_id, company_name, registration_number, tax_number, phone_number, office_address, document_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, query,
		agency.ID,
		agency.UserID,
		agency.CompanyName,
		agency.RegistrationNumber,
		agency.TaxNumber,
		agency.PhoneNumber,
		agency.OfficeAddress,
		agency.DocumentURL,
		agency.Status,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		r.logger.Error("Failed to create agency verification", zap.Error(err), zap.String("user_id", agency.UserID.String()))
		return err
	}

	agency.CreatedAt = createdAt
	agency.UpdatedAt = updatedAt
	return nil
}

func (r *kycRepository) GetAgencyVerificationByUserID(ctx context.Context, userID domain.UUID) (*domain.AgencyVerification, error) {
	query := `
		SELECT id, user_id, company_name, registration_number, tax_number, phone_number, office_address, document_url, status, rejection_reason, verified_at, verified_by, created_at, updated_at
		FROM agency_verifications
		WHERE user_id = $1
	`
	agency := &domain.AgencyVerification{}
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
	)

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&agency.ID, &agency.UserID, &agency.CompanyName, &agency.RegistrationNumber, &agency.TaxNumber,
		&agency.PhoneNumber, &agency.OfficeAddress, &agency.DocumentURL, &agency.Status,
		&rejectionReason, &verifiedAt, &verifiedBy, &agency.CreatedAt, &agency.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get agency verification by user ID", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	if rejectionReason.Valid {
		agency.RejectionReason = &rejectionReason.String
	}
	if verifiedAt.Valid {
		agency.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		vBy := domain.UUID(verifiedBy.String)
		agency.VerifiedBy = &vBy
	}

	return agency, nil
}

func (r *kycRepository) GetAgencyVerificationByID(ctx context.Context, id domain.UUID) (*domain.AgencyVerification, error) {
	query := `
		SELECT id, user_id, company_name, registration_number, tax_number, phone_number, office_address, document_url, status, rejection_reason, verified_at, verified_by, created_at, updated_at
		FROM agency_verifications
		WHERE id = $1
	`
	agency := &domain.AgencyVerification{}
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&agency.ID, &agency.UserID, &agency.CompanyName, &agency.RegistrationNumber, &agency.TaxNumber,
		&agency.PhoneNumber, &agency.OfficeAddress, &agency.DocumentURL, &agency.Status,
		&rejectionReason, &verifiedAt, &verifiedBy, &agency.CreatedAt, &agency.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get agency verification by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if rejectionReason.Valid {
		agency.RejectionReason = &rejectionReason.String
	}
	if verifiedAt.Valid {
		agency.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		vBy := domain.UUID(verifiedBy.String)
		agency.VerifiedBy = &vBy
	}

	return agency, nil
}

func (r *kycRepository) UpdateAgencyVerification(ctx context.Context, agency *domain.AgencyVerification) error {
	query := `
		UPDATE agency_verifications
		SET status = $1, rejection_reason = $2, verified_at = $3, verified_by = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`
	var (
		rejectionReason sql.NullString
		verifiedAt      sql.NullTime
		verifiedBy      sql.NullString
		updatedAt      time.Time
	)

	if agency.RejectionReason != nil {
		rejectionReason = sql.NullString{String: *agency.RejectionReason, Valid: true}
	}
	if agency.VerifiedAt != nil {
		verifiedAt = sql.NullTime{Time: *agency.VerifiedAt, Valid: true}
	}
	if agency.VerifiedBy != nil {
		verifiedBy = sql.NullString{String: agency.VerifiedBy.String(), Valid: true}
	}

	err := r.db.QueryRow(ctx, query,
		agency.Status,
		rejectionReason,
		verifiedAt,
		verifiedBy,
		agency.ID,
	).Scan(&updatedAt)

	if err != nil {
		r.logger.Error("Failed to update agency verification", zap.Error(err), zap.String("id", agency.ID.String()))
		return err
	}

	agency.UpdatedAt = updatedAt
	return nil
}
