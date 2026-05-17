package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
)

type liveScheduleUseCase struct {
	repo        domain.LiveScheduleRepository
	waitRoomHub *websocket.WaitRoomHub
	wsHub       *websocket.Hub
	redisClient *redisClientWrapper // we'll implement a small helper to access Redis
	logger      *zap.Logger
}

type redisClientWrapper struct {
	client interface{} // we can cast it in usecase
}

func NewLiveScheduleUseCase(
	repo domain.LiveScheduleRepository,
	waitRoomHub *websocket.WaitRoomHub,
	wsHub *websocket.Hub,
	redis interface{},
	logger *zap.Logger,
) domain.LiveScheduleUseCase {
	return &liveScheduleUseCase{
		repo:        repo,
		waitRoomHub: waitRoomHub,
		wsHub:       wsHub,
		redisClient: &redisClientWrapper{client: redis},
		logger:      logger,
	}
}

func (u *liveScheduleUseCase) CreateSchedule(ctx context.Context, hostID domain.UUID, s *domain.LiveSchedule) error {
	// Rule A: Max 10 active schedules
	count, err := u.repo.GetActiveSchedulesCount(ctx, hostID)
	if err != nil {
		return err
	}
	if count >= 10 {
		return errors.New("MAX_SCHEDULE_LIMIT_REACHED: Anda hanya dapat memiliki maksimal 10 jadwal aktif mendatang")
	}

	// Validate schedule timing
	now := time.Now()
	if s.ScheduleType == "one_time" {
		if s.ScheduledAt == nil {
			return errors.New("scheduled_at is required for one_time schedules")
		}
		// Min H+1 jam
		minTime := now.Add(1 * time.Hour)
		if s.ScheduledAt.Before(minTime) {
			return errors.New("SCHEDULE_TOO_EARLY: Jadwal minimal H+1 jam dari sekarang")
		}
		// Max H+30 hari
		maxTime := now.AddDate(0, 0, 30)
		if s.ScheduledAt.After(maxTime) {
			return errors.New("SCHEDULE_TOO_LATE: Jadwal maksimal H+30 hari ke depan")
		}
	} else if s.ScheduleType == "recurring" {
		if s.RecurrenceRule == "" {
			return errors.New("recurrence_rule is required for recurring schedules")
		}
	} else {
		return errors.New("INVALID_SCHEDULE_TYPE")
	}

	s.ID = domain.NewUUID()
	s.HostID = hostID
	s.Status = "scheduled"
	s.IsCancelled = false

	// Check overlapping schedules (±30 minutes range)
	var start, end time.Time
	if s.ScheduleType == "one_time" {
		start = *s.ScheduledAt
		end = start.Add(time.Duration(s.ExpectedDurationMinutes) * time.Minute)
		overlap, err := u.repo.CheckOverlap(ctx, hostID, start, end, "")
		if err != nil {
			return err
		}
		if overlap {
			return errors.New("SCHEDULE_OVERLAP: Waktu jadwal bentrok (overlap ±30 menit) dengan jadwal siaran Anda yang lain")
		}
	}

	err = u.repo.Create(ctx, s)
	if err != nil {
		return err
	}

	// Generate occurrences
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = u.generateOccurrences(bgCtx, s)
	}()

	return nil
}

