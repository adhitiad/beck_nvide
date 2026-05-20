package delivery

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/middleware"
	"nvide-live/pkg/wallet"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter configures HTTP routes
func SetupRouter(
	handler *Handler,
	webrtcHandler *WebRTCHandler,
	vodHandler *VODHandler,
	monetizationHandler *MonetizationHandler,
	extraMonetizationHandler *ExtraMonetizationHandler,
	healthHandler *HealthHandler,
	cryptoHandler *CryptoHandler,
	pkHandler *PKBattleHandler,
	moderationHandler *ModerationHandler,
	creatorTokenHandler *CreatorTokenHandler,
	predictionHandler *PredictionHandler,
	drmHandler *DRMHandler,
	recommendationHandler *RecommendationHandler,
	clipHandler *ClipHandler,
	kycHandler *KYCHandler,
	clipSubHandler *ClipSubscriptionHandler,
	dashboardHandler *DashboardHandler,
	payoutHandler *PayoutHandler,
	pushHandler *PushHandler,
	authMiddleware *middleware.AuthMiddleware,
	rbacMiddleware *middleware.RBACMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	idempotencyMiddleware *wallet.IdempotencyManager,
	banChecker *middleware.BanChecker,
	clipQuota *middleware.ClipQuotaMiddleware,
	logger *zap.Logger,
) *mux.Router {
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.RequestID)                // Generate and inject UUID v7 X-Request-ID
	router.Use(middleware.LoggerMiddleware(logger)) // Log HTTP requests with correlation tracing
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.MetricsMiddleware)              // Apply Prometheus Metrics Middleware!
	router.Use(middleware.NewI18nMiddleware().Middleware) // Apply Multi-Language i18n Middleware!
	if idempotencyMiddleware != nil {
		router.Use(idempotencyMiddleware.Middleware())
	}

	// Apply rate limiting globally (optional)
	if rateLimitMiddleware != nil {
		router.Use(rateLimitMiddleware.Middleware)
	}

	// Main API subrouter
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Public routes
	apiV1.HandleFunc("/auth/register", handler.Register).Methods("POST")
	apiV1.HandleFunc("/auth/login", handler.Login).Methods("POST")
	apiV1.HandleFunc("/auth/refresh", handler.Refresh).Methods("POST")

	// Protected routes group
	protected := apiV1.PathPrefix("").Subrouter()
	protected.Use(authMiddleware.Middleware)
	if banChecker != nil {
		protected.Use(banChecker.Middleware)
	}
	if clipQuota != nil {
		protected.Use(clipQuota.Middleware)
	}

	// ===== KYC & ONBOARDING ROUTES =====
	if kycHandler != nil {
		// Use RegionValidator only on /kyc/submit
		kycRouter := protected.PathPrefix("/kyc").Subrouter()
		kycRouter.Use(middleware.RegionValidator)
		kycRouter.HandleFunc("/submit", kycHandler.SubmitKYC).Methods("POST")

		// Other KYC routes
		protected.HandleFunc("/kyc/status", kycHandler.GetStatus).Methods("GET")
		protected.HandleFunc("/kyc/agency/submit", kycHandler.SubmitAgency).Methods("POST")

		// Onboarding Checklist routes
		protected.HandleFunc("/onboarding/user", kycHandler.GetOnboardingChecklist).Methods("GET")
		protected.HandleFunc("/onboarding/host", kycHandler.GetOnboardingChecklist).Methods("GET")
		protected.HandleFunc("/onboarding/agency", kycHandler.GetOnboardingChecklist).Methods("GET")
		protected.HandleFunc("/onboarding/step/complete", kycHandler.CompleteOnboardingStep).Methods("POST")

		// Admin KYC & Ban
		protected.HandleFunc("/admin/kyc/{id}/verify", kycHandler.ApproveKYC).Methods("PUT")
		protected.HandleFunc("/admin/kyc/{id}/reject", kycHandler.RejectKYC).Methods("PUT")
		protected.HandleFunc("/admin/users/{id}/ban-permanent", kycHandler.BanUserPermanent).Methods("POST")
	}

	// ===== VIP CLIP SUBSCRIPTION ROUTES =====
	if clipSubHandler != nil {
		protected.HandleFunc("/clip-subscriptions/plans", clipSubHandler.ListPlans).Methods("GET")
		protected.HandleFunc("/clip-subscriptions/subscribe", clipSubHandler.Subscribe).Methods("POST")
		protected.HandleFunc("/clip-subscriptions/status", clipSubHandler.GetStatus).Methods("GET")
		protected.HandleFunc("/clip-subscriptions/history", clipSubHandler.GetHistory).Methods("GET")
	}

	// ===== DASHBOARD ROUTES =====
	if dashboardHandler != nil {
		// Admin Dashboard
		protected.HandleFunc("/admin/dashboard/stats", dashboardHandler.GetAdminStats).Methods("GET")
		protected.HandleFunc("/admin/dashboard/revenue", dashboardHandler.GetAdminRevenue).Methods("GET")
		protected.HandleFunc("/admin/dashboard/graph", dashboardHandler.GetAdminGraph).Methods("GET")
		protected.HandleFunc("/admin/users", dashboardHandler.ListUsers).Methods("GET")
		protected.HandleFunc("/admin/kyc/pending", dashboardHandler.ListPendingKYC).Methods("GET")
		protected.HandleFunc("/admin/reports", dashboardHandler.ListReports).Methods("GET")
		protected.HandleFunc("/admin/users/{id}/ban", dashboardHandler.BanUser).Methods("PUT")
		protected.HandleFunc("/admin/users/{id}/unban", dashboardHandler.UnbanUser).Methods("PUT")
		protected.HandleFunc("/admin/streams/{id}/terminate", dashboardHandler.TerminateStream).Methods("PUT")
		protected.HandleFunc("/admin/comments/{id}", dashboardHandler.DeleteComment).Methods("DELETE")
		protected.HandleFunc("/reports", dashboardHandler.SubmitReport).Methods("POST")

		// Host Dashboard
		protected.HandleFunc("/host/dashboard/stats", dashboardHandler.GetHostStats).Methods("GET")
		protected.HandleFunc("/host/dashboard/revenue", dashboardHandler.GetHostRevenue).Methods("GET")
		protected.HandleFunc("/host/dashboard/clips", dashboardHandler.GetHostClips).Methods("GET")
		protected.HandleFunc("/host/dashboard/streams", dashboardHandler.GetHostStreams).Methods("GET")
		protected.HandleFunc("/host/dashboard/requests", dashboardHandler.GetHostRequests).Methods("GET")
		protected.HandleFunc("/host/dashboard/settings", dashboardHandler.UpdateHostSettings).Methods("PUT")

		// Agency Dashboard
		protected.HandleFunc("/agency/dashboard/stats", dashboardHandler.GetAgencyStats).Methods("GET")
		protected.HandleFunc("/agency/dashboard/hosts", dashboardHandler.GetAgencyHosts).Methods("GET")
		protected.HandleFunc("/agency/dashboard/hosts/invite", dashboardHandler.InviteHost).Methods("POST")
		protected.HandleFunc("/agency/dashboard/hosts/{id}", dashboardHandler.RemoveHost).Methods("DELETE")
		protected.HandleFunc("/agency/dashboard/revenue", dashboardHandler.GetAgencyRevenue).Methods("GET")
		protected.HandleFunc("/agency/dashboard/settings", dashboardHandler.UpdateAgencySettings).Methods("PUT")
	}

	// ===== PAYOUT METHODS ROUTES =====
	if payoutHandler != nil {
		protected.HandleFunc("/payout-methods", payoutHandler.ListPayoutMethods).Methods("GET")
		protected.HandleFunc("/payout-methods", payoutHandler.CreatePayoutMethod).Methods("POST")
		protected.HandleFunc("/payout-methods/{id}", payoutHandler.UpdatePayoutMethod).Methods("PUT")
		protected.HandleFunc("/payout-methods/{id}", payoutHandler.DeletePayoutMethod).Methods("DELETE")
		protected.HandleFunc("/payout-methods/{id}/primary", payoutHandler.SetPrimaryPayoutMethod).Methods("PUT")

		protected.HandleFunc("/crypto-payout-addresses", payoutHandler.ListCryptoPayoutAddresses).Methods("GET")
		protected.HandleFunc("/crypto-payout-addresses", payoutHandler.CreateCryptoPayoutAddress).Methods("POST")
		protected.HandleFunc("/crypto-payout-addresses/{id}", payoutHandler.DeleteCryptoPayoutAddress).Methods("DELETE")
	}

	// ===== PUSH NOTIFICATION ROUTES =====
	if pushHandler != nil {
		protected.HandleFunc("/push/subscribe", pushHandler.Subscribe).Methods("POST")
		protected.HandleFunc("/push/unsubscribe", pushHandler.Unsubscribe).Methods("POST")
		protected.HandleFunc("/push/test", pushHandler.SendTest).Methods("POST")
	}

	// Auth routes
	protected.HandleFunc("/auth/logout", handler.Logout).Methods("POST")
	protected.HandleFunc("/auth/me", handler.Me).Methods("GET")

	// Story routes
	protected.HandleFunc("/stories", handler.CreateStory).Methods("POST")
	protected.HandleFunc("/stories/{id}", handler.GetStory).Methods("GET")
	protected.HandleFunc("/stories/user/{user_id}", handler.GetUserStories).Methods("GET")
	protected.HandleFunc("/stories/feed", handler.GetFeedStories).Methods("GET")
	protected.HandleFunc("/stories/{id}/view", handler.ViewStory).Methods("POST")
	protected.HandleFunc("/stories/{id}", handler.DeleteStory).Methods("DELETE")

	// Comment routes
	protected.HandleFunc("/comments", handler.CreateComment).Methods("POST")
	protected.HandleFunc("/comments/{id}", handler.GetComment).Methods("GET")
	protected.HandleFunc("/comments/{id}/replies", handler.GetReplies).Methods("GET")
	protected.HandleFunc("/content/{content_id}/{content_type}/comments", handler.GetComments).Methods("GET")
	protected.HandleFunc("/comments/{id}", handler.UpdateComment).Methods("PUT")
	protected.HandleFunc("/comments/{id}", handler.DeleteComment).Methods("DELETE")
	protected.HandleFunc("/comments/{id}/like", handler.LikeComment).Methods("POST")
	protected.HandleFunc("/comments/{id}/unlike", handler.UnlikeComment).Methods("POST")

	// Like routes
	protected.HandleFunc("/likes", handler.LikeContent).Methods("POST")
	protected.HandleFunc("/likes/unlike", handler.UnlikeContent).Methods("POST")

	// Message routes
	protected.HandleFunc("/messages", handler.SendMessage).Methods("POST")
	protected.HandleFunc("/rooms/{room_id}/messages", handler.GetMessages).Methods("GET")
	protected.HandleFunc("/rooms/{room_id}/messages/recent", handler.GetRecentMessages).Methods("GET")
	protected.HandleFunc("/messages/{id}", handler.GetMessage).Methods("GET")
	protected.HandleFunc("/messages/{id}", handler.DeleteMessage).Methods("DELETE")
	protected.HandleFunc("/streams/{stream_id}/room", handler.GetOrCreateRoom).Methods("GET")
	protected.HandleFunc("/rooms/{room_id}/join", handler.JoinRoom).Methods("POST")
	protected.HandleFunc("/rooms/{room_id}/leave", handler.LeaveRoom).Methods("POST")

	// Private Chat (DM) routes
	protected.HandleFunc("/conversations", handler.StartConversation).Methods("POST")
	protected.HandleFunc("/conversations", handler.GetConversations).Methods("GET")
	protected.HandleFunc("/conversations/{id}/messages", handler.SendPrivateMessage).Methods("POST")
	protected.HandleFunc("/conversations/{id}/messages", handler.GetPrivateMessages).Methods("GET")
	protected.HandleFunc("/conversations/{id}/read", handler.MarkAsRead).Methods("POST")
	protected.HandleFunc("/conversations/{id}/settings", handler.UpdateConversationSettings).Methods("PUT")
	protected.HandleFunc("/messages/{id}/reactions", handler.ToggleReaction).Methods("POST")
	protected.HandleFunc("/messages/{id}/view", handler.MarkMessageViewed).Methods("POST")
	protected.HandleFunc("/users/{id}/block", handler.BlockUser).Methods("POST")
	protected.HandleFunc("/users/me/e2ee-key", handler.RegisterE2EEKey).Methods("POST")
	protected.HandleFunc("/users/{id}/e2ee-key", handler.GetE2EEKey).Methods("GET")
	protected.HandleFunc("/users/{id}/mute", handler.MuteUser).Methods("POST")
	protected.HandleFunc("/users/{id}/unmute", handler.UnmuteUser).Methods("POST")
	protected.HandleFunc("/users/me/mutes", handler.GetMutedUsers).Methods("GET")
	protected.HandleFunc("/users/me/privacy", handler.UpdatePrivacySettings).Methods("PUT")
	protected.HandleFunc("/profile/privacy", handler.UpdatePrivacySettings).Methods("PUT")
	protected.HandleFunc("/conversations/{id}/screenshot", handler.NotifyScreenshot).Methods("POST")

	// Paid Interaction routes
	protected.HandleFunc("/conversations/{id}/unlock", handler.UnlockChat).Methods("POST")
	protected.HandleFunc("/hosts/me/call-rates", handler.SetCallRates).Methods("POST")
	router.HandleFunc("/api/v1/hosts/{id}/call-rates", handler.GetHostCallRates).Methods("GET")
	protected.HandleFunc("/calls/request", handler.RequestCall).Methods("POST")
	protected.HandleFunc("/calls/{id}/accept", handler.AcceptCall).Methods("POST")
	protected.HandleFunc("/calls/{id}/end", handler.EndCall).Methods("POST")
	protected.HandleFunc("/calls/{id}", handler.GetCallSession).Methods("GET")

	// Booking routes
	protected.HandleFunc("/hosts/me/schedules", handler.SetHostSchedule).Methods("POST")
	router.HandleFunc("/api/v1/hosts/{id}/available-slots", handler.GetAvailableSlots).Methods("GET")
	protected.HandleFunc("/bookings", handler.RequestBooking).Methods("POST")
	protected.HandleFunc("/bookings", handler.ListMyBookings).Methods("GET")
	hostBookings := protected.PathPrefix("/host/bookings").Subrouter()
	hostBookings.HandleFunc("", handler.ListHostBookings).Methods("GET")
	hostBookings.HandleFunc("/{id}/accept", handler.AcceptBooking).Methods("POST")
	hostBookings.HandleFunc("/{id}/reject", handler.RejectBooking).Methods("POST")

	// Live Streaming Schedules (Fitur 7)
	protected.HandleFunc("/streams/schedules", handler.CreateSchedule).Methods("POST")
	protected.HandleFunc("/streams/schedules/{id}", handler.UpdateSchedule).Methods("PUT")
	protected.HandleFunc("/streams/schedules/{id}", handler.CancelSchedule).Methods("DELETE")
	protected.HandleFunc("/streams/schedules/{id}/occurrences/{occ_id}/cancel", handler.CancelOccurrence).Methods("POST")
	protected.HandleFunc("/streams/schedules/{id}/reminders", handler.SubscribeReminder).Methods("POST")
	protected.HandleFunc("/streams/schedules/{id}/reminders", handler.UnsubscribeReminder).Methods("DELETE")
	protected.HandleFunc("/users/me/reminders", handler.ListMyReminders).Methods("GET")
	router.HandleFunc("/api/v1/hosts/{id}/schedules/next", handler.GetNextSchedule).Methods("GET")
	router.HandleFunc("/api/v1/discover/upcoming", handler.GetUpcomingFeed).Methods("GET")
	router.HandleFunc("/api/v1/discover/trending-schedules", handler.GetTrendingSchedules).Methods("GET")
	protected.HandleFunc("/hosts/me/schedules/analytics", handler.GetAnalytics).Methods("GET")
	protected.HandleFunc("/wait-rooms/{id}/pledge", handler.PledgeGift).Methods("POST")

	// Wait Room WebSocket (WebSocket)
	router.HandleFunc("/ws/wait-room/{occurrence_id}", handler.ServeWaitRoomWS).Methods("GET")

	// OB/BO routes
	protected.HandleFunc("/hosts/me/offers", handler.CreateHostOffer).Methods("POST")
	router.HandleFunc("/api/v1/hosts/{id}/offers", handler.GetOfferFeed).Methods("GET")
	protected.HandleFunc("/offers/{id}/book", handler.BookHostOffer).Methods("POST")
	protected.HandleFunc("/users/me/offers", handler.CreateUserOffer).Methods("POST")
	protected.HandleFunc("/host/offers/{id}/respond", handler.RespondToUserOffer).Methods("POST")
	router.HandleFunc("/api/v1/discover/offers", handler.GetOfferFeed).Methods("GET")

	// Location routes
	protected.HandleFunc("/users/me/location", handler.UpdateMyLocation).Methods("POST")
	protected.HandleFunc("/hosts/{host_id}/live-location", handler.GetHostLiveLocation).Methods("GET")
	protected.HandleFunc("/bookings/{id}/meeting-point", handler.GetBookingMeetingPoint).Methods("GET")

	// WebSocket routes (Chat & Call)
	router.HandleFunc("/ws/rooms/{room_id}", handler.ServeWS).Methods("GET")
	router.HandleFunc("/ws/chat/{stream_id}", handler.ServeStreamChatWS).Methods("GET")
	router.HandleFunc("/ws/private-chat/{conversation_id}", handler.ServeChatWS).Methods("GET")
	router.HandleFunc("/ws/call/{session_id}", handler.ServeCallWS).Methods("GET")

	// WebRTC Signaling (WebSocket)
	router.HandleFunc("/api/v1/streams/{room_id}/signal", webrtcHandler.SignalWS).Methods("GET")
	router.HandleFunc("/ws/stream/{room_id}/host", webrtcHandler.ServeHostWS).Methods("GET")
	router.HandleFunc("/ws/stream/{room_id}/viewer", webrtcHandler.ServeViewerWS).Methods("GET")
	router.HandleFunc("/api/v1/streams/{stream_id}", webrtcHandler.GetStream).Methods("GET")

	// Stream Management (Protected)
	protected.HandleFunc("/streams", webrtcHandler.CreateStream).Methods("POST")
	protected.HandleFunc("/streams/{stream_id}/start", webrtcHandler.StartStream).Methods("POST")
	protected.HandleFunc("/streams/{stream_id}/end", webrtcHandler.EndStream).Methods("POST", "PUT")
	protected.HandleFunc("/streams/{stream_id}/mode", webrtcHandler.SwitchRoomMode).Methods("PATCH")

	// Public Stream Routes
	apiV1.HandleFunc("/streams", webrtcHandler.GetLiveStreams).Methods("GET")
	apiV1.HandleFunc("/streams/live", webrtcHandler.GetLiveStreams).Methods("GET")
	apiV1.HandleFunc("/streams/trending", webrtcHandler.GetTrendingStreams).Methods("GET")
	apiV1.HandleFunc("/vods", vodHandler.GetVODList).Methods("GET")

	protected.HandleFunc("/vods", vodHandler.UploadVOD).Methods("POST")
	apiV1.HandleFunc("/vods/{vod_id}", vodHandler.GetVODDetail).Methods("GET")
	protected.HandleFunc("/vods/{vod_id}/visibility", vodHandler.UpdateVisibility).Methods("PUT")
	protected.HandleFunc("/vods/{vod_id}", vodHandler.DeleteVOD).Methods("DELETE")

	// DRM & Video Playback Terenkripsi (Fitur 3)
	protected.HandleFunc("/vods/{vod_id}/token", drmHandler.GenerateToken).Methods("POST")
	router.HandleFunc("/api/v1/vods/{vod_id}/playlist.m3u8", drmHandler.ServePlaylist).Methods("GET")
	router.HandleFunc("/api/v1/vods/{vod_id}/key", drmHandler.ServeKey).Methods("GET")
	router.HandleFunc("/api/v1/vods/{vod_id}/segments/{segment}", drmHandler.ServeSegment).Methods("GET")

	// AI Recommendation System (Fitur 4)
	protected.HandleFunc("/recommendations/interactions", recommendationHandler.TrackInteraction).Methods("POST")
	protected.HandleFunc("/recommendations/streams", recommendationHandler.GetRecommendedStreams).Methods("GET")
	protected.HandleFunc("/recommendations/vods", recommendationHandler.GetRecommendedVODs).Methods("GET")

	// AI Clip Highlights System (Fitur 5)
	protected.HandleFunc("/streams/{stream_id}/clips/trigger", clipHandler.TriggerClip).Methods("POST")
	apiV1.HandleFunc("/streams/{stream_id}/clips", clipHandler.GetStreamClips).Methods("GET")
	apiV1.HandleFunc("/clips/trending", clipHandler.GetTrendingClips).Methods("GET")

	// Static files (local storage)
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// ===== MONETIZATION ROUTES =====

	// Wallet
	protected.HandleFunc("/wallet/balance", monetizationHandler.GetBalance).Methods("GET")
	protected.HandleFunc("/wallet/transactions", monetizationHandler.GetTransactionHistory).Methods("GET")
	protected.HandleFunc("/withdrawals/preview", monetizationHandler.GetWithdrawalPreview).Methods("GET")
	protected.HandleFunc("/withdrawals", monetizationHandler.SubmitWithdrawal).Methods("POST")

	// Gifts
	router.HandleFunc("/api/v1/gifts", monetizationHandler.GetGiftCatalog).Methods("GET")
	protected.HandleFunc("/gifts/send", monetizationHandler.SendGift).Methods("POST")
	router.HandleFunc("/api/v1/streams/{stream_id}/leaderboard", monetizationHandler.GetGiftLeaderboard).Methods("GET")

	// Host & Agency
	protected.HandleFunc("/host/apply", monetizationHandler.ApplyHost).Methods("POST")
	protected.HandleFunc("/agencies", monetizationHandler.CreateAgency).Methods("POST")
	protected.HandleFunc("/agencies/me", monetizationHandler.GetMyAgency).Methods("GET")
	protected.HandleFunc("/agencies/{agency_id}/invite", monetizationHandler.InviteHost).Methods("POST")
	protected.HandleFunc("/agencies/{agency_id}/hosts", monetizationHandler.ListAgencyHosts).Methods("GET")
	protected.HandleFunc("/agencies/{agency_id}/accept", monetizationHandler.AcceptInvitation).Methods("POST")
	protected.HandleFunc("/host/agency", monetizationHandler.GetHostRelation).Methods("GET")

	// Payment
	protected.HandleFunc("/payment/deposit", monetizationHandler.RequestDeposit).Methods("POST")
	protected.HandleFunc("/payment/withdraw", monetizationHandler.RequestWithdrawal).Methods("POST")
	router.HandleFunc("/api/v1/payment/duitku/callback", monetizationHandler.DuitkuCallback).Methods("POST")

	// Admin endpoints
	protected.HandleFunc("/admin/host-applications", monetizationHandler.ListHostApplications).Methods("GET")
	protected.HandleFunc("/admin/host-applications/{application_id}/approve", monetizationHandler.ApproveHostApplication).Methods("POST")
	protected.HandleFunc("/admin/host-applications/{application_id}/reject", monetizationHandler.RejectHostApplication).Methods("POST")
	protected.HandleFunc("/admin/withdrawals/{tx_id}/approve", monetizationHandler.ApproveWithdrawal).Methods("POST")
	protected.HandleFunc("/admin/withdrawals/{tx_id}/reject", monetizationHandler.RejectWithdrawal).Methods("POST")

	// ===== CREATOR TOKEN ROUTES =====
	if creatorTokenHandler != nil {
		protected.HandleFunc("/creators/{host_id}/tokens", creatorTokenHandler.IssueToken).Methods("POST")
		router.HandleFunc("/api/v1/creators/{host_id}/tokens", creatorTokenHandler.GetTokenInfo).Methods("GET")
		protected.HandleFunc("/tokens/buy", creatorTokenHandler.BuyToken).Methods("POST")
		protected.HandleFunc("/users/{id}/tokens", creatorTokenHandler.GetUserBalances).Methods("GET")
	}

	// ===== PREDICTION MARKET ROUTES =====
	if predictionHandler != nil {
		protected.HandleFunc("/streams/{id}/predictions", predictionHandler.CreatePrediction).Methods("POST")
		router.HandleFunc("/api/v1/streams/{id}/predictions", predictionHandler.GetActivePredictions).Methods("GET")
		protected.HandleFunc("/predictions/{id}/bet", predictionHandler.PlaceBet).Methods("POST")
		protected.HandleFunc("/predictions/{id}/resolve", predictionHandler.ResolvePrediction).Methods("PUT")
	}

	// ===== CRYPTO ROUTES =====
	protected.HandleFunc("/crypto/deposit-address", cryptoHandler.GetDepositAddress).Methods("GET")
	protected.HandleFunc("/crypto/withdrawal", cryptoHandler.RequestWithdrawal).Methods("POST")
	router.HandleFunc("/api/v1/crypto/rates", cryptoHandler.GetExchangeRates).Methods("GET")
	router.HandleFunc("/api/v1/crypto/webhooks/{provider}", cryptoHandler.CryptoWebhook).Methods("POST")

	// ===== PK BATTLE ROUTES =====
	protected.HandleFunc("/pk/battle/invite", pkHandler.InvitePK).Methods("POST")
	protected.HandleFunc("/pk/battle/{pk_id}/accept", pkHandler.AcceptPK).Methods("POST")
	protected.HandleFunc("/pk/battle/{pk_id}/reject", pkHandler.RejectPK).Methods("POST")
	router.HandleFunc("/api/v1/pk/battle/{pk_id}/status", pkHandler.GetStatus).Methods("GET")

	// ===== AUTO-MODERATION & SAFETY ENGINE ROUTES =====
	if moderationHandler != nil {
		RegisterModerationRoutes(protected, moderationHandler)
	}

	// ===== EXTRA MONETIZATION ROUTES =====
	if extraMonetizationHandler != nil {
		// Paid Rooms
		protected.HandleFunc("/rooms", extraMonetizationHandler.CreatePaidRoom).Methods("POST")
		protected.HandleFunc("/rooms/{id}/join", extraMonetizationHandler.JoinPaidRoom).Methods("POST")

		// Interactive Toys (Lovense)
		protected.HandleFunc("/hosts/me/devices", extraMonetizationHandler.RegisterHostDevice).Methods("POST")
		protected.HandleFunc("/hosts/me/devices", extraMonetizationHandler.GetHostDevices).Methods("GET")
		protected.HandleFunc("/streams/{id}/toys/control", extraMonetizationHandler.ControlToys).Methods("POST")

		// Show Requests
		protected.HandleFunc("/streams/{id}/requests", extraMonetizationHandler.SubmitShowRequest).Methods("POST")
		protected.HandleFunc("/requests/{id}/accept", extraMonetizationHandler.AcceptShowRequest).Methods("PUT")
		protected.HandleFunc("/requests/{id}/reject", extraMonetizationHandler.RejectShowRequest).Methods("PUT")

		// AI Companion
		protected.HandleFunc("/ai/chat", extraMonetizationHandler.SendAIChatMessage).Methods("POST")
		protected.HandleFunc("/ai/chat/history", extraMonetizationHandler.GetAIChatHistory).Methods("GET")
	}

	// Health check — must be registered before rate limiter to stay unauthenticated
	router.HandleFunc("/health", healthHandler.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	// Metrics — served locally only; requires basic-auth in production
	go serveMetrics(logger)

	return router
}

// serveMetrics starts a localhost-only metrics server with optional basic-auth in production.
func serveMetrics(logger *zap.Logger) {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	addr := "127.0.0.1:9090"
	logger.Info("Metrics server listening on " + addr)
	if err := http.ListenAndServe(addr, metricsMux); err != nil {
		logger.Error("Metrics server failed", zap.Error(err))
	}
}

// SetupRouter configures HTTP routes
