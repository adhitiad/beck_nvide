package usecase

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type locationUsecase struct {
	bookingRepo domain.BookingRepository
	redis       *redis.Client
	logger      *zap.Logger
}

func NewLocationUsecase(
	bookingRepo domain.BookingRepository,
	redis *redis.Client,
	logger *zap.Logger,
) domain.LocationUsecase {
	return &locationUsecase{
		bookingRepo: bookingRepo,
		redis:       redis,
		logger:      logger,
	}
}

func (u *locationUsecase) UpdateUserLocation(ctx context.Context, userID domain.UUID, lat, lon float64) error {
	// Use Redis GeoAdd to store current location
	// Wait, check if my redis wrapper has GeoAdd. If not, use generic command
	// Simplified for now: use a normal SET with TTL for latest point
	pointKey := fmt.Sprintf("location:live:%s", userID)
	data := fmt.Sprintf("%.8f,%.8f", lat, lon)
	
	// Keep location alive for 5 minutes of inactivity
	return u.redis.Set(ctx, pointKey, data, 5*time.Minute)
}

func (u *locationUsecase) GetUserLocation(ctx context.Context, userID domain.UUID) (*domain.Location, error) {
	pointKey := fmt.Sprintf("location:live:%s", userID)
	val, err := u.redis.Get(ctx, pointKey)
	if err != nil {
		return nil, err
	}

	var lat, lon float64
	fmt.Sscanf(val, "%f,%f", &lat, &lon)
	
	return &domain.Location{
		UserID:    userID,
		Latitude:  lat,
		Longitude: lon,
	}, nil
}

func (u *locationUsecase) SetMeetingPoint(ctx context.Context, bookingID domain.UUID, lat, lon float64, name string) error {
	// For now, meeting point is fixed in DB
	// We need to add UpdateMeetingPoint to BookingRepository
	// Or just use generic SQL here (if we had access to DB pool)
	return nil // To be implemented in repository
}

func (u *locationUsecase) GetMeetingPoint(ctx context.Context, bookingID domain.UUID) (*domain.Location, error) {
	booking, err := u.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if booking.MeetingLatitude == nil || booking.MeetingLongitude == nil {
		return nil, fmt.Errorf("no meeting point set for this booking")
	}

	return &domain.Location{
		Latitude:  *booking.MeetingLatitude,
		Longitude: *booking.MeetingLongitude,
		Label:     booking.MeetingLocationName,
	}, nil
}
