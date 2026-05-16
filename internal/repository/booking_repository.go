package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type bookingRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewBookingRepository(db *pgxpool.Pool, logger *zap.Logger) domain.BookingRepository {
	return &bookingRepository{
		db:     db,
		logger: logger,
	}
}

func (r *bookingRepository) SetSchedule(ctx context.Context, schedules []*domain.HostSchedule) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete old schedule for this host
	if len(schedules) > 0 {
		_, err = tx.Exec(ctx, "DELETE FROM host_schedules WHERE host_id = $1", schedules[0].HostID)
		if err != nil {
			return err
		}
	}

	for _, s := range schedules {
		query := `INSERT INTO host_schedules (id, host_id, day_of_week, start_time, end_time, slot_duration_minutes, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`
		_, err = tx.Exec(ctx, query, s.ID, s.HostID, s.DayOfWeek, s.StartTime, s.EndTime, s.SlotDurationMinutes)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *bookingRepository) GetSchedule(ctx context.Context, hostID domain.UUID) ([]*domain.HostSchedule, error) {
	query := `SELECT id, host_id, day_of_week, start_time::text, end_time::text, slot_duration_minutes, is_active, created_at, updated_at FROM host_schedules WHERE host_id = $1 AND is_active = true`
	rows, err := r.db.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.HostSchedule
	for rows.Next() {
		s := &domain.HostSchedule{}
		err := rows.Scan(&s.ID, &s.HostID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.SlotDurationMinutes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *bookingRepository) AddException(ctx context.Context, ex *domain.HostScheduleException) error {
	query := `INSERT INTO host_schedule_exceptions (id, host_id, exception_date, type, start_time, end_time, reason, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`
	_, err := r.db.Exec(ctx, query, ex.ID, ex.HostID, ex.ExceptionDate, ex.Type, ex.StartTime, ex.EndTime, ex.Reason)
	return err
}

func (r *bookingRepository) GetExceptions(ctx context.Context, hostID domain.UUID, start, end time.Time) ([]*domain.HostScheduleException, error) {
	query := `SELECT id, host_id, exception_date, type, start_time::text, end_time::text, reason FROM host_schedule_exceptions WHERE host_id = $1 AND exception_date BETWEEN $2 AND $3`
	rows, err := r.db.Query(ctx, query, hostID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.HostScheduleException
	for rows.Next() {
		ex := &domain.HostScheduleException{}
		err := rows.Scan(&ex.ID, &ex.HostID, &ex.ExceptionDate, &ex.Type, &ex.StartTime, &ex.EndTime, &ex.Reason)
		if err != nil {
			return nil, err
		}
		result = append(result, ex)
	}
	return result, nil
}

func (r *bookingRepository) CreateBookingType(ctx context.Context, bt *domain.HostBookingType) error {
	query := `INSERT INTO host_booking_types (id, host_id, type, name, description, duration_options, price_per_minute, min_duration, max_duration, is_active, allow_extend, extend_price_per_minute, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())`
	_, err := r.db.Exec(ctx, query, bt.ID, bt.HostID, bt.Type, bt.Name, bt.Description, bt.DurationOptions, bt.PricePerMinute, bt.MinDuration, bt.MaxDuration, bt.IsActive, bt.AllowExtend, bt.ExtendPricePerMinute)
	return err
}

func (r *bookingRepository) GetBookingTypes(ctx context.Context, hostID domain.UUID) ([]*domain.HostBookingType, error) {
	query := `SELECT id, host_id, type, name, description, duration_options, price_per_minute, min_duration, max_duration, is_active, allow_extend, extend_price_per_minute, created_at, updated_at FROM host_booking_types WHERE host_id = $1 AND is_active = true`
	rows, err := r.db.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.HostBookingType
	for rows.Next() {
		bt := &domain.HostBookingType{}
		err := rows.Scan(&bt.ID, &bt.HostID, &bt.Type, &bt.Name, &bt.Description, &bt.DurationOptions, &bt.PricePerMinute, &bt.MinDuration, &bt.MaxDuration, &bt.IsActive, &bt.AllowExtend, &bt.ExtendPricePerMinute, &bt.CreatedAt, &bt.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, bt)
	}
	return result, nil
}

func (r *bookingRepository) CreateBooking(ctx context.Context, b *domain.Booking) error {
	query := `INSERT INTO bookings (id, booking_code, host_id, user_id, booking_type_id, scheduled_at, duration_minutes, ended_at, base_price, platform_fee, processing_fee, tax_fee, agency_fee, total_price, host_earning, status, user_notes, meeting_latitude, meeting_longitude, meeting_location_name, is_realtime_tracking_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW(), NOW())`
	_, err := r.db.Exec(ctx, query, b.ID, b.BookingCode, b.HostID, b.UserID, b.BookingTypeID, b.ScheduledAt, b.DurationMinutes, b.EndedAt, b.BasePrice, b.PlatformFee, b.ProcessingFee, b.TaxFee, b.AgencyFee, b.TotalPrice, b.HostEarning, b.Status, b.UserNotes, b.MeetingLatitude, b.MeetingLongitude, b.MeetingLocationName, b.IsRealtimeTrackingActive)
	return err
}

func (r *bookingRepository) GetBookingByID(ctx context.Context, id domain.UUID) (*domain.Booking, error) {
	query := `SELECT id, booking_code, host_id, user_id, booking_type_id, scheduled_at, duration_minutes, ended_at, base_price, total_price, status, payment_status, room_id, join_token, user_notes, meeting_latitude, meeting_longitude, meeting_location_name, is_realtime_tracking_active, created_at FROM bookings WHERE id = $1`
	b := &domain.Booking{}
	err := r.db.QueryRow(ctx, query, id).Scan(&b.ID, &b.BookingCode, &b.HostID, &b.UserID, &b.BookingTypeID, &b.ScheduledAt, &b.DurationMinutes, &b.EndedAt, &b.BasePrice, &b.TotalPrice, &b.Status, &b.PaymentStatus, &b.RoomID, &b.JoinToken, &b.UserNotes, &b.MeetingLatitude, &b.MeetingLongitude, &b.MeetingLocationName, &b.IsRealtimeTrackingActive, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *bookingRepository) UpdateBookingStatus(ctx context.Context, id domain.UUID, status string) error {
	query := `UPDATE bookings SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

func (r *bookingRepository) CheckOverlap(ctx context.Context, hostID domain.UUID, start, end time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM bookings WHERE host_id = $1 AND status NOT IN ('cancelled', 'host_rejected') AND (scheduled_at, ended_at) OVERLAPS ($2, $3))`
	var exists bool
	err := r.db.QueryRow(ctx, query, hostID, start, end).Scan(&exists)
	return exists, err
}

func (r *bookingRepository) ListBookingsByHost(ctx context.Context, hostID domain.UUID, status string) ([]*domain.Booking, error) {
	query := `SELECT id, booking_code, host_id, user_id, scheduled_at, duration_minutes, status FROM bookings WHERE host_id = $1`
	if status != "" {
		query += " AND status = $2"
	}
	query += " ORDER BY scheduled_at ASC"

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = r.db.Query(ctx, query, hostID, status)
	} else {
		rows, err = r.db.Query(ctx, query, hostID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Booking
	for rows.Next() {
		b := &domain.Booking{}
		err := rows.Scan(&b.ID, &b.BookingCode, &b.HostID, &b.UserID, &b.ScheduledAt, &b.DurationMinutes, &b.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

func (r *bookingRepository) ListBookingsByUser(ctx context.Context, userID domain.UUID, status string) ([]*domain.Booking, error) {
	query := `SELECT id, booking_code, host_id, user_id, scheduled_at, duration_minutes, status FROM bookings WHERE user_id = $1`
	if status != "" {
		query += " AND status = $2"
	}
	query += " ORDER BY scheduled_at DESC"

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = r.db.Query(ctx, query, userID, status)
	} else {
		rows, err = r.db.Query(ctx, query, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Booking
	for rows.Next() {
		b := &domain.Booking{}
		err := rows.Scan(&b.ID, &b.BookingCode, &b.HostID, &b.UserID, &b.ScheduledAt, &b.DurationMinutes, &b.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}
