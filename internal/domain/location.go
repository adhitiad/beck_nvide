package domain

import "context"

type Location struct {
	UserID    UUID    `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Label     string  `json:"label,omitempty"`
}

type LocationUsecase interface {
	// Real-time tracking (Redis Geo)
	UpdateUserLocation(ctx context.Context, userID UUID, lat, lon float64) error
	GetUserLocation(ctx context.Context, userID UUID) (*Location, error)
	
	// Meeting points
	SetMeetingPoint(ctx context.Context, bookingID UUID, lat, lon float64, name string) error
	GetMeetingPoint(ctx context.Context, bookingID UUID) (*Location, error)
}
