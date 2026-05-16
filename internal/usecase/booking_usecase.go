package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type bookingUsecase struct {
	repo         domain.BookingRepository
	walletRepo   domain.WalletRepository
	agencyRepo   domain.AgencyRepository
	withdrawalUC domain.WithdrawalUsecase // For fee calculation
	redis        *redis.Client
	logger       *zap.Logger
}

func NewBookingUsecase(
	repo domain.BookingRepository,
	walletRepo domain.WalletRepository,
	agencyRepo domain.AgencyRepository,
	withdrawalUC domain.WithdrawalUsecase,
	redis *redis.Client,
	logger *zap.Logger,
) domain.BookingUsecase {
	return &bookingUsecase{
		repo:         repo,
		walletRepo:   walletRepo,
		agencyRepo:   agencyRepo,
		withdrawalUC: withdrawalUC,
		redis:        redis,
		logger:       logger,
	}
}

func (u *bookingUsecase) SetSchedule(ctx context.Context, schedules []*domain.HostSchedule) error {
	return u.repo.SetSchedule(ctx, schedules)
}

func (u *bookingUsecase) SetBookingType(ctx context.Context, bt *domain.HostBookingType) error {
	return u.repo.CreateBookingType(ctx, bt)
}

func (u *bookingUsecase) GetBookingTypes(ctx context.Context, hostID domain.UUID) ([]*domain.HostBookingType, error) {
	return u.repo.GetBookingTypes(ctx, hostID)
}

func (u *bookingUsecase) GetAvailableSlots(ctx context.Context, hostID domain.UUID, date time.Time) ([]map[string]interface{}, error) {
	// 1. Get recurring schedule for that day of week
	dayOfWeek := int(date.Weekday())
	schedules, err := u.repo.GetSchedule(ctx, hostID)
	if err != nil {
		return nil, err
	}

	var daySchedule *domain.HostSchedule
	for _, s := range schedules {
		if s.DayOfWeek == dayOfWeek {
			daySchedule = s
			break
		}
	}

	if daySchedule == nil {
		return []map[string]interface{}{}, nil
	}

	// 2. Get exceptions for that date
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	exceptions, _ := u.repo.GetExceptions(ctx, hostID, startOfDay, endOfDay)

	// 3. Get existing bookings for that date
	// (Simplified: just check overlap for each slot)

	// 4. Generate slots
	startTime, _ := time.Parse("15:04:05", daySchedule.StartTime)
	endTime, _ := time.Parse("15:04:05", daySchedule.EndTime)
	
	duration := time.Duration(daySchedule.SlotDurationMinutes) * time.Minute
	var slots []map[string]interface{}

	curr := startTime
	for curr.Add(duration).Before(endTime) || curr.Add(duration).Equal(endTime) {
		slotStart := time.Date(date.Year(), date.Month(), date.Day(), curr.Hour(), curr.Minute(), 0, 0, date.Location())
		slotEnd := slotStart.Add(duration)

		// Check if unavailable in exceptions
		isUnavailable := false
		for _, ex := range exceptions {
			if ex.Type == "unavailable" {
				isUnavailable = true
				break
			}
			// TODO: check special hours
		}

		if !isUnavailable {
			// Check overlap with existing bookings
			booked, _ := u.repo.CheckOverlap(ctx, hostID, slotStart, slotEnd)
			slots = append(slots, map[string]interface{}{
				"start_time": slotStart,
				"end_time":   slotEnd,
				"is_booked":  booked,
			})
		}

		curr = curr.Add(duration)
	}

	return slots, nil
}

