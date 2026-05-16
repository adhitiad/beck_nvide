package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type offerRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewOfferRepository(db *pgxpool.Pool, logger *zap.Logger) domain.OfferRepository {
	return &offerRepository{
		db:     db,
		logger: logger,
	}
}

func (r *offerRepository) CreateHostOffer(ctx context.Context, o *domain.HostOffer) error {
	query := `INSERT INTO host_offers (id, host_id, offer_code, title, description, booking_type_id, offer_mode, specific_at, recurring_days, recurring_start_time, recurring_end_time, slot_duration_minutes, base_price_per_minute, discount_percentage, final_price_per_minute, max_bookings, status, expires_at, is_auto_confirm, tags, thumbnail_url, latitude, longitude, location_name, share_location_type, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, NOW(), NOW())`
	_, err := r.db.Exec(ctx, query, o.ID, o.HostID, o.OfferCode, o.Title, o.Description, o.BookingTypeID, o.OfferMode, o.SpecificAt, o.RecurringDays, o.RecurringStartTime, o.RecurringEndTime, o.SlotDurationMinutes, o.BasePricePerMinute, o.DiscountPercentage, o.FinalPricePerMinute, o.MaxBookings, o.Status, o.ExpiresAt, o.IsAutoConfirm, o.Tags, o.ThumbnailURL, o.Latitude, o.Longitude, o.LocationName, o.ShareLocationType)
	return err
}

func (r *offerRepository) GetHostOfferByID(ctx context.Context, id domain.UUID) (*domain.HostOffer, error) {
	query := `SELECT id, host_id, offer_code, title, description, booking_type_id, offer_mode, specific_at, recurring_days, recurring_start_time::text, recurring_end_time::text, slot_duration_minutes, base_price_per_minute, discount_percentage, final_price_per_minute, max_bookings, bookings_made, status, expires_at, is_auto_confirm, tags, thumbnail_url, latitude, longitude, location_name, share_location_type FROM host_offers WHERE id = $1`
	o := &domain.HostOffer{}
	err := r.db.QueryRow(ctx, query, id).Scan(&o.ID, &o.HostID, &o.OfferCode, &o.Title, &o.Description, &o.BookingTypeID, &o.OfferMode, &o.SpecificAt, &o.RecurringDays, &o.RecurringStartTime, &o.RecurringEndTime, &o.SlotDurationMinutes, &o.BasePricePerMinute, &o.DiscountPercentage, &o.FinalPricePerMinute, &o.MaxBookings, &o.BookingsMade, &o.Status, &o.ExpiresAt, &o.IsAutoConfirm, &o.Tags, &o.ThumbnailURL, &o.Latitude, &o.Longitude, &o.LocationName, &o.ShareLocationType)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (r *offerRepository) UpdateHostOfferBookings(ctx context.Context, id domain.UUID, bookingsMade int) error {
	query := `UPDATE host_offers SET bookings_made = $1, status = CASE WHEN $1 >= max_bookings THEN 'fully_booked' ELSE status END, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, bookingsMade, id)
	return err
}

func (r *offerRepository) CreateUserOffer(ctx context.Context, o *domain.UserOffer) error {
	query := `INSERT INTO user_offers (id, offer_code, user_id, host_id, booking_type_id, offer_type, proposed_at, proposed_duration_minutes, proposed_price_per_minute, total_offer_amount, message, status, is_prepaid, prepaid_amount, latitude, longitude, location_name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW())`
	_, err := r.db.Exec(ctx, query, o.ID, o.OfferCode, o.UserID, o.HostID, o.BookingTypeID, o.OfferType, o.ProposedAt, o.ProposedDurationMinutes, o.ProposedPricePerMinute, o.TotalOfferAmount, o.Message, o.Status, o.IsPrepaid, o.PrepaidAmount, o.Latitude, o.Longitude, o.LocationName)
	return err
}

func (r *offerRepository) GetUserOfferByID(ctx context.Context, id domain.UUID) (*domain.UserOffer, error) {
	query := `SELECT id, offer_code, user_id, host_id, booking_type_id, offer_type, proposed_at, proposed_duration_minutes, proposed_price_per_minute, total_offer_amount, message, status, host_message, host_counter_price, is_prepaid, prepaid_amount, latitude, longitude, location_name FROM user_offers WHERE id = $1`
	o := &domain.UserOffer{}
	err := r.db.QueryRow(ctx, query, id).Scan(&o.ID, &o.OfferCode, &o.UserID, &o.HostID, &o.BookingTypeID, &o.OfferType, &o.ProposedAt, &o.ProposedDurationMinutes, &o.ProposedPricePerMinute, &o.TotalOfferAmount, &o.Message, &o.Status, &o.HostMessage, &o.HostCounterPrice, &o.IsPrepaid, &o.PrepaidAmount, &o.Latitude, &o.Longitude, &o.LocationName)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (r *offerRepository) UpdateUserOfferStatus(ctx context.Context, id domain.UUID, status string, convertedBookingID *domain.UUID) error {
	query := `UPDATE user_offers SET status = $1, converted_booking_id = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, convertedBookingID, id)
	return err
}

func (r *offerRepository) ListHostOffers(ctx context.Context, hostID domain.UUID, status string) ([]*domain.HostOffer, error) {
	query := `SELECT id, title, final_price_per_minute, status FROM host_offers WHERE host_id = $1`
	if status != "" {
		query += " AND status = $2"
	}
	rows, err := r.db.Query(ctx, query, hostID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.HostOffer
	for rows.Next() {
		o := &domain.HostOffer{}
		err := rows.Scan(&o.ID, &o.Title, &o.FinalPricePerMinute, &o.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, nil
}

func (r *offerRepository) SearchHostOffers(ctx context.Context, filters map[string]interface{}) ([]*domain.HostOffer, error) {
	query := `SELECT id, host_id, title, final_price_per_minute, thumbnail_url FROM host_offers WHERE status = 'active'`
	// Simplified search logic
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.HostOffer
	for rows.Next() {
		o := &domain.HostOffer{}
		err := rows.Scan(&o.ID, &o.HostID, &o.Title, &o.FinalPricePerMinute, &o.ThumbnailURL)
		if err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, nil
}

func (r *offerRepository) ListIncomingUserOffers(ctx context.Context, hostID domain.UUID) ([]*domain.UserOffer, error) {
	query := `SELECT id, offer_code, user_id, proposed_at, total_offer_amount, status FROM user_offers WHERE host_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.UserOffer
	for rows.Next() {
		o := &domain.UserOffer{}
		err := rows.Scan(&o.ID, &o.OfferCode, &o.UserID, &o.ProposedAt, &o.TotalOfferAmount, &o.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, nil
}

func (r *offerRepository) ListOutgoingUserOffers(ctx context.Context, userID domain.UUID) ([]*domain.UserOffer, error) {
	query := `SELECT id, offer_code, host_id, proposed_at, total_offer_amount, status FROM user_offers WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.UserOffer
	for rows.Next() {
		o := &domain.UserOffer{}
		err := rows.Scan(&o.ID, &o.OfferCode, &o.HostID, &o.ProposedAt, &o.TotalOfferAmount, &o.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, nil
}