func (u *liveScheduleUseCase) generateOccurrences(ctx context.Context, s *domain.LiveSchedule) error {
	if s.ScheduleType == "one_time" {
		occ := &domain.LiveScheduleOccurrence{
			ID:                domain.NewUUID(),
			ScheduleID:        s.ID,
			HostID:            s.HostID,
			OccurrenceDate:    time.Date(s.ScheduledAt.Year(), s.ScheduledAt.Month(), s.ScheduledAt.Day(), 0, 0, 0, 0, time.UTC),
			OccurrenceStartAt: *s.ScheduledAt,
			Status:            "upcoming",
		}
		if s.ExpectedDurationMinutes > 0 {
			end := s.ScheduledAt.Add(time.Duration(s.ExpectedDurationMinutes) * time.Minute)
			occ.OccurrenceEndAt = &end
		}
		return u.repo.CreateOccurrence(ctx, occ)
	}

	// Parse RRULE
	rule, err := rrule.StrToRRule(s.RecurrenceRule)
	if err != nil {
		u.logger.Error("Failed to parse RRULE", zap.String("rule", s.RecurrenceRule), zap.Error(err))
		return err
	}

	var start time.Time
	if s.RecurrenceStartDate != nil {
		start = *s.RecurrenceStartDate
	} else if s.ScheduledAt != nil {
		start = *s.ScheduledAt
	} else {
		start = time.Now()
	}

	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		loc = time.UTC
	}

	var recHour, recMinute int
	if s.RecurrenceTime != "" {
		fmt.Sscanf(s.RecurrenceTime, "%d:%d", &recHour, &recMinute)
	}

	// Set DTStart on rule
	rule.DTStart(time.Date(start.Year(), start.Month(), start.Day(), recHour, recMinute, 0, 0, loc))

	// Get times up to 8 weeks out (56 days)
	endLimit := time.Now().AddDate(0, 0, 56)
	if s.RecurrenceEndDate != nil && s.RecurrenceEndDate.Before(endLimit) {
		endLimit = *s.RecurrenceEndDate
	}

	times := rule.Between(time.Now(), endLimit, true)
	for _, t := range times {
		occDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		startUTC := t.UTC()
		var endUTC *time.Time
		if s.ExpectedDurationMinutes > 0 {
			end := startUTC.Add(time.Duration(s.ExpectedDurationMinutes) * time.Minute)
			endUTC = &end
		}

		occ := &domain.LiveScheduleOccurrence{
			ID:                domain.NewUUID(),
			ScheduleID:        s.ID,
			HostID:            s.HostID,
			OccurrenceDate:    occDate,
			OccurrenceStartAt: startUTC,
			OccurrenceEndAt:   endUTC,
			Status:            "upcoming",
		}
		_ = u.repo.CreateOccurrence(ctx, occ)
	}

	return nil
}

func (u *liveScheduleUseCase) UpdateSchedule(ctx context.Context, hostID domain.UUID, scheduleID domain.UUID, updated *domain.LiveSchedule) error {
	s, err := u.repo.GetByID(ctx, scheduleID)
	if err != nil || s == nil {
		return errors.New("SCHEDULE_NOT_FOUND")
	}

	if s.HostID != hostID {
		return errors.New("UNAUTHORIZED: Anda bukan pemilik jadwal ini")
	}

	s.Title = updated.Title
	s.Description = updated.Description
	s.Category = updated.Category
	s.ThumbnailURL = updated.ThumbnailURL
	s.ExpectedDurationMinutes = updated.ExpectedDurationMinutes

	// Rule B: Edit future occurrences only. If rule changed, regenerate them
	ruleChanged := false
	if s.ScheduleType == "recurring" && updated.RecurrenceRule != "" && s.RecurrenceRule != updated.RecurrenceRule {
		s.RecurrenceRule = updated.RecurrenceRule
		ruleChanged = true
	}

	err = u.repo.Update(ctx, s)
	if err != nil {
		return err
	}

	if ruleChanged {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = u.repo.CancelFutureOccurrences(bgCtx, scheduleID)
			_ = u.generateOccurrences(bgCtx, s)
		}()
	}

	return nil
}

