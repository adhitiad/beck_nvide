package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type liveScheduleRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewLiveScheduleRepository(db *pgxpool.Pool, logger *zap.Logger) domain.LiveScheduleRepository {
	return &liveScheduleRepository{
		db:     db,
		logger: logger,
	}
}

// LiveSchedule queries
const selectScheduleFields = `
	id, host_id, title, description, category, thumbnail_url, schedule_type,
	scheduled_at, recurrence_rule, recurrence_start_date, recurrence_end_date, recurrence_time, timezone,
	expected_duration_minutes, status, is_cancelled, cancelled_at, cancellation_reason,
	actual_stream_id, went_live_at, max_wait_room_users, created_at, updated_at
`

func scanSchedule(row pgx.Row, s *domain.LiveSchedule) error {
	return row.Scan(
		&s.ID, &s.HostID, &s.Title, &s.Description, &s.Category, &s.ThumbnailURL, &s.ScheduleType,
		&s.ScheduledAt, &s.RecurrenceRule, &s.RecurrenceStartDate, &s.RecurrenceEndDate, &s.RecurrenceTime, &s.Timezone,
		&s.ExpectedDurationMinutes, &s.Status, &s.IsCancelled, &s.CancelledAt, &s.CancellationReason,
		&s.ActualStreamID, &s.WentLiveAt, &s.MaxWaitRoomUsers, &s.CreatedAt, &s.UpdatedAt,
	)
}

func (r *liveScheduleRepository) Create(ctx context.Context, s *domain.LiveSchedule) error {
	query := `
		INSERT INTO live_schedules (
			id, host_id, title, description, category, thumbnail_url, schedule_type,
			scheduled_at, recurrence_rule, recurrence_start_date, recurrence_end_date, recurrence_time, timezone,
			expected_duration_minutes, status, is_cancelled, cancelled_at, cancellation_reason,
			actual_stream_id, went_live_at, max_wait_room_users, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		s.ID, s.HostID, s.Title, s.Description, s.Category, s.ThumbnailURL, s.ScheduleType,
		s.ScheduledAt, s.RecurrenceRule, s.RecurrenceStartDate, s.RecurrenceEndDate, s.RecurrenceTime, s.Timezone,
		s.ExpectedDurationMinutes, s.Status, s.IsCancelled, s.CancelledAt, s.CancellationReason,
		s.ActualStreamID, s.WentLiveAt, s.MaxWaitRoomUsers,
	).Scan(&s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to create live schedule", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) Update(ctx context.Context, s *domain.LiveSchedule) error {
	query := `
		UPDATE live_schedules
		SET title = $1, description = $2, category = $3, thumbnail_url = $4, schedule_type = $5,
			scheduled_at = $6, recurrence_rule = $7, recurrence_start_date = $8, recurrence_end_date = $9, recurrence_time = $10, timezone = $11,
			expected_duration_minutes = $12, status = $13, is_cancelled = $14, cancelled_at = $15, cancellation_reason = $16,
			actual_stream_id = $17, went_live_at = $18, max_wait_room_users = $19, updated_at = NOW()
		WHERE id = $20
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		s.Title, s.Description, s.Category, s.ThumbnailURL, s.ScheduleType,
		s.ScheduledAt, s.RecurrenceRule, s.RecurrenceStartDate, s.RecurrenceEndDate, s.RecurrenceTime, s.Timezone,
		s.ExpectedDurationMinutes, s.Status, s.IsCancelled, s.CancelledAt, s.CancellationReason,
		s.ActualStreamID, s.WentLiveAt, s.MaxWaitRoomUsers, s.ID,
	).Scan(&s.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to update live schedule", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.LiveSchedule, error) {
	query := `SELECT ` + selectScheduleFields + ` FROM live_schedules WHERE id = $1`
	var s domain.LiveSchedule
	err := scanSchedule(r.db.QueryRow(ctx, query, id), &s)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get live schedule by ID", zap.Error(err))
		return nil, err
	}
	return &s, nil
}

