package domain

import (
	"context"
	"time"
)

type LiveSchedule struct {
	ID                      UUID       `json:"id"`
	HostID                  UUID       `json:"host_id"`
	Title                   string     `json:"title"`
	Description             string     `json:"description"`
	Category                string     `json:"category"`
	ThumbnailURL            string     `json:"thumbnail_url"`
	ScheduleType            string     `json:"schedule_type"` // "one_time", "recurring"
	ScheduledAt             *time.Time `json:"scheduled_at,omitempty"`
	RecurrenceRule          string     `json:"recurrence_rule,omitempty"`
	RecurrenceStartDate     *time.Time `json:"recurrence_start_date,omitempty"`
	RecurrenceEndDate       *time.Time `json:"recurrence_end_date,omitempty"`
	RecurrenceTime          string     `json:"recurrence_time,omitempty"`
	Timezone                string     `json:"timezone"`
	ExpectedDurationMinutes int        `json:"expected_duration_minutes"`
	Status                  string     `json:"status"` // "scheduled", "live", "completed", "cancelled", "expired"
	IsCancelled             bool       `json:"is_cancelled"`
	CancelledAt             *time.Time `json:"cancelled_at,omitempty"`
	CancellationReason      string     `json:"cancellation_reason,omitempty"`
	ActualStreamID          *UUID      `json:"actual_stream_id,omitempty"`
	WentLiveAt              *time.Time `json:"went_live_at,omitempty"`
	MaxWaitRoomUsers        int        `json:"max_wait_room_users"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type LiveScheduleOccurrence struct {
	ID                UUID       `json:"id"`
	ScheduleID        UUID       `json:"schedule_id"`
	HostID            UUID       `json:"host_id"`
	OccurrenceDate    time.Time  `json:"occurrence_date"`
	OccurrenceStartAt time.Time  `json:"occurrence_start_at"`
	OccurrenceEndAt   *time.Time `json:"occurrence_end_at,omitempty"`
	Status            string     `json:"status"` // "upcoming", "live", "completed", "missed", "cancelled"
	ActualStreamID    *UUID      `json:"actual_stream_id,omitempty"`
	WaitRoomOpenedAt  *time.Time `json:"wait_room_opened_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`

	// Rich join info (not in schema, but for delivery)
	ScheduleTitle string `json:"schedule_title,omitempty"`
	ScheduleDesc  string `json:"schedule_description,omitempty"`
	HostUsername  string `json:"host_username,omitempty"`
	HostAvatar    string `json:"host_avatar,omitempty"`
}

type UserScheduleReminder struct {
	ID               UUID       `json:"id"`
	UserID           UUID       `json:"user_id"`
	ScheduleID       UUID       `json:"schedule_id"`
	Remind24h        bool       `json:"remind_24h"`
	Remind1h         bool       `json:"remind_1h"`
	Remind15m        bool       `json:"remind_15m"`
	RemindLiveStart  bool       `json:"remind_live_start"`
	PushEnabled      bool       `json:"push_enabled"`
	EmailEnabled     bool       `json:"email_enabled"`
	SMSEnabled       bool       `json:"sms_enabled"`
	IsActive         bool       `json:"is_active"`
	UnsubscribedAt   *time.Time `json:"unsubscribed_at,omitempty"`
	JoinedWaitRoomAt *time.Time `json:"joined_wait_room_at,omitempty"`
	LeftWaitRoomAt   *time.Time `json:"left_wait_room_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type ReminderLog struct {
	ID           UUID       `json:"id"`
	ReminderID   UUID       `json:"reminder_id"`
	ReminderType string     `json:"reminder_type"` // "24h", "1h", "15m", "live_start"
	Channel      string     `json:"channel"`       // "push", "ws", "email", "sms"
	SentAt       time.Time  `json:"sent_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	OpenedAt     *time.Time `json:"opened_at,omitempty"`
	IsSuccess    bool       `json:"is_success"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type WaitRoom struct {
	ID               UUID       `json:"id"`
	OccurrenceID     UUID       `json:"occurrence_id"`
	HostID           UUID       `json:"host_id"`
	Status           string     `json:"status"` // "waiting", "opening_soon", "live_started", "closed"
	OpenedAt         *time.Time `json:"opened_at,omitempty"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
	CurrentUserCount int        `json:"current_user_count"`
	PeakUserCount    int        `json:"peak_user_count"`
	CreatedAt        time.Time  `json:"created_at"`
}

type WaitRoomMessage struct {
	ID          UUID      `json:"id"`
	WaitRoomID  UUID      `json:"wait_room_id"`
	UserID      UUID      `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	UserLevel   int       `json:"user_level,omitempty"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"` // "text", "system", "gift_pledge"
	CreatedAt   time.Time `json:"created_at"`
}

type HostScheduleStat struct {
	ID                       UUID       `json:"id"`
	HostID                   UUID       `json:"host_id"`
	ScheduleID               *UUID      `json:"schedule_id,omitempty"`
	OccurrenceID             *UUID      `json:"occurrence_id,omitempty"`
	TotalRemindersSent       int        `json:"total_reminders_sent"`
	TotalRemindersOpened     int        `json:"total_reminders_opened"`
	WaitRoomJoined           int        `json:"wait_room_joined"`
	WaitRoomToLiveConversion int        `json:"wait_room_to_live_conversion"`
	LiveStartViewers         int        `json:"live_start_viewers"`
	ScheduledAt              *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
}

