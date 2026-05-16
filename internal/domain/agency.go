package domain

import (
	"context"
	"time"
)

// Host application statuses
const (
	ApplicationPending  = "pending"
	ApplicationApproved = "approved"
	ApplicationRejected = "rejected"
)

// Agency status
const (
	AgencyStatusActive    = "active"
	AgencyStatusSuspended = "suspended"
)

// Agency host status
const (
	AgencyHostActive  = "active"
	AgencyHostInvited = "invited"
	AgencyHostRemoved = "removed"
)

// HostApplication represents a host application
type HostApplication struct {
	ID                UUID       `json:"id"`
	UserID            UUID       `json:"user_id"`
	Bio               string     `json:"bio"`
	IDCardURL         string     `json:"id_card_url"`
	BankAccountName   string     `json:"bank_account_name"`
	BankAccountNumber string     `json:"bank_account_number"`
	BankName          string     `json:"bank_name"`
	Status            string     `json:"status"`
	ReviewedBy        *UUID      `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason   string     `json:"rejection_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Agency represents an agency entity
type Agency struct {
	ID             UUID      `json:"id"`
	OwnerID        UUID      `json:"owner_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	LogoURL        string    `json:"logo_url"`
	CommissionRate int       `json:"commission_rate"` // percentage
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AgencyHost represents a host under an agency
type AgencyHost struct {
	AgencyID      UUID      `json:"agency_id"`
	HostID        UUID      `json:"host_id"`
	JoinedAt      time.Time `json:"joined_at"`
	Status        string    `json:"status"`
	RevenueShare  int       `json:"revenue_share"`  // host share %
	TotalEarnings int64     `json:"total_earnings"`

	// Relations
	Host   *User   `json:"host,omitempty"`
	Agency *Agency `json:"agency,omitempty"`
}

// Repositories
type HostApplicationRepository interface {
	Create(ctx context.Context, app *HostApplication) error
	GetByID(ctx context.Context, id UUID) (*HostApplication, error)
	GetByUserID(ctx context.Context, userID UUID) (*HostApplication, error)
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]*HostApplication, error)
	Update(ctx context.Context, app *HostApplication) error
}

type AgencyRepository interface {
	Create(ctx context.Context, agency *Agency) error
	GetByID(ctx context.Context, id UUID) (*Agency, error)
	GetByOwnerID(ctx context.Context, ownerID UUID) (*Agency, error)
	Update(ctx context.Context, agency *Agency) error

	AddHost(ctx context.Context, ah *AgencyHost) error
	RemoveHost(ctx context.Context, agencyID, hostID UUID) error
	GetHostRelation(ctx context.Context, hostID UUID) (*AgencyHost, error)
	ListHosts(ctx context.Context, agencyID UUID) ([]*AgencyHost, error)
	UpdateHostEarnings(ctx context.Context, agencyID, hostID UUID, amount int64) error
}