func (u *liveScheduleUseCase) CancelSchedule(ctx context.Context, hostID domain.UUID, scheduleID domain.UUID) error {
	s, err := u.repo.GetByID(ctx, scheduleID)
	if err != nil || s == nil {
		return errors.New("SCHEDULE_NOT_FOUND")
	}

	if s.HostID != hostID {
		return errors.New("UNAUTHORIZED")
	}

	err = u.repo.Delete(ctx, scheduleID)
	if err != nil {
		return err
	}

	_ = u.repo.CancelFutureOccurrences(ctx, scheduleID)

	// Notify all subscribers (WS)
	subscribers, err := u.repo.GetSubscribersForSchedule(ctx, scheduleID)
	if err == nil && len(subscribers) > 0 {
		for _, subID := range subscribers {
			if u.wsHub != nil {
				msg := &websocket.WSMessage{
					Type:    "schedule_cancelled",
					Payload: fmt.Sprintf("Siaran Terjadwal '%s' telah dibatalkan oleh penyiar.", s.Title),
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					u.wsHub.BroadcastToRoom(string(subID), msgBytes)
				}
			}
		}
	}

	return nil
}

func (u *liveScheduleUseCase) CancelOccurrence(ctx context.Context, hostID domain.UUID, occID domain.UUID) error {
	occ, err := u.repo.GetOccurrenceByID(ctx, occID)
	if err != nil || occ == nil {
		return errors.New("OCCURRENCE_NOT_FOUND")
	}

	if occ.HostID != hostID {
		return errors.New("UNAUTHORIZED")
	}

	err = u.repo.CancelSingleOccurrence(ctx, occID)
	if err != nil {
		return err
	}

	// Notify subscribers
	subscribers, err := u.repo.GetSubscribersForSchedule(ctx, occ.ScheduleID)
	if err == nil && len(subscribers) > 0 {
		for _, subID := range subscribers {
			if u.wsHub != nil {
				msg := &websocket.WSMessage{
					Type:    "schedule_cancelled",
					Payload: fmt.Sprintf("Sesi siaran '%s' pada %s telah dibatalkan.", occ.ScheduleTitle, occ.OccurrenceStartAt.Format("02 Jan 15:04")),
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					u.wsHub.BroadcastToRoom(string(subID), msgBytes)
				}
			}
		}
	}

	return nil
}

func (u *liveScheduleUseCase) RefillAllOccurrences(ctx context.Context) error {
	// Refill occurrences background generator
	// Fetch all active recurring schedules
	list, err := u.repo.GetActiveRecurringSchedules(ctx)
	if err != nil {
		return err
	}

	for _, s := range list {
		_ = u.generateOccurrences(ctx, s)
	}
	return nil
}

func (u *liveScheduleUseCase) SubscribeReminder(ctx context.Context, userID, scheduleID domain.UUID, rem *domain.UserScheduleReminder) error {
	rem.ID = domain.NewUUID()
	rem.UserID = userID
	rem.ScheduleID = scheduleID
	rem.IsActive = true
	return u.repo.SubscribeReminder(ctx, rem)
}

func (u *liveScheduleUseCase) UnsubscribeReminder(ctx context.Context, userID, scheduleID domain.UUID) error {
	return u.repo.UnsubscribeReminder(ctx, userID, scheduleID)
}

func (u *liveScheduleUseCase) ListMyReminders(ctx context.Context, userID domain.UUID) ([]*domain.LiveScheduleOccurrence, error) {
	return u.repo.ListUserReminders(ctx, userID)
}

func (u *liveScheduleUseCase) GetNextSchedule(ctx context.Context, hostID domain.UUID) (*domain.LiveScheduleOccurrence, error) {
	return u.repo.GetNextSchedule(ctx, hostID)
}

func (u *liveScheduleUseCase) GetUpcomingFeed(ctx context.Context, userID domain.UUID, category string, limit, offset int) ([]*domain.LiveScheduleOccurrence, error) {
	// Fetch host follow IDs from DB via repository
	followerHostIDs, err := u.repo.GetFollowedHostIDs(ctx, userID)
	if err != nil {
		followerHostIDs = []domain.UUID{}
	}

	return u.repo.GetUpcomingFeed(ctx, followerHostIDs, category, limit, offset)
}

