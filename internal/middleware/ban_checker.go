package middleware

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type BanChecker struct {
	bannedRepo domain.BannedUserRepository
	logger     *zap.Logger
}

func NewBanChecker(bannedRepo domain.BannedUserRepository, logger *zap.Logger) *BanChecker {
	return &BanChecker{
		bannedRepo: bannedRepo,
		logger:     logger,
	}
}

func (m *BanChecker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Check IP address
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		// Clean port from RemoteAddr
		if strings.Contains(ip, ":") {
			parts := strings.Split(ip, ":")
			ip = parts[0]
		}

		if ip != "" {
			isBanned, banInfo, err := m.bannedRepo.IsIPBanned(ctx, ip)
			if err == nil && isBanned {
				m.logger.Warn("Blocked request from banned IP address", zap.String("ip", ip), zap.String("reason", banInfo.Reason))
				writeJSONError(w, http.StatusForbidden, "BANNED", "Your access has been permanently blocked due to a policy violation.")
				return
			}
		}

		// 2. Check Device Fingerprint header
		fingerprint := r.Header.Get("X-Device-Fingerprint")
		if fingerprint != "" {
			isBanned, banInfo, err := m.bannedRepo.IsDeviceBanned(ctx, fingerprint)
			if err == nil && isBanned {
				m.logger.Warn("Blocked request from banned device fingerprint", zap.String("fingerprint", fingerprint), zap.String("reason", banInfo.Reason))
				writeJSONError(w, http.StatusForbidden, "BANNED", "Your device has been permanently blocked due to a policy violation.")
				return
			}
		}

		// 3. Check authenticated User ID if present
		userID, ok := GetUserIDFromContext(ctx)
		if ok {
			isBanned, banInfo, err := m.bannedRepo.IsBanned(ctx, userID)
			if err == nil && isBanned {
				m.logger.Warn("Blocked request from banned user ID", zap.String("user_id", userID.String()), zap.String("reason", banInfo.Reason))
				writeJSONError(w, http.StatusForbidden, "BANNED", "Your account has been permanently blocked due to a policy violation.")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