func (r *liveScheduleRepository) Delete(ctx context.Context, id domain.UUID) error {
	// Soft delete / mark cancelled
	query := `UPDATE live_schedules SET status = 'cancelled', is_cancelled = true, cancelled_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete live schedule", zap.Error(err))
		return err
	}
	return nil
}

// LiveScheduleOccurrence queries
const selectOccurrenceFields = `
	o.id, o.schedule_id, o.host_id, o.occurrence_date, o.occurrence_start_at, o.occurrence_end_at,
	o.status, o.actual_stream_id, o.wait_room_opened_at, o.created_at
`

func scanOccurrence(row pgx.Row, o *domain.LiveScheduleOccurrence) error {
	return row.Scan(
		&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
		&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
	)
}

func (r *liveScheduleRepository) CreateOccurrence(ctx context.Context, o *domain.LiveScheduleOccurrence) error {
	query := `
		INSERT INTO schedule_occurrences (
			id, schedule_id, host_id, occurrence_date, occurrence_start_at, occurrence_end_at,
			status, actual_stream_id, wait_room_opened_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (schedule_id, occurrence_date) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query,
		o.ID, o.ScheduleID, o.HostID, o.OccurrenceDate, o.OccurrenceStartAt, o.OccurrenceEndAt,
		o.Status, o.ActualStreamID, o.WaitRoomOpenedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create occurrence", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) GetOccurrenceByID(ctx context.Context, id domain.UUID) (*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM schedule_occurrences o
		JOIN live_schedules s ON o.schedule_id = s.id
		JOIN users u ON o.host_id = u.id
		WHERE o.id = $1
	`
	var o domain.LiveScheduleOccurrence
	err := r.db.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
		&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
		&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get occurrence by ID", zap.Error(err))
		return nil, err
	}
	return &o, nil
}

func (r *liveScheduleRepository) GetOccurrenceByScheduleAndDate(ctx context.Context, scheduleID domain.UUID, date string) (*domain.LiveScheduleOccurrence, error) {
	query := `SELECT ` + selectOccurrenceFields + ` FROM schedule_occurrences o WHERE o.schedule_id = $1 AND o.occurrence_date = $2`
	var o domain.LiveScheduleOccurrence
	err := scanOccurrence(r.db.QueryRow(ctx, query, scheduleID, date), &o)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *liveScheduleRepository) UpdateOccurrence(ctx context.Context, o *domain.LiveScheduleOccurrence) error {
	query := `
		UPDATE schedule_occurrences
		SET status = $1, occurrence_start_at = $2, occurrence_end_at = $3,
			actual_stream_id = $4, wait_room_opened_at = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(ctx, query,
		o.Status, o.OccurrenceStartAt, o.OccurrenceEndAt,
		o.ActualStreamID, o.WaitRoomOpenedAt, o.ID,
	)
	if err != nil {
		r.logger.Error("Failed to update occurrence", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) CancelFutureOccurrences(ctx context.Context, scheduleID domain.UUID) error {
	query := `
		UPDATE schedule_occurrences
		SET status = 'cancelled'
		WHERE schedule_id = $1 AND status = 'upcoming' AND occurrence_start_at > NOW()
	`
	_, err := r.db.Exec(ctx, query, scheduleID)
	if err != nil {
		r.logger.Error("Failed to cancel future occurrences", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) CancelSingleOccurrence(ctx context.Context, occID domain.UUID) error {
	query := `
		UPDATE schedule_occurrences
		SET status = 'cancelled'
		WHERE id = $1 AND status = 'upcoming'
	`
	_, err := r.db.Exec(ctx, query, occID)
	if err != nil {
		r.logger.Error("Failed to cancel single occurrence", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) GetActiveSchedulesCount(ctx context.Context, hostID domain.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM live_schedules WHERE host_id = $1 AND status = 'scheduled'`
	var count int
	err := r.db.QueryRow(ctx, query, hostID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *liveScheduleRepository) CheckOverlap(ctx context.Context, hostID domain.UUID, start, end time.Time, excludeID domain.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM schedule_occurrences
			WHERE host_id = $1 AND status = 'upcoming' AND id != $2
			AND (
				(occurrence_start_at - INTERVAL '30 minutes' <= $3 AND occurrence_start_at + (COALESCE(occurrence_end_at - occurrence_start_at, INTERVAL '60 minutes') + INTERVAL '30 minutes') >= $3)
				OR (occurrence_start_at - INTERVAL '30 minutes' <= $4 AND occurrence_start_at + (COALESCE(occurrence_end_at - occurrence_start_at, INTERVAL '60 minutes') + INTERVAL '30 minutes') >= $4)
				OR (occurrence_start_at >= $3 AND occurrence_start_at <= $4)
			)
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, hostID, excludeID, start, end).Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check overlap", zap.Error(err))
		return false, err
	}
	return exists, nil
}

// User reminders
func (r *liveScheduleRepository) SubscribeReminder(ctx context.Context, rem *domain.UserScheduleReminder) error {
	query := `
		INSERT INTO user_schedule_reminders (
			id, user_id, schedule_id, remind_24h, remind_1h, remind_15m, remind_live_start,
			push_enabled, email_enabled, sms_enabled, is_active, unsubscribed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (user_id, schedule_id) DO UPDATE SET
			remind_24h = EXCLUDED.remind_24h,
			remind_1h = EXCLUDED.remind_1h,
			remind_15m = EXCLUDED.remind_15m,
			remind_live_start = EXCLUDED.remind_live_start,
			push_enabled = EXCLUDED.push_enabled,
			is_active = true,
			unsubscribed_at = NULL
	`
	_, err := r.db.Exec(ctx, query,
		rem.ID, rem.UserID, rem.ScheduleID, rem.Remind24h, rem.Remind1h, rem.Remind15m, rem.RemindLiveStart,
		rem.PushEnabled, rem.EmailEnabled, rem.SMSEnabled, rem.IsActive, rem.UnsubscribedAt,
	)
	if err != nil {
		r.logger.Error("Failed to subscribe reminder", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) UnsubscribeReminder(ctx context.Context, userID, scheduleID domain.UUID) error {
	query := `
		UPDATE user_schedule_reminders
		SET is_active = false, unsubscribed_at = NOW()
		WHERE user_id = $1 AND schedule_id = $2
	`
	_, err := r.db.Exec(ctx, query, userID, scheduleID)
	if err != nil {
		r.logger.Error("Failed to unsubscribe reminder", zap.Error(err))
		return err
	}
	return nil
}

func (r *liveScheduleRepository) GetReminder(ctx context.Context, userID, scheduleID domain.UUID) (*domain.UserScheduleReminder, error) {
	query := `
		SELECT id, user_id, schedule_id, remind_24h, remind_1h, remind_15m, remind_live_start,
			push_enabled, email_enabled, sms_enabled, is_active, unsubscribed_at, joined_wait_room_at, left_wait_room_at, created_at
		FROM user_schedule_reminders
		WHERE user_id = $1 AND schedule_id = $2
	`
	var rem domain.UserScheduleReminder
	err := r.db.QueryRow(ctx, query, userID, scheduleID).Scan(
		&rem.ID, &rem.UserID, &rem.ScheduleID, &rem.Remind24h, &rem.Remind1h, &rem.Remind15m, &rem.RemindLiveStart,
		&rem.PushEnabled, &rem.EmailEnabled, &rem.SMSEnabled, &rem.IsActive, &rem.UnsubscribedAt, &rem.JoinedWaitRoomAt, &rem.LeftWaitRoomAt, &rem.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rem, nil
}

func (r *liveScheduleRepository) ListUserReminders(ctx context.Context, userID domain.UUID) ([]*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM user_schedule_reminders r
		JOIN live_schedules s ON r.schedule_id = s.id
		JOIN schedule_occurrences o ON s.id = o.schedule_id
		JOIN users u ON o.host_id = u.id
		WHERE r.user_id = $1 AND r.is_active = true AND o.occurrence_start_at > NOW() AND o.status = 'upcoming'
		ORDER BY o.occurrence_start_at ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*domain.LiveScheduleOccurrence, 0)
	for rows.Next() {
		var o domain.LiveScheduleOccurrence
		err := rows.Scan(
			&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
			&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
			&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &o)
	}
	return list, nil
}

func (r *liveScheduleRepository) LogReminder(ctx context.Context, log *domain.ReminderLog) error {
	query := `
		INSERT INTO reminder_logs (id, reminder_id, reminder_type, channel, sent_at, delivered_at, opened_at, is_success, error_message)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		log.ID, log.ReminderID, log.ReminderType, log.Channel, log.DeliveredAt, log.OpenedAt, log.IsSuccess, log.ErrorMessage,
	)
	return err
}

func (r *liveScheduleRepository) GetUpcomingOccurrencesForReminder(ctx context.Context, withinMinutes int) ([]*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM schedule_occurrences o
		JOIN live_schedules s ON o.schedule_id = s.id
		JOIN users u ON o.host_id = u.id
		WHERE o.status = 'upcoming' AND o.occurrence_start_at BETWEEN NOW() AND NOW() + ($1 * INTERVAL '1 minute')
	`
	rows, err := r.db.Query(ctx, query, withinMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*domain.LiveScheduleOccurrence, 0)
	for rows.Next() {
		var o domain.LiveScheduleOccurrence
		err := rows.Scan(
			&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
			&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
			&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &o)
	}
	return list, nil
}

func (r *liveScheduleRepository) GetSubscribersForSchedule(ctx context.Context, scheduleID domain.UUID) ([]domain.UUID, error) {
	query := `SELECT user_id FROM user_schedule_reminders WHERE schedule_id = $1 AND is_active = true`
	rows, err := r.db.Query(ctx, query, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []domain.UUID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, domain.UUID(id))
		}
	}
	return ids, nil
}

// Wait room
func (r *liveScheduleRepository) CreateWaitRoom(ctx context.Context, wr *domain.WaitRoom) error {
	query := `
		INSERT INTO wait_rooms (id, occurrence_id, host_id, status, opened_at, closed_at, current_user_count, peak_user_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (occurrence_id) DO UPDATE SET
			status = EXCLUDED.status,
			current_user_count = EXCLUDED.current_user_count,
			peak_user_count = GREATEST(wait_rooms.peak_user_count, EXCLUDED.current_user_count)
	`
	_, err := r.db.Exec(ctx, query,
		wr.ID, wr.OccurrenceID, wr.HostID, wr.Status, wr.OpenedAt, wr.ClosedAt, wr.CurrentUserCount, wr.PeakUserCount,
	)
	return err
}

func (r *liveScheduleRepository) GetWaitRoomByOccurrenceID(ctx context.Context, occID domain.UUID) (*domain.WaitRoom, error) {
	query := `SELECT id, occurrence_id, host_id, status, opened_at, closed_at, current_user_count, peak_user_count, created_at FROM wait_rooms WHERE occurrence_id = $1`
	var wr domain.WaitRoom
	err := r.db.QueryRow(ctx, query, occID).Scan(
		&wr.ID, &wr.OccurrenceID, &wr.HostID, &wr.Status, &wr.OpenedAt, &wr.ClosedAt, &wr.CurrentUserCount, &wr.PeakUserCount, &wr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &wr, nil
}

func (r *liveScheduleRepository) UpdateWaitRoom(ctx context.Context, wr *domain.WaitRoom) error {
	query := `
		UPDATE wait_rooms
		SET status = $1, opened_at = $2, closed_at = $3, current_user_count = $4,
			peak_user_count = GREATEST(peak_user_count, $4)
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query, wr.Status, wr.OpenedAt, wr.ClosedAt, wr.CurrentUserCount, wr.ID)
	return err
}

func (r *liveScheduleRepository) SaveWaitRoomMessage(ctx context.Context, m *domain.WaitRoomMessage) error {
	query := `
		INSERT INTO wait_room_messages (id, wait_room_id, user_id, content, message_type, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err := r.db.Exec(ctx, query, m.ID, m.WaitRoomID, m.UserID, m.Content, m.MessageType)
	return err
}

func (r *liveScheduleRepository) GetWaitRoomMessages(ctx context.Context, waitRoomID domain.UUID, limit int) ([]*domain.WaitRoomMessage, error) {
	query := `
		SELECT wm.id, wm.wait_room_id, wm.user_id, u.username, u.user_level, wm.content, wm.message_type, wm.created_at
		FROM wait_room_messages wm
		JOIN users u ON wm.user_id = u.id
		WHERE wm.wait_room_id = $1
		ORDER BY wm.created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, waitRoomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.WaitRoomMessage
	for rows.Next() {
		var m domain.WaitRoomMessage
		err := rows.Scan(
			&m.ID, &m.WaitRoomID, &m.UserID, &m.Username, &m.UserLevel, &m.Content, &m.MessageType, &m.CreatedAt,
		)
		if err == nil {
			list = append(list, &m)
		}
	}
	return list, nil
}

// Discovery and stats
func (r *liveScheduleRepository) GetNextSchedule(ctx context.Context, hostID domain.UUID) (*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM schedule_occurrences o
		JOIN live_schedules s ON o.schedule_id = s.id
		JOIN users u ON o.host_id = u.id
		WHERE o.host_id = $1 AND o.status = 'upcoming' AND o.occurrence_start_at > NOW()
		ORDER BY o.occurrence_start_at ASC
		LIMIT 1
	`
	var o domain.LiveScheduleOccurrence
	err := r.db.QueryRow(ctx, query, hostID).Scan(
		&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
		&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
		&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *liveScheduleRepository) GetUpcomingFeed(ctx context.Context, followerHostIDs []domain.UUID, category string, limit, offset int) ([]*domain.LiveScheduleOccurrence, error) {
	// If no follows, just pull popular ones
	var rows pgx.Rows
	var err error

	if len(followerHostIDs) > 0 {
		query := `
			SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
			FROM schedule_occurrences o
			JOIN live_schedules s ON o.schedule_id = s.id
			JOIN users u ON o.host_id = u.id
			WHERE o.status = 'upcoming' AND o.occurrence_start_at > NOW()
			AND o.host_id = ANY($1)
			AND ($2 = '' OR s.category = $2)
			ORDER BY o.occurrence_start_at ASC
			LIMIT $3 OFFSET $4
		`
		rows, err = r.db.Query(ctx, query, followerHostIDs, category, limit, offset)
	} else {
		query := `
			SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
			FROM schedule_occurrences o
			JOIN live_schedules s ON o.schedule_id = s.id
			JOIN users u ON o.host_id = u.id
			WHERE o.status = 'upcoming' AND o.occurrence_start_at > NOW()
			AND ($1 = '' OR s.category = $1)
			ORDER BY o.occurrence_start_at ASC
			LIMIT $2 OFFSET $3
		`
		rows, err = r.db.Query(ctx, query, category, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LiveScheduleOccurrence
	for rows.Next() {
		var o domain.LiveScheduleOccurrence
		err := rows.Scan(
			&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
			&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
			&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
		)
		if err == nil {
			list = append(list, &o)
		}
	}
	return list, nil
}

func (r *liveScheduleRepository) GetTrendingSchedules(ctx context.Context, limit int) ([]*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM schedule_occurrences o
		JOIN live_schedules s ON o.schedule_id = s.id
		JOIN users u ON o.host_id = u.id
		LEFT JOIN user_schedule_reminders usr ON s.id = usr.schedule_id AND usr.is_active = true
		WHERE o.status = 'upcoming' AND o.occurrence_start_at > NOW()
		GROUP BY o.id, s.id, u.id
		ORDER BY COUNT(usr.id) DESC, o.occurrence_start_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LiveScheduleOccurrence
	for rows.Next() {
		var o domain.LiveScheduleOccurrence
		err := rows.Scan(
			&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
			&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
			&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
		)
		if err == nil {
			list = append(list, &o)
		}
	}
	return list, nil
}

func (r *liveScheduleRepository) GetScheduleStats(ctx context.Context, hostID domain.UUID) ([]*domain.HostScheduleStat, error) {
	query := `
		SELECT id, host_id, schedule_id, occurrence_id, total_reminders_sent, total_reminders_opened,
			wait_room_joined, wait_room_to_live_conversion, live_start_viewers, scheduled_at, created_at
		FROM host_schedule_stats
		WHERE host_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.HostScheduleStat
	for rows.Next() {
		var s domain.HostScheduleStat
		err := rows.Scan(
			&s.ID, &s.HostID, &s.ScheduleID, &s.OccurrenceID, &s.TotalRemindersSent, &s.TotalRemindersOpened,
			&s.WaitRoomJoined, &s.WaitRoomToLiveConversion, &s.LiveStartViewers, &s.ScheduledAt, &s.CreatedAt,
		)
		if err == nil {
			list = append(list, &s)
		}
	}
	return list, nil
}

func (r *liveScheduleRepository) LinkStreamToOccurrence(ctx context.Context, occurrenceID domain.UUID, streamID domain.UUID) error {
	query := `
		UPDATE schedule_occurrences
		SET status = 'live', actual_stream_id = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, streamID, occurrenceID)
	if err != nil {
		return err
	}

	// Update the schedule table too
	querySched := `
		UPDATE live_schedules ls
		SET status = 'live', actual_stream_id = $1, went_live_at = NOW()
		FROM schedule_occurrences o
		WHERE o.schedule_id = ls.id AND o.id = $2
	`
	_, err = r.db.Exec(ctx, querySched, streamID, occurrenceID)
	return err
}

func (r *liveScheduleRepository) MarkMissedOccurrences(ctx context.Context) error {
	// If occurrence is more than 30 minutes in the past and still 'upcoming', mark as missed
	query := `
		UPDATE schedule_occurrences
		SET status = 'missed'
		WHERE status = 'upcoming' AND occurrence_start_at < NOW() - INTERVAL '30 minutes'
	`
	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *liveScheduleRepository) GetUnprocessedOccurrences(ctx context.Context, hostID domain.UUID) ([]*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT ` + selectOccurrenceFields + `, s.title, s.description, u.username, u.avatar_url
		FROM schedule_occurrences o
		JOIN live_schedules s ON o.schedule_id = s.id
		JOIN users u ON o.host_id = u.id
		WHERE o.host_id = $1 AND o.status = 'upcoming'
		ORDER BY o.occurrence_start_at ASC
	`
	rows, err := r.db.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LiveScheduleOccurrence
	for rows.Next() {
		var o domain.LiveScheduleOccurrence
		err := rows.Scan(
			&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
			&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
			&o.ScheduleTitle, &o.ScheduleDesc, &o.HostUsername, &o.HostAvatar,
		)
		if err == nil {
			list = append(list, &o)
		}
	}
	return list, nil
}

func (r *liveScheduleRepository) GetActiveRecurringSchedules(ctx context.Context) ([]*domain.LiveSchedule, error) {
	query := `
		SELECT id, host_id, title, description, category, thumbnail_url, schedule_type,
			recurrence_rule, recurrence_start_date, recurrence_end_date, recurrence_time, timezone,
			expected_duration_minutes, status
		FROM live_schedules
		WHERE schedule_type = 'recurring' AND status = 'scheduled'
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("Failed to fetch active recurring schedules", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LiveSchedule
	for rows.Next() {
		var s domain.LiveSchedule
		err := rows.Scan(
			&s.ID, &s.HostID, &s.Title, &s.Description, &s.Category, &s.ThumbnailURL, &s.ScheduleType,
			&s.RecurrenceRule, &s.RecurrenceStartDate, &s.RecurrenceEndDate, &s.RecurrenceTime, &s.Timezone,
			&s.ExpectedDurationMinutes, &s.Status,
		)
		if err == nil {
			list = append(list, &s)
		}
	}
	return list, nil
}

func (r *liveScheduleRepository) GetUsernameByID(ctx context.Context, id domain.UUID) (string, error) {
	var username string
	err := r.db.QueryRow(ctx, "SELECT username FROM users WHERE id = $1", id).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}

func (r *liveScheduleRepository) GetFollowedHostIDs(ctx context.Context, userID domain.UUID) ([]domain.UUID, error) {
	query := "SELECT target_id FROM user_blocks WHERE blocker_id = $1"
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []domain.UUID
	for rows.Next() {
		var hid string
		if err := rows.Scan(&hid); err == nil {
			uID, _ := domain.FromString(hid)
			ids = append(ids, uID)
		}
	}
	return ids, nil
}

func (r *liveScheduleRepository) GetUpcomingOccurrenceInWindow(ctx context.Context, hostID domain.UUID, windowStart, windowEnd time.Time) (*domain.LiveScheduleOccurrence, error) {
	query := `
		SELECT o.id, o.schedule_id, o.host_id, o.occurrence_date, o.occurrence_start_at, o.occurrence_end_at, o.status, o.actual_stream_id, o.wait_room_opened_at, o.created_at
		FROM schedule_occurrences o
		WHERE o.host_id = $1 AND o.status = 'upcoming'
		AND o.occurrence_start_at BETWEEN $2 AND $3
		LIMIT 1
	`
	var o domain.LiveScheduleOccurrence
	err := r.db.QueryRow(ctx, query, hostID, windowStart, windowEnd).Scan(
		&o.ID, &o.ScheduleID, &o.HostID, &o.OccurrenceDate, &o.OccurrenceStartAt, &o.OccurrenceEndAt,
		&o.Status, &o.ActualStreamID, &o.WaitRoomOpenedAt, &o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get upcoming occurrence in window", zap.Error(err))
		return nil, err
	}
	return &o, nil
}
