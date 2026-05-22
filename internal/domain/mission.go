package domain

import (
	"context"
	"time"
)

// Mission types
const (
	MissionTypeWatch      = "watch"
	MissionTypeSendGift   = "send_gift"
	MissionTypeLiveStream = "live_stream"
	MissionTypeLogin      = "login"
	MissionTypeShare      = "share"
	MissionTypeComment    = "comment"
)

// Reward types
const (
	RewardTypeCoin = "coin"
	RewardTypeEXP  = "exp"
	RewardTypeItem = "item"
)

// Leaderboard types
const (
	LeaderboardHostIncome         = "host_income"
	LeaderboardUserGift           = "user_gift"
	LeaderboardFamilyContribution = "family_contribution"
)

// Leaderboard periods
const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
)

// DailyMission represents a mission template
type DailyMission struct {
	ID           UUID   `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Type         string `json:"type"`          // watch, send_gift, live_stream, login, share, comment
	TargetValue  int    `json:"target_value"`  // how many times or minutes
	RewardType   string `json:"reward_type"`   // coin, exp, item
	RewardValue  int64  `json:"reward_value"`
	RewardItemID *UUID  `json:"reward_item_id,omitempty"`
	RoleTarget   string `json:"role_target"` // all, user, host
	IsActive     bool   `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserMission represents a user's progress on a mission
type UserMission struct {
	ID          UUID       `json:"id"`
	UserID      UUID       `json:"user_id"`
	MissionID   UUID       `json:"mission_id"`
	Progress    int        `json:"progress"`
	IsCompleted bool       `json:"is_completed"`
	IsClaimed   bool       `json:"is_claimed"`
	MissionDate time.Time  `json:"mission_date"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Relations
	Mission *DailyMission `json:"mission,omitempty"`
}

// UserBadge represents an achievement badge earned by a user
type UserBadge struct {
	ID             UUID      `json:"id"`
	UserID         UUID      `json:"user_id"`
	BadgeName      string    `json:"badge_name"`
	BadgeIcon      string    `json:"badge_icon"`
	AchievementKey string    `json:"achievement_key"` // top_supporter, veteran_viewer, etc.
	Description    string    `json:"description"`
	EarnedAt       time.Time `json:"earned_at"`
}

// LeaderboardEntry represents a snapshot entry in a leaderboard
type LeaderboardEntry struct {
	ID            UUID      `json:"id"`
	Type          string    `json:"type"`   // host_income, user_gift, family_contribution
	Period        string    `json:"period"` // daily, weekly, monthly
	UserID        UUID      `json:"user_id"`
	Score         int64     `json:"score"`
	Rank          int       `json:"rank"`
	SnapshotDate  time.Time `json:"snapshot_date"`
	RewardClaimed bool      `json:"reward_claimed"`
	CreatedAt     time.Time `json:"created_at"`

	// Relations
	User *User `json:"user,omitempty"`
}

// MissionRepository defines the contract for mission and gamification data access
type MissionRepository interface {
	// Missions
	ListActiveMissions(ctx context.Context, roleTarget string) ([]*DailyMission, error)
	GetMissionByID(ctx context.Context, id UUID) (*DailyMission, error)

	// User missions
	GetOrCreateUserMissions(ctx context.Context, userID UUID, date time.Time) ([]*UserMission, error)
	GetUserMission(ctx context.Context, userID, missionID UUID, date time.Time) (*UserMission, error)
	UpdateProgress(ctx context.Context, id UUID, progress int, completed bool) error
	ClaimReward(ctx context.Context, id UUID) error
	IncrementMissionProgress(ctx context.Context, userID UUID, missionType string, delta int, date time.Time) error

	// Badges
	AwardBadge(ctx context.Context, badge *UserBadge) error
	GetUserBadges(ctx context.Context, userID UUID) ([]*UserBadge, error)
	HasBadge(ctx context.Context, userID UUID, achievementKey string) (bool, error)

	// Leaderboard
	CreateSnapshot(ctx context.Context, entry *LeaderboardEntry) error
	GetLeaderboard(ctx context.Context, lbType, period string, date time.Time, limit int) ([]*LeaderboardEntry, error)
	GetUserRank(ctx context.Context, lbType, period string, userID UUID, date time.Time) (*LeaderboardEntry, error)
}
