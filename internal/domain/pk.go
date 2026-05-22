package domain

import (
	"context"
	"time"
)

// PK Battle statuses
const (
	PKStatusInvite     = "invite"
	PKStatusActive     = "active"
	PKStatusPunishment = "punishment"
	PKStatusEnded      = "ended"
	PKStatusRejected   = "rejected"
)

type PKBattle struct {
	ID              UUID       `json:"id"`
	StreamAID       UUID       `json:"stream_a_id"`
	StreamBID       UUID       `json:"stream_b_id"`
	HostAID         UUID       `json:"host_a_id"`
	HostBID         UUID       `json:"host_b_id"`
	Status          string     `json:"status"` // invite, active, punishment, ended, rejected
	WinnerID        *UUID      `json:"winner_id,omitempty"`
	ScoreA          int64      `json:"score_a"`
	ScoreB          int64      `json:"score_b"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	PunishmentStart *time.Time `json:"punishment_start,omitempty"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`

	// Enhanced multi-round fields
	CurrentRound  int    `json:"current_round"`
	TotalRounds   int    `json:"total_rounds"`
	RoundDuration int    `json:"round_duration"` // seconds per round
	WinnerReward  int64  `json:"winner_reward"`
	Config        string `json:"config"` // JSONB for extra settings
}

// PKBattleRound represents scores for a single round within a PK battle
type PKBattleRound struct {
	ID          UUID       `json:"id"`
	PKID        UUID       `json:"pk_id"`
	RoundNumber int        `json:"round_number"`
	ScoreA      int64      `json:"score_a"`
	ScoreB      int64      `json:"score_b"`
	WinnerID    *UUID      `json:"winner_id,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PKBattleConfig holds configurable PK settings
type PKBattleConfig struct {
	TotalRounds   int   `json:"total_rounds"`
	RoundDuration int   `json:"round_duration"` // seconds
	WinnerReward  int64 `json:"winner_reward"`  // bonus coins
	PunishmentDur int   `json:"punishment_duration"` // seconds
}

// DefaultPKConfig returns default PK battle configuration
func DefaultPKConfig() PKBattleConfig {
	return PKBattleConfig{
		TotalRounds:   3,
		RoundDuration: 180,
		WinnerReward:  0,
		PunishmentDur: 120,
	}
}

type PKBattleRepository interface {
	Create(ctx context.Context, pk *PKBattle) error
	Update(ctx context.Context, pk *PKBattle) error
	GetByID(ctx context.Context, id UUID) (*PKBattle, error)
	GetActiveByHost(ctx context.Context, hostID UUID) (*PKBattle, error)

	// Round management
	CreateRound(ctx context.Context, round *PKBattleRound) error
	UpdateRound(ctx context.Context, round *PKBattleRound) error
	GetRound(ctx context.Context, pkID UUID, roundNumber int) (*PKBattleRound, error)
	ListRounds(ctx context.Context, pkID UUID) ([]*PKBattleRound, error)

	// History
	GetHistory(ctx context.Context, hostID UUID, limit, offset int) ([]*PKBattle, error)
}
