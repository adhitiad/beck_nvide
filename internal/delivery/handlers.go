package delivery

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
	"nvide-live/internal/websocket"
)

// Handler wraps HTTP handlers with dependencies
type Handler struct {
	authUseCase    *usecase.AuthUseCase
	userUseCase    *usecase.UserUseCase
	storyUseCase   *usecase.StoryUseCase
	commentUseCase *usecase.CommentUseCase
	likeUseCase    *usecase.LikeUseCase
	messageUseCase *usecase.MessageUseCase
	privateChatUseCase domain.PrivateChatUsecase
	paidInteractionUseCase domain.PaidInteractionUsecase
	bookingUseCase domain.BookingUsecase
	offerUseCase   domain.OfferUsecase
	locationUseCase domain.LocationUsecase
	liveScheduleUseCase domain.LiveScheduleUseCase
	waitRoomHub    *websocket.WaitRoomHub
	wsHub          *websocket.Hub
	logger         *zap.Logger
}

// NewHandler creates new HTTP handlers
func NewHandler(
	authUseCase *usecase.AuthUseCase,
	userUseCase *usecase.UserUseCase,
	storyUseCase *usecase.StoryUseCase,
	commentUseCase *usecase.CommentUseCase,
	likeUseCase *usecase.LikeUseCase,
	messageUseCase *usecase.MessageUseCase,
	privateChatUseCase domain.PrivateChatUsecase,
	paidInteractionUseCase domain.PaidInteractionUsecase,
	bookingUseCase domain.BookingUsecase,
	offerUseCase domain.OfferUsecase,
	locationUseCase domain.LocationUsecase,
	liveScheduleUseCase domain.LiveScheduleUseCase,
	waitRoomHub *websocket.WaitRoomHub,
	wsHub *websocket.Hub,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		authUseCase:   authUseCase,
		userUseCase:   userUseCase,
		storyUseCase:  storyUseCase,
		commentUseCase: commentUseCase,
		likeUseCase:   likeUseCase,
		messageUseCase: messageUseCase,
		privateChatUseCase: privateChatUseCase,
		paidInteractionUseCase: paidInteractionUseCase,
		bookingUseCase:         bookingUseCase,
		offerUseCase:           offerUseCase,
		locationUseCase:        locationUseCase,
		liveScheduleUseCase:    liveScheduleUseCase,
		waitRoomHub:            waitRoomHub,
		wsHub:                  wsHub,
		logger:                 logger,
	}
}

// writeJSON writes a standardized JSON envelope response (delegates to shared middleware helper)
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	middleware.WriteJSON(w, status, data)
}

// writeError writes a standardised JSON error response (delegates to shared middleware helper)
func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	middleware.WriteJSONError(w, status, code, message)
}

// writeErrorLocalized writes a localized error response using the shared envelope
func (h *Handler) writeErrorLocalized(ctx context.Context, w http.ResponseWriter, status int, code, translationKey string, defaultMessage string) {
	msg := middleware.T(ctx, translationKey)
	if msg == translationKey {
		msg = defaultMessage
	}
	middleware.WriteJSONError(w, status, code, msg)
}

// handleError handles domain errors and converts to HTTP response
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	h.handleErrorCtx(context.Background(), w, err)
}

// handleErrorCtx handles domain errors and converts to localized HTTP response
func (h *Handler) handleErrorCtx(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case err == domain.ErrNotFound:
		h.writeErrorLocalized(ctx, w, http.StatusNotFound, domain.ErrCodeNotFound, "not_found", "Resource not found")
	case err == domain.ErrConflict:
		h.writeErrorLocalized(ctx, w, http.StatusConflict, domain.ErrCodeConflict, "conflict", err.Error())
	case err == domain.ErrUnauthorized:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeUnauthorized, "unauthorized", err.Error())
	case err == domain.ErrForbidden:
		h.writeErrorLocalized(ctx, w, http.StatusForbidden, domain.ErrCodeForbidden, "forbidden", err.Error())
	case err == domain.ErrInvalidToken:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeInvalidToken, "invalid_token", err.Error())
	case err == domain.ErrExpiredToken:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeExpiredToken, "expired_token", err.Error())
	case err == domain.ErrTokenRevoked:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeTokenRevoked, "token_revoked", err.Error())
	case err == domain.ErrRateLimitExceeded:
		h.writeErrorLocalized(ctx, w, http.StatusTooManyRequests, domain.ErrCodeRateLimit, "rate_limit", err.Error())
	case err == domain.ErrInvalidCredentials:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeInvalidCreds, "invalid_credentials", err.Error())
	case err != nil:
		if _, ok := err.(domain.ValidationError); ok {
			h.writeErrorLocalized(ctx, w, http.StatusBadRequest, domain.ErrCodeValidation, "validation_error", err.Error())
		} else {
			h.logger.Error("Unhandled error", zap.Error(err))
			h.writeErrorLocalized(ctx, w, http.StatusInternalServerError, domain.ErrCodeInternal, "internal_error", "Internal server error")
		}
	}
}