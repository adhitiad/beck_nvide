package domain

import (
	"context"
	"time"
)

// Offer Statuses
const (
	OfferActive      = "active"
	OfferExpired     = "expired"
	OfferFullyBooked = "fully_booked"
	
	UserOfferPending   = "pending"
	UserOfferAccepted  = "accepted"
	UserOfferRejected  = "rejected"
	UserOfferCountered = "countered"
)

// HostOffer (OB - Offer Book by Host)
type HostOffer struct {
	ID                   UUID      `json:"id" db:"id"`
	HostID               UUID      `json:"host_id" db:"host_id"`
	OfferCode            string    `json:"offer_code" db:"offer_code"`
	Title                string    `json:"title" db:"title"`
	Description          string    `json:"description" db:"description"`
	BookingTypeID        UUID      `json:"booking_type_id" db:"booking_type_id"`
	
	OfferMode            string    `json:"offer_mode" db:"offer_mode"` // "specific", "recurring"
	SpecificAt           *time.Time `json:"specific_at,omitempty" db:"specific_at"`
	RecurringDays        []int     `json:"recurring_days,omitempty" db:"recurring_days"`
	RecurringStartTime   string    `json:"recurring_start_time,omitempty" db:"recurring_start_time"`
	RecurringEndTime     string    `json:"recurring_end_time,omitempty" db:"recurring_end_time"`
	SlotDurationMinutes  int       `json:"slot_duration_minutes" db:"slot_duration_minutes"`
	
	BasePricePerMinute   float64   `json:"base_price_per_minute" db:"base_price_per_minute"`
	DiscountPercentage   float64   `json:"discount_percentage" db:"discount_percentage"`
	FinalPricePerMinute  float64   `json:"final_price_per_minute" db:"final_price_per_minute"`
	
	MaxBookings          int       `json:"max_bookings" db:"max_bookings"`
	BookingsMade         int       `json:"bookings_made" db:"bookings_made"`
	MaxBookingsPerUser   int       `json:"max_bookings_per_user" db:"max_bookings_per_user"`
	
	Status               string    `json:"status" db:"status"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsAutoConfirm        bool      `json:"is_auto_confirm" db:"is_auto_confirm"`
	
	Latitude             *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude            *float64  `json:"longitude,omitempty" db:"longitude"`
	LocationName         string    `json:"location_name,omitempty" db:"location_name"`
	ShareLocationType    string    `json:"share_location_type" db:"share_location_type"` // none, fixed, realtime
	
	Tags                 []string  `json:"tags,omitempty" db:"tags"`
	ThumbnailURL         string    `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// UserOffer (BO - Book Offer by User)
type UserOffer struct {
	ID                    UUID      `json:"id" db:"id"`
	OfferCode             string    `json:"offer_code" db:"offer_code"`
	UserID                UUID      `json:"user_id" db:"user_id"`
	HostID                UUID      `json:"host_id" db:"host_id"`
	
	BookingTypeID         *UUID     `json:"booking_type_id,omitempty" db:"booking_type_id"`
	OfferType             string    `json:"offer_type" db:"offer_type"` // "standard", "custom_price"
	
	ProposedAt            time.Time `json:"proposed_at" db:"proposed_at"`
	ProposedDurationMinutes int       `json:"proposed_duration_minutes" db:"proposed_duration_minutes"`
	ProposedPricePerMinute float64   `json:"proposed_price_per_minute,omitempty" db:"proposed_price_per_minute"`
	TotalOfferAmount      float64   `json:"total_offer_amount" db:"total_offer_amount"`
	
	Message               string    `json:"message,omitempty" db:"message"`
	Status                string    `json:"status" db:"status"`
	
	HostResponseAt        *time.Time `json:"host_response_at,omitempty" db:"host_response_at"`
	HostMessage           string    `json:"host_message,omitempty" db:"host_message"`
	HostCounterPrice      *float64  `json:"host_counter_price,omitempty" db:"host_counter_price"`
	HostCounterAt         *time.Time `json:"host_counter_at,omitempty" db:"host_counter_at"`
	
	ConvertedBookingID    *UUID     `json:"converted_booking_id,omitempty" db:"converted_booking_id"`
	IsPrepaid             bool      `json:"is_prepaid" db:"is_prepaid"`
	PrepaidAmount         float64   `json:"prepaid_amount" db:"prepaid_amount"`
	
	Latitude              *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude             *float64  `json:"longitude,omitempty" db:"longitude"`
	LocationName          string    `json:"location_name,omitempty" db:"location_name"`
	
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// OfferRepository
type OfferRepository interface {
	// Host Offers (OB)
	CreateHostOffer(ctx context.Context, offer *HostOffer) error
	GetHostOfferByID(ctx context.Context, id UUID) (*HostOffer, error)
	ListHostOffers(ctx context.Context, hostID UUID, status string) ([]*HostOffer, error)
	UpdateHostOfferBookings(ctx context.Context, id UUID, bookingsMade int) error
	SearchHostOffers(ctx context.Context, filters map[string]interface{}) ([]*HostOffer, error)
	
	// User Offers (BO)
	CreateUserOffer(ctx context.Context, offer *UserOffer) error
	GetUserOfferByID(ctx context.Context, id UUID) (*UserOffer, error)
	UpdateUserOfferStatus(ctx context.Context, id UUID, status string, convertedBookingID *UUID) error
	ListIncomingUserOffers(ctx context.Context, hostID UUID) ([]*UserOffer, error)
	ListOutgoingUserOffers(ctx context.Context, userID UUID) ([]*UserOffer, error)
}

// OfferUsecase
type OfferUsecase interface {
	// OB logic
	CreateOB(ctx context.Context, hostID UUID, req *HostOffer) (*HostOffer, error)
	BookOB(ctx context.Context, userID UUID, offerID UUID, slotStart time.Time) (*Booking, error)
	
	// BO logic
	CreateBO(ctx context.Context, userID UUID, hostID UUID, req *UserOffer) (*UserOffer, error)
	RespondToBO(ctx context.Context, hostID UUID, offerID UUID, action string, message string, counterPrice *float64) error
	AcceptBOCounter(ctx context.Context, userID UUID, offerID UUID) (*Booking, error)
	
	// Discovery
	GetOfferFeed(ctx context.Context, filters map[string]interface{}) ([]*HostOffer, error)
}
