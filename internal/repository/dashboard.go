package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type dashboardRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewDashboardRepository membuat instance baru dari dashboardRepository
func NewDashboardRepository(db *pgxpool.Pool, logger *zap.Logger) domain.DashboardRepository {
	return &dashboardRepository{
		db:     db,
		logger: logger,
	}
}

// === ADMIN DASHBOARD IMPLEMENTATION ===

func (r *dashboardRepository) GetAdminStats(ctx context.Context) (*domain.AdminStats, error) {
	stats := &domain.AdminStats{}

	// 1. Total Users
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count users", zap.Error(err))
		return nil, err
	}

	// 2. Total Hosts
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users u 
		JOIN roles r ON u.role_id = r.id 
		WHERE r.name = 'host'
	`).Scan(&stats.TotalHosts)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count hosts", zap.Error(err))
		return nil, err
	}

	// 3. Total Agencies
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM agencies").Scan(&stats.TotalAgencies)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count agencies", zap.Error(err))
		return nil, err
	}

	// 4. Total Active Streams
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM streams WHERE status = 'live'").Scan(&stats.TotalActiveStreams)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count active streams", zap.Error(err))
		return nil, err
	}

	// 5. Total Transactions Today
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions 
		WHERE status = 'success' AND created_at >= CURRENT_DATE
	`).Scan(&stats.TotalTransactions)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count transactions today", zap.Error(err))
		return nil, err
	}

	// 6. Total Banned Users
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM banned_users").Scan(&stats.TotalBanned)
	if err != nil {
		r.logger.Error("GetAdminStats: failed to count banned users", zap.Error(err))
		return nil, err
	}

	return stats, nil
}

