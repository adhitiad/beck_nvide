package domain

import (
	"context"
	"time"
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
}

type PKBattleRepository interface {
	Create(ctx context.Context, pk *PKBattle) error
	Update(ctx context.Context, pk *PKBattle) error
	GetByID(ctx context.Context, id UUID) (*PKBattle, error)
	GetActiveByHost(ctx context.Context, hostID UUID) (*PKBattle, error)
}
