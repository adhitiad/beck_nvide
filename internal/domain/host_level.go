package domain

import (
	"context"
	"time"
)

// Host tier names
const (
	HostTierNewbie  = "newbie"
	HostTierBronze  = "bronze"
	HostTierSilver  = "silver"
	HostTierGold    = "gold"
	HostTierDiamond = "diamond"
)

// HostLevel represents a host tier definition
type HostLevel struct {
	ID              UUID      `json:"id"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"display_name"`
	MinStreamHours  int       `json:"min_stream_hours"`
	MinTotalIncome  int64     `json:"min_total_income"`
	CommissionRate  int       `json:"commission_rate"` // host share %
	BadgeURL        string    `json:"badge_url"`
	Perks           string    `json:"perks"` // JSONB
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

// HostLevelRepository defines the contract for host level data access
type HostLevelRepository interface {
	ListAll(ctx context.Context) ([]*HostLevel, error)
	GetByName(ctx context.Context, name string) (*HostLevel, error)
	GetByID(ctx context.Context, id UUID) (*HostLevel, error)
	GetEligibleLevel(ctx context.Context, streamHours int, totalIncome int64) (*HostLevel, error)
}