func (r *dashboardRepository) GetAdminRevenue(ctx context.Context) (*domain.AdminRevenue, error) {
	rev := &domain.AdminRevenue{}

	// 1. Gift Fee
	err := r.db.QueryRow(ctx, "SELECT COALESCE(SUM(platform_fee), 0) FROM gift_transactions").Scan(&rev.GiftFee)
	if err != nil {
		r.logger.Error("GetAdminRevenue: failed to calculate gift platform fee", zap.Error(err))
		return nil, err
	}

	// 2. Prediction Fee (5% from resolved prediction pools)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CAST((total_yes_pool + total_no_pool) * 0.05 AS BIGINT)), 0) 
		FROM predictions 
		WHERE status = 'resolved' AND (total_yes_pool + total_no_pool) > 0
	`).Scan(&rev.PredictionFee)
	if err != nil {
		r.logger.Error("GetAdminRevenue: failed to calculate prediction platform fee", zap.Error(err))
		return nil, err
	}

	// 3. Subscription Fee (100% of vip_subscription goes to platform)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE type = 'vip_subscription' AND status = 'success'
	`).Scan(&rev.SubscriptionFee)
	if err != nil {
		r.logger.Error("GetAdminRevenue: failed to calculate subscription fee", zap.Error(err))
		return nil, err
	}

	// 4. Private Room Fee (assume 10% platform share of paid room entries)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CAST(ABS(amount) * 0.10 AS BIGINT)), 0) FROM transactions 
		WHERE type = 'paid_room_entry' AND status = 'success'
	`).Scan(&rev.PrivateRoomFee)
	if err != nil {
		r.logger.Error("GetAdminRevenue: failed to calculate private room fee", zap.Error(err))
		return nil, err
	}

	rev.TotalRevenue = rev.GiftFee + rev.PredictionFee + rev.SubscriptionFee + rev.PrivateRoomFee
	return rev, nil
}

func (r *dashboardRepository) GetAdminGraph(ctx context.Context, period string) ([]*domain.GraphPoint, error) {
	var interval string
	var labelFormat string

	switch period {
	case "weekly":
		interval = "12 weeks"
		labelFormat = "IYYY-IW"
	case "monthly":
		interval = "12 months"
		labelFormat = "YYYY-MM"
	default: // daily
		interval = "30 days"
		labelFormat = "YYYY-MM-DD"
	}

	query := fmt.Sprintf(`
		SELECT label, SUM(val) as value FROM (
			SELECT TO_CHAR(created_at, '%s') AS label, COALESCE(SUM(platform_fee), 0) AS val
			FROM gift_transactions
			WHERE created_at >= NOW() - INTERVAL '%s'
			GROUP BY label
			
			UNION ALL
			
			SELECT TO_CHAR(created_at, '%s') AS label, COALESCE(SUM(amount), 0) AS val
			FROM transactions
			WHERE type = 'vip_subscription' AND status = 'success' AND created_at >= NOW() - INTERVAL '%s'
			GROUP BY label
		) t
		GROUP BY label
		ORDER BY label ASC
	`, labelFormat, interval, labelFormat, interval)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("GetAdminGraph: query error", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	points := make([]*domain.GraphPoint, 0)
	for rows.Next() {
		p := &domain.GraphPoint{}
		if err := rows.Scan(&p.Label, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, nil
}

func (r *dashboardRepository) ListUsers(ctx context.Context, filter domain.UserListFilter) ([]*domain.AdminUserListItem, int64, error) {
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	countQuery := `
		SELECT COUNT(*) 
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN banned_users b ON u.id = b.user_id
		WHERE 1=1
	`
	dataQuery := `
		SELECT u.id, u.username, u.email, COALESCE(r.name, '') as role, 
		       CASE WHEN b.id IS NOT NULL THEN 'banned' ELSE 'active' END as status,
		       u.created_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN banned_users b ON u.id = b.user_id
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filter.Role != "" {
		countQuery += fmt.Sprintf(" AND r.name = $%d", argIndex)
		dataQuery += fmt.Sprintf(" AND r.name = $%d", argIndex)
		args = append(args, filter.Role)
		argIndex++
	}

	if filter.Status != "" {
		if filter.Status == "banned" {
			countQuery += " AND b.id IS NOT NULL"
			dataQuery += " AND b.id IS NOT NULL"
		} else {
			countQuery += " AND b.id IS NULL"
			dataQuery += " AND b.id IS NULL"
		}
	}

	if filter.Search != "" {
		countQuery += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d)", argIndex, argIndex)
		dataQuery += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	// Count total
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&count)
	if err != nil {
		r.logger.Error("ListUsers: failed to count filtered users", zap.Error(err))
		return nil, 0, err
	}

	// Fetch page
	dataQuery += fmt.Sprintf(" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		r.logger.Error("ListUsers: failed to fetch page data", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*domain.AdminUserListItem, 0)
	for rows.Next() {
		u := &domain.AdminUserListItem{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, count, nil
}

func (r *dashboardRepository) ListReports(ctx context.Context, limit, offset int) ([]*domain.Report, error) {
	query := `
		SELECT id, reporter_id, reported_user_id, stream_id, chat_message_id, report_type, reason, status, COALESCE(action_taken, '') as action_taken, created_at 
		FROM reports 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("ListReports: failed to fetch reports", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	reports := make([]*domain.Report, 0)
	for rows.Next() {
		rep := &domain.Report{}
		if err := rows.Scan(
			&rep.ID, &rep.ReporterID, &rep.ReportedUserID, &rep.StreamID, &rep.ChatMessageID,
			&rep.ReportType, &rep.Reason, &rep.Status, &rep.ActionTaken, &rep.CreatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}

	return reports, nil
}

func (r *dashboardRepository) CreateReport(ctx context.Context, report *domain.Report) error {
	query := `
		INSERT INTO reports (id, reporter_id, reported_user_id, stream_id, chat_message_id, report_type, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		report.ID,
		report.ReporterID,
		report.ReportedUserID,
		report.StreamID,
		report.ChatMessageID,
		report.ReportType,
		report.Reason,
		report.Status,
	)
	if err != nil {
		r.logger.Error("CreateReport: failed insert report", zap.Error(err))
		return err
	}
	return nil
}

func (r *dashboardRepository) UpdateReportStatus(ctx context.Context, id domain.UUID, status, actionTaken string) error {
	query := `UPDATE reports SET status = $1, action_taken = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, actionTaken, id)
	if err != nil {
		r.logger.Error("UpdateReportStatus: update error", zap.Error(err))
		return err
	}
	return nil
}

func (r *dashboardRepository) GetReportByID(ctx context.Context, id domain.UUID) (*domain.Report, error) {
	query := `
		SELECT id, reporter_id, reported_user_id, stream_id, chat_message_id, report_type, reason, status, COALESCE(action_taken, '') as action_taken, created_at 
		FROM reports 
		WHERE id = $1
	`
	rep := &domain.Report{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rep.ID, &rep.ReporterID, &rep.ReportedUserID, &rep.StreamID, &rep.ChatMessageID,
		&rep.ReportType, &rep.Reason, &rep.Status, &rep.ActionTaken, &rep.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("GetReportByID: scan error", zap.Error(err))
		return nil, err
	}
	return rep, nil
}

// === HOST DASHBOARD IMPLEMENTATION ===

func (r *dashboardRepository) GetHostStats(ctx context.Context, hostID domain.UUID) (*domain.HostStats, error) {
	stats := &domain.HostStats{}

	// 1. Total Views (Peak viewer sum across ended streams)
	err := r.db.QueryRow(ctx, "SELECT COALESCE(SUM(viewer_peak), 0) FROM streams WHERE host_id = $1", hostID).Scan(&stats.TotalViews)
	if err != nil {
		r.logger.Error("GetHostStats: failed to sum viewer_peak", zap.Error(err))
		return nil, err
	}

	// 2. Total Likes
	err = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(like_count), 0) FROM streams WHERE host_id = $1", hostID).Scan(&stats.TotalLikes)
	if err != nil {
		r.logger.Error("GetHostStats: failed to sum stream likes", zap.Error(err))
		return nil, err
	}

	// 3. Total Gifts Received
	err = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(quantity), 0) FROM gift_transactions WHERE receiver_id = $1", hostID).Scan(&stats.TotalGifts)
	if err != nil {
		r.logger.Error("GetHostStats: failed to sum gifts", zap.Error(err))
		return nil, err
	}

	// 4. Total Earnings (Sum of Host Earnings transactions)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE user_id = $1 AND type = 'host_earning' AND status = 'success'
	`).Scan(&stats.TotalEarnings)
	if err != nil {
		r.logger.Error("GetHostStats: failed to sum host earnings", zap.Error(err))
		return nil, err
	}

	// 5. Total Subscribers (Holders of Creator Tokens)
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ut.user_id) 
		FROM user_tokens ut 
		JOIN creator_tokens ct ON ut.token_id = ct.id 
		WHERE ct.host_id = $1
	`).Scan(&stats.TotalSubscribers)
	if err != nil {
		r.logger.Error("GetHostStats: failed to count subscribers", zap.Error(err))
		return nil, err
	}

	return stats, nil
}

func (r *dashboardRepository) GetHostRevenue(ctx context.Context, hostID domain.UUID, period string) (*domain.HostRevenue, error) {
	rev := &domain.HostRevenue{Period: period}
	var startTime time.Time

	switch period {
	case "weekly":
		startTime = time.Now().AddDate(0, 0, -7)
	case "monthly":
		startTime = time.Now().AddDate(0, 0, -30)
	default: // daily
		startTime = time.Now().AddDate(0, 0, -1)
	}

	// 1. Gift Earnings
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(host_earning), 0) FROM gift_transactions 
		WHERE receiver_id = $1 AND created_at >= $2
	`, hostID, startTime).Scan(&rev.GiftEarnings)
	if err != nil {
		return nil, err
	}

	// 2. Call Earnings
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE user_id = $1 AND type = 'paid_call_tick' AND status = 'success' AND created_at >= $2
	`, hostID, startTime).Scan(&rev.CallEarnings)
	if err != nil {
		return nil, err
	}

	// 3. Room Earnings (Private Paid Room Entries)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE user_id = $1 AND type = 'host_earning' AND metadata LIKE '%room_id%' AND status = 'success' AND created_at >= $2
	`, hostID, startTime).Scan(&rev.RoomEarnings)
	if err != nil {
		return nil, err
	}

	// 4. Token Earnings (Creator Token purchases)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE user_id = $1 AND type = 'creator_token_buy' AND status = 'success' AND created_at >= $2
	`, hostID, startTime).Scan(&rev.TokenEarnings)
	if err != nil {
		return nil, err
	}

	rev.TotalEarnings = rev.GiftEarnings + rev.CallEarnings + rev.RoomEarnings + rev.TokenEarnings
	return rev, nil
}

func (r *dashboardRepository) GetHostSettings(ctx context.Context, hostID domain.UUID) (*domain.HostDashboardSettings, error) {
	// Look up private profile, and default call rates if available
	settings := &domain.HostDashboardSettings{
		AllowIncognito:     false,
		IsPrivateProfile:   false,
		DefaultCallRateIDR: 1000,
	}

	err := r.db.QueryRow(ctx, "SELECT is_private_profile FROM users WHERE id = $1", hostID).Scan(&settings.IsPrivateProfile)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Fetch default call rate if there's call rate table
	var rate int64
	err = r.db.QueryRow(ctx, "SELECT rate_per_minute FROM host_call_rates WHERE host_id = $1 LIMIT 1", hostID).Scan(&rate)
	if err == nil {
		settings.DefaultCallRateIDR = rate
	}

	return settings, nil
}

func (r *dashboardRepository) UpdateHostSettings(ctx context.Context, hostID domain.UUID, settings *domain.HostDashboardSettings) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET is_private_profile = $1 WHERE id = $2", settings.IsPrivateProfile, hostID)
	if err != nil {
		return err
	}

	// Upsert call rates
	_, err = r.db.Exec(ctx, `
		INSERT INTO host_call_rates (id, host_id, rate_per_minute, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (host_id) DO UPDATE SET rate_per_minute = EXCLUDED.rate_per_minute, updated_at = NOW()
	`, domain.NewUUID(), hostID, settings.DefaultCallRateIDR)

	return err
}

func (r *dashboardRepository) GetHostClips(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.StreamClip, error) {
	query := `
		SELECT sc.id, sc.stream_id, sc.title, sc.clip_url, sc.duration, sc.score, sc.created_at
		FROM stream_clips sc
		JOIN streams s ON sc.stream_id = s.id
		WHERE s.host_id = $1
		ORDER BY sc.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, hostID, limit, offset)
	if err != nil {
		r.logger.Error("GetHostClips: failed to fetch clips", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	clips := make([]*domain.StreamClip, 0)
	for rows.Next() {
		c := &domain.StreamClip{}
		if err := rows.Scan(&c.ID, &c.StreamID, &c.Title, &c.ClipURL, &c.Duration, &c.Score, &c.CreatedAt); err != nil {
			return nil, err
		}
		clips = append(clips, c)
	}
	return clips, nil
}

func (r *dashboardRepository) GetHostStreams(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.Stream, error) {
	query := `
		SELECT id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id, created_at, updated_at
		FROM streams
		WHERE host_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, hostID, limit, offset)
	if err != nil {
		r.logger.Error("GetHostStreams: failed to fetch streams", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	streams := make([]*domain.Stream, 0)
	for rows.Next() {
		s := &domain.Stream{}
		err := rows.Scan(
			&s.ID, &s.HostID, &s.Title, &s.Description, &s.ThumbnailURL, &s.Status,
			&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		streams = append(streams, s)
	}
	return streams, nil
}

func (r *dashboardRepository) GetHostRequests(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.ShowRequest, error) {
	query := `
		SELECT sr.id, sr.stream_id, sr.user_id, sr.description, sr.tips_amount, sr.status, sr.created_at
		FROM show_requests sr
		JOIN streams s ON sr.stream_id = s.id
		WHERE s.host_id = $1
		ORDER BY sr.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, hostID, limit, offset)
	if err != nil {
		r.logger.Error("GetHostRequests: failed to fetch show requests", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	requests := make([]*domain.ShowRequest, 0)
	for rows.Next() {
		sr := &domain.ShowRequest{}
		if err := rows.Scan(&sr.ID, &sr.StreamID, &sr.UserID, &sr.Description, &sr.TipsAmount, &sr.Status, &sr.CreatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, sr)
	}
	return requests, nil
}

// === AGENCY DASHBOARD IMPLEMENTATION ===

func (r *dashboardRepository) GetAgencyStats(ctx context.Context, ownerID domain.UUID) (*domain.AgencyStats, error) {
	stats := &domain.AgencyStats{}

	// Retrieve Agency ID
	var agencyID domain.UUID
	err := r.db.QueryRow(ctx, "SELECT id FROM agencies WHERE owner_id = $1 AND status = 'active'", ownerID).Scan(&agencyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("agency tidak ditemukan atau nonaktif untuk pemilik ini")
		}
		return nil, err
	}

	// 1. Total Hosts
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM agency_hosts WHERE agency_id = $1 AND status = 'active'", agencyID).Scan(&stats.TotalHosts)
	if err != nil {
		return nil, err
	}

	// 2. Active Hosts
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT s.host_id) 
		FROM agency_hosts ah 
		JOIN streams s ON ah.host_id = s.host_id 
		WHERE ah.agency_id = $1 AND ah.status = 'active' AND s.status = 'live'
	`, agencyID).Scan(&stats.ActiveHosts)
	if err != nil {
		return nil, err
	}

	// 3. Total Earnings IDR
	err = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(total_earnings), 0) FROM agency_hosts WHERE agency_id = $1", agencyID).Scan(&stats.TotalEarningsIDR)
	if err != nil {
		return nil, err
	}

	// 4. Agency Commission IDR
	err = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(agency_commission), 0) FROM gift_transactions WHERE agency_id = $1", agencyID).Scan(&stats.AgencyCommissionIDR)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *dashboardRepository) GetAgencyHosts(ctx context.Context, ownerID domain.UUID) ([]*domain.AgencyHostItem, error) {
	var agencyID domain.UUID
	err := r.db.QueryRow(ctx, "SELECT id FROM agencies WHERE owner_id = $1 AND status = 'active'", ownerID).Scan(&agencyID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ah.host_id, u.username, COALESCE(u.display_name, u.username) as display_name, ah.joined_at, ah.status, ah.revenue_share, ah.total_earnings
		FROM agency_hosts ah
		JOIN users u ON ah.host_id = u.id
		WHERE ah.agency_id = $1
		ORDER BY ah.joined_at DESC
	`
	rows, err := r.db.Query(ctx, query, agencyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hosts := make([]*domain.AgencyHostItem, 0)
	for rows.Next() {
		h := &domain.AgencyHostItem{}
		if err := rows.Scan(&h.HostID, &h.Username, &h.DisplayName, &h.JoinedAt, &h.Status, &h.RevenueShare, &h.TotalEarnings); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}

	return hosts, nil
}

func (r *dashboardRepository) GetAgencyRevenue(ctx context.Context, ownerID domain.UUID, period string) (*domain.AgencyRevenue, error) {
	var agencyID domain.UUID
	var commissionRate int
	err := r.db.QueryRow(ctx, "SELECT id, commission_rate FROM agencies WHERE owner_id = $1 AND status = 'active'", ownerID).Scan(&agencyID, &commissionRate)
	if err != nil {
		return nil, err
	}

	rev := &domain.AgencyRevenue{
		Period:          period,
		CommissionRates: commissionRate,
	}

	var startTime time.Time
	switch period {
	case "weekly":
		startTime = time.Now().AddDate(0, 0, -7)
	case "monthly":
		startTime = time.Now().AddDate(0, 0, -30)
	default:
		startTime = time.Now().AddDate(0, 0, -1)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(agency_commission), 0) FROM gift_transactions 
		WHERE agency_id = $1 AND created_at >= $2
	`, agencyID, startTime).Scan(&rev.TotalEarnings)
	if err != nil {
		return nil, err
	}

	return rev, nil
}

func (r *dashboardRepository) GetAgencySettings(ctx context.Context, ownerID domain.UUID) (*domain.AgencyDashboardSettings, error) {
	settings := &domain.AgencyDashboardSettings{}
	query := "SELECT name, COALESCE(description, '') as description, COALESCE(logo_url, '') as logo_url, commission_rate FROM agencies WHERE owner_id = $1 LIMIT 1"
	err := r.db.QueryRow(ctx, query, ownerID).Scan(&settings.Name, &settings.Description, &settings.LogoURL, &settings.CommissionRate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return settings, nil
}

func (r *dashboardRepository) UpdateAgencySettings(ctx context.Context, ownerID domain.UUID, settings *domain.AgencyDashboardSettings) error {
	query := `
		UPDATE agencies 
		SET name = $1, description = $2, logo_url = $3, commission_rate = $4, updated_at = NOW()
		WHERE owner_id = $5
	`
	_, err := r.db.Exec(ctx, query, settings.Name, settings.Description, settings.LogoURL, settings.CommissionRate, ownerID)
	return err
}

func (r *dashboardRepository) DeleteAgencyHost(ctx context.Context, agencyOwnerID domain.UUID, hostID domain.UUID) error {
	var agencyID domain.UUID
	err := r.db.QueryRow(ctx, "SELECT id FROM agencies WHERE owner_id = $1 AND status = 'active'", agencyOwnerID).Scan(&agencyID)
	if err != nil {
		return err
	}

	query := "DELETE FROM agency_hosts WHERE agency_id = $1 AND host_id = $2"
	res, err := r.db.Exec(ctx, query, agencyID, hostID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