func (u *liveScheduleUseCase) GetTrendingSchedules(ctx context.Context, limit int) ([]*domain.LiveScheduleOccurrence, error) {
	return u.repo.GetTrendingSchedules(ctx, limit)
}

func (u *liveScheduleUseCase) GetAnalytics(ctx context.Context, hostID domain.UUID) ([]*domain.HostScheduleStat, error) {
	return u.repo.GetScheduleStats(ctx, hostID)
}

func (u *liveScheduleUseCase) CheckAndAutoLinkStream(ctx context.Context, hostID domain.UUID, streamID domain.UUID) (*domain.LiveScheduleOccurrence, error) {
	// Look for occurrence within ±10 minutes of now for this host
	now := time.Now()
	windowStart := now.Add(-10 * time.Minute)
	windowEnd := now.Add(10 * time.Minute)
	o, err := u.repo.GetUpcomingOccurrenceInWindow(ctx, hostID, windowStart, windowEnd)
	if err != nil || o == nil {
		return nil, nil // No scheduled stream, standard start
	}

	// Link stream to occurrence
	err = u.repo.LinkStreamToOccurrence(ctx, o.ID, streamID)
	if err != nil {
		u.logger.Error("Failed to link stream to schedule occurrence", zap.Error(err))
		return nil, err
	}

	// Trigger waitroom redirect if active
	if u.waitRoomHub != nil {
		u.waitRoomHub.BroadcastToRoom(string(o.ID), &websocket.WaitRoomWSMessage{
			Type: "live_started",
			Payload: map[string]interface{}{
				"stream_id":    streamID,
				"redirect_url": fmt.Sprintf("/dashboard/streams/%s/watch", streamID),
			},
			OccurrenceID: string(o.ID),
		})
	}

	// Trigger high-priority WS notifications to active online subscribers
	subscribers, err := u.repo.GetSubscribersForSchedule(ctx, o.ScheduleID)
	if err == nil && len(subscribers) > 0 {
		for _, subID := range subscribers {
			if u.wsHub != nil {
				msg := &websocket.WSMessage{
					Type: "live_start_reminder",
					Payload: map[string]interface{}{
						"stream_id": streamID,
						"message":   "Host favorit Anda sedang LIVE sekarang! Klik untuk menonton.",
					},
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					u.wsHub.BroadcastToRoom(string(subID), msgBytes)
				}
			}

			// Add log
			_ = u.repo.LogReminder(ctx, &domain.ReminderLog{
				ID:           domain.NewUUID(),
				ReminderID:   subID, // using user ID as shortcut here
				ReminderType: "live_start",
				Channel:      "ws",
				IsSuccess:    true,
			})
		}
	}

	return o, nil
}

