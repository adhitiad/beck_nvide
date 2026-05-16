package delivery

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/middleware"
)

// SetupRouter configures HTTP routes
func SetupRouter(
	handler *Handler,
	webrtcHandler *WebRTCHandler,
	vodHandler *VODHandler,
	monetizationHandler *MonetizationHandler,
	healthHandler *HealthHandler,
	cryptoHandler *CryptoHandler,
	authMiddleware *middleware.AuthMiddleware,
	rbacMiddleware *middleware.RBACMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	logger *zap.Logger,
) *mux.Router {
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))

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

	// Paid Interaction routes
	protected.HandleFunc("/conversations/{id}/unlock", handler.UnlockChat).Methods("POST")
	protected.HandleFunc("/hosts/me/call-rates", handler.SetCallRates).Methods("POST")
	router.HandleFunc("/api/v1/hosts/{id}/call-rates", handler.GetHostCallRates).Methods("GET")
	protected.HandleFunc("/calls/request", handler.RequestCall).Methods("POST")
	protected.HandleFunc("/calls/{id}/accept", handler.AcceptCall).Methods("POST")
	protected.HandleFunc("/calls/{id}/end", handler.EndCall).Methods("POST")

	// Booking routes
	protected.HandleFunc("/hosts/me/schedules", handler.SetHostSchedule).Methods("POST")
	router.HandleFunc("/api/v1/hosts/{id}/available-slots", handler.GetAvailableSlots).Methods("GET")
	protected.HandleFunc("/bookings", handler.RequestBooking).Methods("POST")
	protected.HandleFunc("/host/bookings/{id}/accept", handler.AcceptBooking).Methods("POST")
	protected.HandleFunc("/host/bookings/{id}/reject", handler.RejectBooking).Methods("POST")

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
	router.HandleFunc("/ws/chat/{conversation_id}", handler.ServeChatWS).Methods("GET")
	router.HandleFunc("/ws/call/{session_id}", handler.ServeCallWS).Methods("GET")

	// WebRTC Signaling (WebSocket)
	router.HandleFunc("/api/v1/ws/stream/{room_id}/signal", webrtcHandler.SignalWS).Methods("GET")
	
	// Stream Management (Protected)
	protected.HandleFunc("/streams", webrtcHandler.CreateStream).Methods("POST")
	protected.HandleFunc("/streams/{stream_id}/start", webrtcHandler.StartStream).Methods("POST")
	protected.HandleFunc("/streams/{stream_id}/end", webrtcHandler.EndStream).Methods("POST")
	
	// Public Stream Routes
	apiV1.HandleFunc("/streams/live", webrtcHandler.GetLiveStreams).Methods("GET")
	apiV1.HandleFunc("/vods", vodHandler.GetVODList).Methods("GET")

	// VOD Management
	protected.HandleFunc("/vods", vodHandler.UploadVOD).Methods("POST")
	apiV1.HandleFunc("/vods/{vod_id}", vodHandler.GetVODDetail).Methods("GET")
	protected.HandleFunc("/vods/{vod_id}/visibility", vodHandler.UpdateVisibility).Methods("PUT")
	protected.HandleFunc("/vods/{vod_id}", vodHandler.DeleteVOD).Methods("DELETE")

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
	protected.HandleFunc("/agencies/{agency_id}/invite", monetizationHandler.InviteHost).Methods("POST")
	protected.HandleFunc("/agencies/{agency_id}/hosts", monetizationHandler.ListAgencyHosts).Methods("GET")

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

	// ===== CRYPTO ROUTES =====
	protected.HandleFunc("/crypto/deposit-address", cryptoHandler.GetDepositAddress).Methods("GET")
	protected.HandleFunc("/crypto/withdrawal", cryptoHandler.RequestWithdrawal).Methods("POST")
	router.HandleFunc("/api/v1/crypto/rates", cryptoHandler.GetExchangeRates).Methods("GET")
	router.HandleFunc("/api/v1/crypto/webhooks/{provider}", cryptoHandler.CryptoWebhook).Methods("POST")

	// Health check
	router.HandleFunc("/health", healthHandler.HealthCheck).Methods("GET")

	// Ready check
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	return router
}
