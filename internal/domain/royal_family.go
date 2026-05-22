package domain

import (
	"context"
	"time"
)

// Royal Family member roles
const (
	RFRoleOwner  = "owner"
	RFRoleElder  = "elder"
	RFRoleMember = "member"
)

// Contribution sources
const (
	RFContribDirect  = "direct"
	RFContribGift    = "gift"
	RFContribMission = "mission"
)

// RoyalFamily represents a host's clan/family
type RoyalFamily struct {
	ID                UUID      `json:"id"`
	HostID            UUID      `json:"host_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	BadgeURL          string    `json:"badge_url"`
	Level             int       `json:"level"`
	TotalContribution int64     `json:"total_contribution"`
	MaxMembers        int       `json:"max_members"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Host    *User                `json:"host,omitempty"`
	Members []*RoyalFamilyMember `json:"members,omitempty"`
}

// RoyalFamilyMember represents a member in a royal family
type RoyalFamilyMember struct {
	ID                UUID      `json:"id"`
	FamilyID          UUID      `json:"family_id"`
	UserID            UUID      `json:"user_id"`
	Role              string    `json:"role"` // owner, elder, member
	TotalContribution int64     `json:"total_contribution"`
	JoinedAt          time.Time `json:"joined_at"`

	// Relations
	User *User `json:"user,omitempty"`
}

// RoyalFamilyContribution represents a contribution log entry
type RoyalFamilyContribution struct {
	ID        UUID      `json:"id"`
	FamilyID  UUID      `json:"family_id"`
	UserID    UUID      `json:"user_id"`
	Amount    int64     `json:"amount"`
	Source    string    `json:"source"` // direct, gift, mission
	CreatedAt time.Time `json:"created_at"`
}

// RoyalFamilyRepository defines the contract for royal family data access
type RoyalFamilyRepository interface {
	// Family
	Create(ctx context.Context, family *RoyalFamily) error
	GetByID(ctx context.Context, id UUID) (*RoyalFamily, error)
	GetByHostID(ctx context.Context, hostID UUID) (*RoyalFamily, error)
	Update(ctx context.Context, family *RoyalFamily) error

	// Members
	AddMember(ctx context.Context, member *RoyalFamilyMember) error
	RemoveMember(ctx context.Context, familyID, userID UUID) error
	GetMember(ctx context.Context, familyID, userID UUID) (*RoyalFamilyMember, error)
	GetUserFamily(ctx context.Context, userID UUID) (*RoyalFamilyMember, error)
	ListMembers(ctx context.Context, familyID UUID, limit, offset int) ([]*RoyalFamilyMember, error)
	CountMembers(ctx context.Context, familyID UUID) (int, error)
	UpdateMemberRole(ctx context.Context, familyID, userID UUID, role string) error
	UpdateMemberContribution(ctx context.Context, familyID, userID UUID, amount int64) error

	// Contributions
	AddContribution(ctx context.Context, contrib *RoyalFamilyContribution) error
	GetContributionLeaderboard(ctx context.Context, familyID UUID, limit int) ([]*RoyalFamilyMember, error)

	// Family Leaderboard (top families)
	GetTopFamilies(ctx context.Context, limit int) ([]*RoyalFamily, error)
}
