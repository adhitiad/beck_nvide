package domain

import (
	"context"
	"time"
)

// Booking Statuses
const (
	BookingPending        = "pending"
	BookingConfirmed      = "confirmed"
	BookingActive         = "active"
	BookingCompleted      = "completed"
	BookingCancelled      = "cancelled"
	BookingRejected       = "host_rejected"
	BookingCounterOffered = "counter_offered"
	BookingSettled        = "settled"
)

// Booking Types
const (
	BookingTypeChat   = "chat_session"
	BookingTypeVoice  = "voice_call"
	BookingTypeVideo  = "video_call"
	BookingTypeCustom = "custom_request"
)

// HostSchedule template
type HostSchedule struct {
	ID                  UUID      `json:"id" db:"id"`
	HostID              UUID      `json:"host_id" db:"host_id"`
	DayOfWeek           int       `json:"day_of_week" db:"day_of_week"` // 0-6
	StartTime           string    `json:"start_time" db:"start_time"`
	EndTime             string    `json:"end_time" db:"end_time"`
	SlotDurationMinutes int       `json:"slot_duration_minutes" db:"slot_duration_minutes"`
	IsActive            bool      `json:"is_active" db:"is_active"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// HostScheduleException
type HostScheduleException struct {
	ID            UUID       `json:"id" db:"id"`
	HostID        UUID       `json:"host_id" db:"host_id"`
	ExceptionDate time.Time  `json:"exception_date" db:"exception_date"`
	Type          string     `json:"type" db:"type"` // "unavailable", "special_hours"
	StartTime     *string    `json:"start_time,omitempty" db:"start_time"`
	EndTime       *string    `json:"end_time,omitempty" db:"end_time"`
	Reason        string     `json:"reason,omitempty" db:"reason"`
}

// HostBookingType
type HostBookingType struct {
	ID                    UUID      `json:"id" db:"id"`
	HostID                UUID      `json:"host_id" db:"host_id"`
	Type                  string    `json:"type" db:"type"`
	Name                  string    `json:"name" db:"name"`
	Description           string    `json:"description" db:"description"`
	DurationOptions       []int     `json:"duration_options" db:"duration_options"`
	PricePerMinute        float64   `json:"price_per_minute" db:"price_per_minute"`
	MinDuration           int       `json:"min_duration" db:"min_duration"`
	MaxDuration           int       `json:"max_duration" db:"max_duration"`
	IsActive              bool      `json:"is_active" db:"is_active"`
	AllowExtend           bool      `json:"allow_extend" db:"allow_extend"`
	ExtendPricePerMinute  *float64  `json:"extend_price_per_minute,omitempty" db:"extend_price_per_minute"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// Booking record
type Booking struct {
	ID              UUID      `json:"id" db:"id"`
	BookingCode     string    `json:"booking_code" db:"booking_code"`
	HostID          UUID      `json:"host_id" db:"host_id"`
	UserID          UUID      `json:"user_id" db:"user_id"`
	BookingTypeID   UUID      `json:"booking_type_id" db:"booking_type_id"`
	ScheduledAt     time.Time `json:"scheduled_at" db:"scheduled_at"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	EndedAt         time.Time `json:"ended_at" db:"ended_at"`

	BasePrice    float64 `json:"base_price" db:"base_price"`
	PlatformFee  float64 `json:"platform_fee" db:"platform_fee"`
	ProcessingFee float64 `json:"processing_fee" db:"processing_fee"`
	TaxFee       float64 `json:"tax_fee" db:"tax_fee"`
	AgencyFee    float64 `json:"agency_fee" db:"agency_fee"`
	TotalPrice   float64 `json:"total_price" db:"total_price"`
	HostEarning  float64 `json:"host_earning" db:"host_earning"`

	Status        string     `json:"status" db:"status"`
	PaymentStatus string     `json:"payment_status" db:"payment_status"`
	RoomID        *UUID      `json:"room_id,omitempty" db:"room_id"`
	JoinToken     string     `json:"join_token,omitempty" db:"join_token"`
	
	UserNotes     string     `json:"user_notes,omitempty" db:"user_notes"`
	
	MeetingLatitude         *float64 `json:"meeting_latitude,omitempty" db:"meeting_latitude"`
	MeetingLongitude        *float64 `json:"meeting_longitude,omitempty" db:"meeting_longitude"`
	MeetingLocationName     string   `json:"meeting_location_name,omitempty" db:"meeting_location_name"`
	IsRealtimeTrackingActive bool     `json:"is_realtime_tracking_active" db:"is_realtime_tracking_active"`
	
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// BookingRepository
type BookingRepository interface {
	// Schedule
	SetSchedule(ctx context.Context, schedules []*HostSchedule) error
	GetSchedule(ctx context.Context, hostID UUID) ([]*HostSchedule, error)
	AddException(ctx context.Context, ex *HostScheduleException) error
	GetExceptions(ctx context.Context, hostID UUID, start, end time.Time) ([]*HostScheduleException, error)
	
	// Types
	CreateBookingType(ctx context.Context, bt *HostBookingType) error
	GetBookingTypes(ctx context.Context, hostID UUID) ([]*HostBookingType, error)
	
	// Booking
	CreateBooking(ctx context.Context, b *Booking) error
	GetBookingByID(ctx context.Context, id UUID) (*Booking, error)
	UpdateBookingStatus(ctx context.Context, id UUID, status string) error
	ListBookingsByHost(ctx context.Context, hostID UUID, status string) ([]*Booking, error)
	ListBookingsByUser(ctx context.Context, userID UUID, status string) ([]*Booking, error)
	CheckOverlap(ctx context.Context, hostID UUID, start, end time.Time) (bool, error)
}

// BookingUsecase
type BookingUsecase interface {
	// Schedule
	SetSchedule(ctx context.Context, schedules []*HostSchedule) error
	GetAvailableSlots(ctx context.Context, hostID UUID, date time.Time) ([]map[string]interface{}, error)
	
	// Booking Types
	SetBookingType(ctx context.Context, bt *HostBookingType) error
	GetBookingTypes(ctx context.Context, hostID UUID) ([]*HostBookingType, error)

	// Booking Lifecycle
	RequestBooking(ctx context.Context, userID, hostID, typeID UUID, scheduledAt time.Time, duration int, notes string, lat, lon *float64, locName string) (*Booking, error)
	AcceptBooking(ctx context.Context, hostID, bookingID UUID) error
	RejectBooking(ctx context.Context, hostID, bookingID UUID, reason string) error
	CancelBooking(ctx context.Context, userID, bookingID UUID, reason string) error
	ListBookingsByHost(ctx context.Context, hostID UUID, status string) ([]*Booking, error)
	ListBookingsByUser(ctx context.Context, userID UUID, status string) ([]*Booking, error)
	
	// Execution
	StartSession(ctx context.Context, bookingID UUID) (string, error) // Returns join token
	EndSession(ctx context.Context, bookingID UUID) error
}
