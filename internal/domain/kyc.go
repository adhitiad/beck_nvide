package domain

import (
	"context"
	"time"
)

// KYCVerification represents eKYC data for hosts and individual agencies
type KYCVerification struct {
	ID               UUID       `json:"id" db:"id"`
	UserID           UUID       `json:"user_id" db:"user_id"`
	IDCardNumber     string     `json:"id_card_number" db:"id_card_number"`
	FullName         string     `json:"full_name" db:"full_name"`
	Gender           string     `json:"gender" db:"gender"`
	Country          string     `json:"country" db:"country"`
	DocumentURL      string     `json:"document_url" db:"document_url"`
	SelfieURL        string     `json:"selfie_url" db:"selfie_url"`
	Status           string     `json:"status" db:"status"` // 'pending', 'approved', 'rejected'
	RejectionReason  *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy       *UUID      `json:"verified_by,omitempty" db:"verified_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// AgencyVerification represents company registration data for business agencies
type AgencyVerification struct {
	ID                 UUID       `json:"id" db:"id"`
	UserID             UUID       `json:"user_id" db:"user_id"`
	CompanyName        string     `json:"company_name" db:"company_name"`
	RegistrationNumber string     `json:"registration_number" db:"registration_number"` // Akta Pendirian / SIUP / NIB
	TaxNumber          string     `json:"tax_number" db:"tax_number"`                   // NPWP Badan
	PhoneNumber        string     `json:"phone_number" db:"phone_number"`
	OfficeAddress      string     `json:"office_address" db:"office_address"`
	DocumentURL        string     `json:"document_url" db:"document_url"` // PDF of Akta/SIUP/NPWP
	Status             string     `json:"status" db:"status"`             // 'pending', 'approved', 'rejected'
	RejectionReason    *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy         *UUID      `json:"verified_by,omitempty" db:"verified_by"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// KYCRepository defines data access methods for KYC verifications
type KYCRepository interface {
	CreateKYC(ctx context.Context, kyc *KYCVerification) error
	GetKYCByUserID(ctx context.Context, userID UUID) (*KYCVerification, error)
	GetKYCByID(ctx context.Context, id UUID) (*KYCVerification, error)
	UpdateKYC(ctx context.Context, kyc *KYCVerification) error
	ListPendingKYC(ctx context.Context, limit, offset int) ([]*KYCVerification, error)

	CreateAgencyVerification(ctx context.Context, agency *AgencyVerification) error
	GetAgencyVerificationByUserID(ctx context.Context, userID UUID) (*AgencyVerification, error)
	GetAgencyVerificationByID(ctx context.Context, id UUID) (*AgencyVerification, error)
	UpdateAgencyVerification(ctx context.Context, agency *AgencyVerification) error
}