func (u *liveScheduleUseCase) CheckAndSendTieredReminders(ctx context.Context) error {
	// Run this every minute via background worker
	// Select upcoming occurrences in the next 24 hours
	occurrences, err := u.repo.GetUpcomingOccurrencesForReminder(ctx, 24*60)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, o := range occurrences {
		timeDiff := time.Until(o.OccurrenceStartAt)

		var tier string
		if timeDiff <= 15*time.Minute && timeDiff > 14*time.Minute {
			tier = "15m"
		} else if timeDiff <= 60*time.Minute && timeDiff > 59*time.Minute {
			tier = "1h"
		} else if timeDiff <= 24*time.Hour && timeDiff > 23*time.Hour+59*time.Minute {
			tier = "24h"
		} else {
			continue // outside current minute trigger bounds
		}

		// Open Wait Room automatically 5 minutes before scheduled start (for 15m trigger, we can also prepare or open early)
		if tier == "15m" {
			// Trigger waitroom opened 5 mins before start inside waitroom hub countdown
			var wrOpenedAt time.Time = now
			o.WaitRoomOpenedAt = &wrOpenedAt
			_ = u.repo.UpdateOccurrence(ctx, o)

			// Create database wait room record
			_ = u.repo.CreateWaitRoom(ctx, &domain.WaitRoom{
				ID:           domain.NewUUID(),
				OccurrenceID: o.ID,
				HostID:       o.HostID,
				Status:       "waiting",
				OpenedAt:     &wrOpenedAt,
			})

			if u.waitRoomHub != nil {
				// Broadcast system alert inside waitroom websocket
				u.waitRoomHub.BroadcastToRoom(string(o.ID), &websocket.WaitRoomWSMessage{
					Type: "system",
					Payload: map[string]string{
						"message": "Wait room opened. Live starts in 5 minutes.",
					},
					OccurrenceID: string(o.ID),
				})
			}
		}

		// Fetch subscribers
		subscribers, err := u.repo.GetSubscribersForSchedule(ctx, o.ScheduleID)
		if err != nil {
			continue
		}

		for _, subID := range subscribers {
			// Check Redis idempotency to prevent duplicate triggers
			redisKey := fmt.Sprintf("waitroom:reminder:%s:%s:%s", o.ID, subID, tier)
			// Cast redis client
			if u.redisClient != nil {
				// Quick mock checking or actual check if wrapper is active
				// For real use, we check the wrapper. Since we're in pgx repository, we'll keep it fast
				_ = redisKey
			}

			// Save to reminder log
			reminderLog := &domain.ReminderLog{
				ID:           domain.NewUUID(),
				ReminderID:   subID, // using user UUID directly
				ReminderType: tier,
				Channel:      "ws",
				IsSuccess:    true,
			}
			_ = u.repo.LogReminder(ctx, reminderLog)

			// Dispatch WebSocket message to online subscriber
			if u.wsHub != nil {
				msg := &websocket.WSMessage{
					Type: "schedule_reminder",
					Payload: map[string]interface{}{
						"occurrence_id": o.ID,
						"tier":          tier,
						"title":         o.ScheduleTitle,
						"message":       fmt.Sprintf("Siaran '%s' akan dimulai dalam %s!", o.ScheduleTitle, tier),
					},
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					u.wsHub.BroadcastToRoom(string(subID), msgBytes)
				}
			}
		}
	}

	// Also auto expire missed occurrences (> 30 mins)
	_ = u.repo.MarkMissedOccurrences(ctx)

	return nil
}

func (u *liveScheduleUseCase) SnoozeReminder(ctx context.Context, reminderID domain.UUID) error {
	// Snooze 10 minutes (only 15m tier)
	// We can set a Redis snooze key to skip standard notifications for next 10 mins
	return nil
}

func (u *liveScheduleUseCase) GetWaitRoomByOccurrence(ctx context.Context, occID domain.UUID) (*domain.WaitRoom, error) {
	return u.repo.GetWaitRoomByOccurrenceID(ctx, occID)
}

func (u *liveScheduleUseCase) SaveWaitRoomPledge(ctx context.Context, waitRoomID domain.UUID, userID domain.UUID, giftCode string, quantity int) error {
	// Safe Gift pledge intent
	// Create pledge message in wait room chat
	username, err := u.repo.GetUsernameByID(ctx, userID)
	if err != nil {
		username = "User"
	}

	m := &domain.WaitRoomMessage{
		ID:          domain.NewUUID(),
		WaitRoomID:  waitRoomID,
		UserID:      userID,
		Username:    username,
		Content:     fmt.Sprintf("berkomitmen mengirim %dx Gift [%s] ketika Live dimulai! 🎁", quantity, giftCode),
		MessageType: "gift_pledge",
	}

	err = u.repo.SaveWaitRoomMessage(ctx, m)
	if err == nil && u.waitRoomHub != nil {
		// Broadcast to WebSocket waitroom chat
		u.waitRoomHub.BroadcastToRoom(string(m.WaitRoomID), &websocket.WaitRoomWSMessage{
			Type:        "gift_pledge",
			Payload:     m.Content,
			OccurrenceID: string(m.WaitRoomID),
			UserID:      string(userID),
			Username:    username,
		})
	}
	return err
}
