package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/SherClockHolmes/webpush-go"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type pushNotificationUsecase struct {
	repo         domain.PushSubscriptionRepository
	vapidPublic  string
	vapidPrivate string
	vapidSubject string
	logger       *zap.Logger
}

func NewPushNotificationUsecase(
	repo domain.PushSubscriptionRepository,
	vapidPublic string,
	vapidPrivate string,
	vapidSubject string,
	logger *zap.Logger,
) domain.PushNotificationUsecase {
	return &pushNotificationUsecase{
		repo:         repo,
		vapidPublic:  strings.TrimSpace(vapidPublic),
		vapidPrivate: strings.TrimSpace(vapidPrivate),
		vapidSubject: strings.TrimSpace(vapidSubject),
		logger:       logger,
	}
}

func (u *pushNotificationUsecase) Subscribe(ctx context.Context, userID domain.UUID, endpoint, p256dh, auth string, topics []string) error {
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(p256dh) == "" || strings.TrimSpace(auth) == "" {
		return errors.New("endpoint and keys are required")
	}
	item := &domain.PushSubscription{
		ID:        domain.NewUUIDv7(),
		UserID:    userID,
		Endpoint:  strings.TrimSpace(endpoint),
		P256DHKey: strings.TrimSpace(p256dh),
		AuthKey:   strings.TrimSpace(auth),
		Topics:    topics,
	}
	return u.repo.Upsert(ctx, item)
}

func (u *pushNotificationUsecase) Unsubscribe(ctx context.Context, userID domain.UUID, endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("endpoint is required")
	}
	return u.repo.DeleteByEndpointAndUserID(ctx, userID, endpoint)
}

func (u *pushNotificationUsecase) SendTest(ctx context.Context, userID domain.UUID, title, body string) (int, error) {
	if u.vapidPublic == "" || u.vapidPrivate == "" || u.vapidSubject == "" {
		return 0, errors.New("vapid keys are not configured")
	}
	if strings.TrimSpace(title) == "" {
		title = "NVide Live"
	}
	if strings.TrimSpace(body) == "" {
		body = "Push notification test berhasil."
	}

	subs, err := u.repo.ListByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"title": title,
		"body":  body,
		"data": map[string]string{
			"url": "/dashboard/notifications",
		},
	})

	success := 0
	for _, s := range subs {
		sub := &webpush.Subscription{
			Endpoint: s.Endpoint,
			Keys: webpush.Keys{
				P256dh: s.P256DHKey,
				Auth:   s.AuthKey,
			},
		}
		resp, sendErr := webpush.SendNotification(payload, sub, &webpush.Options{
			Subscriber:      u.vapidSubject,
			VAPIDPublicKey:  u.vapidPublic,
			VAPIDPrivateKey: u.vapidPrivate,
			TTL:             60,
		})
		if sendErr != nil {
			u.logger.Warn("Failed to send push notification", zap.Error(sendErr), zap.String("endpoint", s.Endpoint))
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success++
		}
	}
	return success, nil
}
