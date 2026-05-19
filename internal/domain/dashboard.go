package domain

import (
	"context"
	"time"
)

// AdminStats mewakili statistik untuk Admin Dashboard
type AdminStats struct {
	TotalUsers         int64 `json:"total_users"`
	TotalHosts         int64 `json:"total_hosts"`
	TotalAgencies      int64 `json:"total_agencies"`
	TotalActiveStreams int64 `json:"total_active_streams"`
	TotalTransactions  int64 `json:"total_transactions_today"`
	TotalBanned        int64 `json:"total_banned"`
}

// AdminRevenue mewakili pendapatan platform berdasarkan kategori
type AdminRevenue struct {
	TotalRevenue    int64 `json:"total_revenue"`
	GiftFee         int64 `json:"gift_fee"`
	PredictionFee   int64 `json:"prediction_fee"`
	SubscriptionFee int64 `json:"subscription_fee"`
	PrivateRoomFee  int64 `json:"private_room_fee"`
}

// GraphPoint mewakili titik data untuk grafik dashboard
type GraphPoint struct {
	Label string `json:"label"` // e.g. YYYY-MM-DD
	Value int64  `json:"value"`
}

// UserListFilter mewakili filter untuk pencarian user oleh admin
type UserListFilter struct {
	Role   string `json:"role"`
	Status string `json:"status"` // e.g. 'active', 'banned'
	Search string `json:"search"`
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
}

// AdminUserListItem mewakili informasi user untuk list admin
type AdminUserListItem struct {
	ID        UUID      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"` // 'active' atau 'banned'
	CreatedAt time.Time `json:"created_at"`
}

// Report mewakili pengaduan/laporan konten
type Report struct {
	ID             UUID      `json:"id"`
	ReporterID     UUID      `json:"reporter_id"`
	ReportedUserID *UUID     `json:"reported_user_id,omitempty"`
	StreamID       *UUID     `json:"stream_id,omitempty"`
	ChatMessageID  *UUID     `json:"chat_message_id,omitempty"`
	ReportType     string    `json:"report_type"` // 'explicit_content', 'violence', 'harassment', 'lgbt_content', 'other'
	Reason         string    `json:"reason"`
	Status         string    `json:"status"` // 'pending', 'reviewed', 'action_taken', 'ignored'
	ActionTaken    string    `json:"action_taken,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// HostStats mewakili statistik untuk Host Dashboard
type HostStats struct {
	TotalViews       int64 `json:"total_views"`
	TotalLikes       int64 `json:"total_likes"`
	TotalGifts       int64 `json:"total_gifts_received"`
	TotalEarnings    int64 `json:"total_earnings"`
	TotalSubscribers int64 `json:"total_subscribers"`
}

// HostRevenue mewakili rincian pendapatan host
type HostRevenue struct {
	Period        string `json:"period"` // e.g. 'daily', 'weekly', 'monthly'
	TotalEarnings int64  `json:"total_earnings"`
	GiftEarnings  int64  `json:"gift_earnings"`
	CallEarnings  int64  `json:"call_earnings"`
	RoomEarnings  int64  `json:"room_earnings"`
	TokenEarnings int64  `json:"token_earnings"`
}

// HostDashboardSettings mewakili pengaturan dashboard host
type HostDashboardSettings struct {
	AllowIncognito     bool  `json:"allow_incognito"`
	IsPrivateProfile   bool  `json:"is_private_profile"`
	DefaultCallRateIDR int64 `json:"default_call_rate_idr"`
}

// AgencyStats mewakili statistik untuk Agency Dashboard
type AgencyStats struct {
	TotalHosts          int64 `json:"total_hosts"`
	ActiveHosts         int64 `json:"active_hosts"`
	TotalEarningsIDR    int64 `json:"total_earnings_idr"`
	AgencyCommissionIDR int64 `json:"agency_commission_idr"`
}

// AgencyHostItem mewakili informasi host yang berada di bawah agency
type AgencyHostItem struct {
	HostID        UUID      `json:"host_id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"display_name"`
	JoinedAt      time.Time `json:"joined_at"`
	Status        string    `json:"status"` // 'active', 'invited'
	RevenueShare  int       `json:"revenue_share"`
	TotalEarnings int64     `json:"total_earnings"`
}

// AgencyRevenue mewakili pendapatan agency
type AgencyRevenue struct {
	Period          string `json:"period"`
	TotalEarnings   int64  `json:"total_earnings"`
	CommissionRates int    `json:"commission_rates"`
}

// AgencyDashboardSettings mewakili pengaturan dashboard agency
type AgencyDashboardSettings struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	LogoURL        string `json:"logo_url"`
	CommissionRate int    `json:"commission_rate"`
}