type LiveScheduleRepository interface {
	Create(ctx context.Context, schedule *LiveSchedule) error
	Update(ctx context.Context, schedule *LiveSchedule) error
	GetByID(ctx context.Context, id UUID) (*LiveSchedule, error)
	Delete(ctx context.Context, id UUID) error

	CreateOccurrence(ctx context.Context, occ *LiveScheduleOccurrence) error
	GetOccurrenceByID(ctx context.Context, id UUID) (*LiveScheduleOccurrence, error)
	GetOccurrenceByScheduleAndDate(ctx context.Context, scheduleID UUID, date string) (*LiveScheduleOccurrence, error)
	UpdateOccurrence(ctx context.Context, occ *LiveScheduleOccurrence) error
	CancelFutureOccurrences(ctx context.Context, scheduleID UUID) error
	CancelSingleOccurrence(ctx context.Context, occID UUID) error
	GetActiveSchedulesCount(ctx context.Context, hostID UUID) (int, error)
	CheckOverlap(ctx context.Context, hostID UUID, start, end time.Time, excludeID UUID) (bool, error)

	SubscribeReminder(ctx context.Context, reminder *UserScheduleReminder) error
	UnsubscribeReminder(ctx context.Context, userID, scheduleID UUID) error
	GetReminder(ctx context.Context, userID, scheduleID UUID) (*UserScheduleReminder, error)
	ListUserReminders(ctx context.Context, userID UUID) ([]*LiveScheduleOccurrence, error)
	LogReminder(ctx context.Context, log *ReminderLog) error
	GetUpcomingOccurrencesForReminder(ctx context.Context, withinMinutes int) ([]*LiveScheduleOccurrence, error)
	GetSubscribersForSchedule(ctx context.Context, scheduleID UUID) ([]UUID, error)

	CreateWaitRoom(ctx context.Context, room *WaitRoom) error
	GetWaitRoomByOccurrenceID(ctx context.Context, occID UUID) (*WaitRoom, error)
	UpdateWaitRoom(ctx context.Context, room *WaitRoom) error
	SaveWaitRoomMessage(ctx context.Context, msg *WaitRoomMessage) error
	GetWaitRoomMessages(ctx context.Context, waitRoomID UUID, limit int) ([]*WaitRoomMessage, error)

	GetNextSchedule(ctx context.Context, hostID UUID) (*LiveScheduleOccurrence, error)
	GetUpcomingFeed(ctx context.Context, followerHostIDs []UUID, category string, limit, offset int) ([]*LiveScheduleOccurrence, error)
	GetTrendingSchedules(ctx context.Context, limit int) ([]*LiveScheduleOccurrence, error)
	GetScheduleStats(ctx context.Context, hostID UUID) ([]*HostScheduleStat, error)
	LinkStreamToOccurrence(ctx context.Context, occurrenceID UUID, streamID UUID) error
	MarkMissedOccurrences(ctx context.Context) error
	GetUnprocessedOccurrences(ctx context.Context, hostID UUID) ([]*LiveScheduleOccurrence, error)

	// Clean architecture additions to avoid direct usecase-db dependencies
	GetActiveRecurringSchedules(ctx context.Context) ([]*LiveSchedule, error)
	GetUsernameByID(ctx context.Context, id UUID) (string, error)
	GetFollowedHostIDs(ctx context.Context, userID UUID) ([]UUID, error)
	GetUpcomingOccurrenceInWindow(ctx context.Context, hostID UUID, windowStart, windowEnd time.Time) (*LiveScheduleOccurrence, error)
}

type LiveScheduleUseCase interface {
	CreateSchedule(ctx context.Context, hostID UUID, schedule *LiveSchedule) error
	UpdateSchedule(ctx context.Context, hostID UUID, scheduleID UUID, schedule *LiveSchedule) error
	CancelSchedule(ctx context.Context, hostID UUID, scheduleID UUID) error
	CancelOccurrence(ctx context.Context, hostID UUID, occID UUID) error
	RefillAllOccurrences(ctx context.Context) error

	SubscribeReminder(ctx context.Context, userID, scheduleID UUID, req *UserScheduleReminder) error
	UnsubscribeReminder(ctx context.Context, userID, scheduleID UUID) error
	ListMyReminders(ctx context.Context, userID UUID) ([]*LiveScheduleOccurrence, error)

	GetNextSchedule(ctx context.Context, hostID UUID) (*LiveScheduleOccurrence, error)
	GetUpcomingFeed(ctx context.Context, userID UUID, category string, limit, offset int) ([]*LiveScheduleOccurrence, error)
	GetTrendingSchedules(ctx context.Context, limit int) ([]*LiveScheduleOccurrence, error)
	GetAnalytics(ctx context.Context, hostID UUID) ([]*HostScheduleStat, error)

	CheckAndAutoLinkStream(ctx context.Context, hostID UUID, streamID UUID) (*LiveScheduleOccurrence, error)
	CheckAndSendTieredReminders(ctx context.Context) error
	SnoozeReminder(ctx context.Context, reminderID UUID) error

	GetWaitRoomByOccurrence(ctx context.Context, occID UUID) (*WaitRoom, error)
	SaveWaitRoomPledge(ctx context.Context, waitRoomID UUID, userID UUID, giftCode string, quantity int) error
}