func (u *bookingUsecase) RequestBooking(ctx context.Context, userID, hostID, typeID domain.UUID, scheduledAt time.Time, duration int, notes string, lat, lon *float64, locName string) (*domain.Booking, error) {
	// 1. Validate slot availability
	endedAt := scheduledAt.Add(time.Duration(duration) * time.Minute)
	overlap, err := u.repo.CheckOverlap(ctx, hostID, scheduledAt, endedAt)
	if err != nil || overlap {
		return nil, errors.New("slot is already booked or unavailable")
	}

	// 2. Get price and calculate fees
	bookingTypes, _ := u.repo.GetBookingTypes(ctx, hostID)
	var bType *domain.HostBookingType
	for _, bt := range bookingTypes {
		if bt.ID == typeID {
			bType = bt
			break
		}
	}
	if bType == nil {
		return nil, errors.New("invalid booking type")
	}

	basePrice := bType.PricePerMinute * float64(duration)
	
	// Based on spec: Total = Base + Platform + Processing + Tax + Agency
	
	// Actually based on spec in Withdrawal: Total = Base - Fees. 
	// But in Booking spec: Total = Base + Platform Fee + Processing Fee + Tax + Agency Fee
	// Let's follow Booking spec: User pays Base + Fees. Host gets Base - Agency Fee.
	
	fPlatform := float64(int64(basePrice)) * 0.15
	fProcessing := float64(int64(basePrice)) * 0.035
	fTax := float64(int64(basePrice)) * 0.10
	
	agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, hostID)
	fAgency := 0.0
	if agencyRel != nil {
		fAgency = float64(int64(basePrice)) * 0.067
	}

	grandTotal := basePrice + fPlatform + fProcessing + fTax + fAgency

	// 3. Check and Freeze Balance
	wallet, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil || float64(wallet.Balance) < grandTotal {
		return nil, errors.New("insufficient balance")
	}

	// 4. Create Booking
	booking := &domain.Booking{
		ID:              domain.NewUUIDv7(),
		BookingCode:     fmt.Sprintf("BOK-%s-%d", time.Now().Format("20060102"), rand.Intn(9999)),
		HostID:          hostID,
		UserID:          userID,
		BookingTypeID:   typeID,
		ScheduledAt:     scheduledAt,
		DurationMinutes: duration,
		EndedAt:         endedAt,
		BasePrice:       basePrice,
		PlatformFee:     fPlatform,
		ProcessingFee:   fProcessing,
		TaxFee:          fTax,
		AgencyFee:       fAgency,
		TotalPrice:      grandTotal,
		HostEarning:     basePrice - fAgency,
		Status:          domain.BookingPending,
		UserNotes:       notes,
		MeetingLatitude:     lat,
		MeetingLongitude:    lon,
		MeetingLocationName: locName,
		CreatedAt:     time.Now(),
	}

	err = u.walletRepo.DebitBalance(ctx, userID, int64(grandTotal))
	if err != nil {
		return nil, err
	}

	err = u.repo.CreateBooking(ctx, booking)
	if err != nil {
		return nil, err
	}

	// 5. Lock slot in Redis for 5 minutes
	lockKey := fmt.Sprintf("booking:lock:%s:%d", hostID, scheduledAt.Unix())
	u.redis.Set(ctx, lockKey, booking.ID.String(), 5*time.Minute)

	return booking, nil
}

func (u *bookingUsecase) AcceptBooking(ctx context.Context, hostID, bookingID domain.UUID) error {
	booking, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil || booking.HostID != hostID {
		return errors.New("booking not found")
	}

	if booking.Status != domain.BookingPending {
		return errors.New("booking is not pending")
	}

	return u.repo.UpdateBookingStatus(ctx, bookingID, domain.BookingConfirmed)
}

func (u *bookingUsecase) RejectBooking(ctx context.Context, hostID, bookingID domain.UUID, reason string) error {
	booking, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil || booking.HostID != hostID {
		return errors.New("booking not found")
	}

	// Refund escrow
	u.walletRepo.CreditBalance(ctx, booking.UserID, int64(booking.TotalPrice))

	return u.repo.UpdateBookingStatus(ctx, bookingID, domain.BookingRejected)
}

func (u *bookingUsecase) CancelBooking(ctx context.Context, userID, bookingID domain.UUID, reason string) error {
	booking, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil || booking.UserID != userID {
		return errors.New("booking not found")
	}

	// Apply cancellation policy (simplified: full refund if > 24h)
	now := time.Now()
	if booking.ScheduledAt.Sub(now) > 24*time.Hour {
		u.walletRepo.CreditBalance(ctx, booking.UserID, int64(booking.TotalPrice))
	} else {
		// Partial refund logic...
		u.walletRepo.CreditBalance(ctx, booking.UserID, int64(booking.TotalPrice/2))
	}

	return u.repo.UpdateBookingStatus(ctx, bookingID, domain.BookingCancelled)
}

func (u *bookingUsecase) StartSession(ctx context.Context, bookingID domain.UUID) (string, error) {
	// 1. Generate room and token (Mocked)
	return "mock-join-token", nil
}

func (u *bookingUsecase) EndSession(ctx context.Context, bookingID domain.UUID) error {
	// 1. Mark complete
	u.repo.UpdateBookingStatus(ctx, bookingID, domain.BookingCompleted)
	
	// 2. Release funds after delay (Simplified: immediate)
	booking, _ := u.repo.GetBookingByID(ctx, bookingID)
	u.walletRepo.CreditBalance(ctx, booking.HostID, int64(booking.HostEarning))
	
	if booking.AgencyFee > 0 {
		agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, booking.HostID)
		if agencyRel != nil {
			agency, _ := u.agencyRepo.GetByID(ctx, agencyRel.AgencyID)
			u.walletRepo.CreditBalance(ctx, agency.OwnerID, int64(booking.AgencyFee))
		}
	}
	
	return nil
}