// DashboardRepository mendefinisikan operasi basis data untuk dashboard
type DashboardRepository interface {
	// Admin Dashboard
	GetAdminStats(ctx context.Context) (*AdminStats, error)
	GetAdminRevenue(ctx context.Context) (*AdminRevenue, error)
	GetAdminGraph(ctx context.Context, period string) ([]*GraphPoint, error)
	ListUsers(ctx context.Context, filter UserListFilter) ([]*AdminUserListItem, int64, error)
	ListReports(ctx context.Context, limit, offset int) ([]*Report, error)
	CreateReport(ctx context.Context, report *Report) error
	UpdateReportStatus(ctx context.Context, id UUID, status, actionTaken string) error
	GetReportByID(ctx context.Context, id UUID) (*Report, error)

	// Host Dashboard
	GetHostStats(ctx context.Context, hostID UUID) (*HostStats, error)
	GetHostRevenue(ctx context.Context, hostID UUID, period string) (*HostRevenue, error)
	GetHostSettings(ctx context.Context, hostID UUID) (*HostDashboardSettings, error)
	UpdateHostSettings(ctx context.Context, hostID UUID, settings *HostDashboardSettings) error
	GetHostClips(ctx context.Context, hostID UUID, limit, offset int) ([]*StreamClip, error)
	GetHostStreams(ctx context.Context, hostID UUID, limit, offset int) ([]*Stream, error)
	GetHostRequests(ctx context.Context, hostID UUID, limit, offset int) ([]*ShowRequest, error)

	// Agency Dashboard
	GetAgencyStats(ctx context.Context, ownerID UUID) (*AgencyStats, error)
	GetAgencyHosts(ctx context.Context, ownerID UUID) ([]*AgencyHostItem, error)
	GetAgencyRevenue(ctx context.Context, ownerID UUID, period string) (*AgencyRevenue, error)
	GetAgencySettings(ctx context.Context, ownerID UUID) (*AgencyDashboardSettings, error)
	UpdateAgencySettings(ctx context.Context, ownerID UUID, settings *AgencyDashboardSettings) error
	DeleteAgencyHost(ctx context.Context, agencyOwnerID UUID, hostID UUID) error
}

// DashboardUseCase mendefinisikan logika bisnis untuk dashboard
type DashboardUseCase interface {
	// Admin Dashboard
	GetAdminStats(ctx context.Context) (*AdminStats, error)
	GetAdminRevenue(ctx context.Context) (*AdminRevenue, error)
	GetAdminGraph(ctx context.Context, period string) ([]*GraphPoint, error)
	ListUsers(ctx context.Context, filter UserListFilter) ([]*AdminUserListItem, int64, error)
	ListPendingKYC(ctx context.Context, limit, offset int) ([]*KYCVerification, error)
	ListReports(ctx context.Context, limit, offset int) ([]*Report, error)
	BanUser(ctx context.Context, adminID UUID, targetUserID UUID, reason string, isPermanent bool) error
	UnbanUser(ctx context.Context, targetUserID UUID) error
	TerminateStream(ctx context.Context, adminID UUID, streamID UUID, reason string) error
	DeleteComment(ctx context.Context, adminID UUID, commentID UUID) error
	SubmitReport(ctx context.Context, reporterID UUID, reportedUserID *UUID, streamID *UUID, chatMessageID *UUID, reportType string, reason string) (*Report, error)

	// Host Dashboard
	GetHostStats(ctx context.Context, hostID UUID) (*HostStats, error)
	GetHostRevenue(ctx context.Context, hostID UUID, period string) (*HostRevenue, error)
	GetHostClips(ctx context.Context, hostID UUID, limit, offset int) ([]*StreamClip, error)
	GetHostStreams(ctx context.Context, hostID UUID, limit, offset int) ([]*Stream, error)
	GetHostRequests(ctx context.Context, hostID UUID, limit, offset int) ([]*ShowRequest, error)
	UpdateHostSettings(ctx context.Context, hostID UUID, settings *HostDashboardSettings) error

	// Agency Dashboard
	GetAgencyStats(ctx context.Context, ownerID UUID) (*AgencyStats, error)
	GetAgencyHosts(ctx context.Context, ownerID UUID) ([]*AgencyHostItem, error)
	InviteHostToAgency(ctx context.Context, ownerID UUID, hostUsername string, revenueShare int) error
	RemoveHostFromAgency(ctx context.Context, ownerID UUID, hostID UUID) error
	GetAgencyRevenue(ctx context.Context, ownerID UUID, period string) (*AgencyRevenue, error)
	UpdateAgencySettings(ctx context.Context, ownerID UUID, settings *AgencyDashboardSettings) error
}
