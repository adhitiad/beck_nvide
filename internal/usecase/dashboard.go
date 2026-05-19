package usecase

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type dashboardUseCase struct {
	dashboardRepo domain.DashboardRepository
	userRepo      domain.UserRepository
	bannedRepo    domain.BannedUserRepository
	streamRepo    domain.StreamRepository
	commentRepo   domain.CommentRepository
	kycRepo       domain.KYCRepository
	agencyRepo    domain.AgencyRepository
	logger        *zap.Logger
}

// NewDashboardUseCase membuat instance baru dari dashboardUseCase
func NewDashboardUseCase(
	dashboardRepo domain.DashboardRepository,
	userRepo domain.UserRepository,
	bannedRepo domain.BannedUserRepository,
	streamRepo domain.StreamRepository,
	commentRepo domain.CommentRepository,
	kycRepo domain.KYCRepository,
	agencyRepo domain.AgencyRepository,
	logger *zap.Logger,
) domain.DashboardUseCase {
	return &dashboardUseCase{
		dashboardRepo: dashboardRepo,
		userRepo:      userRepo,
		bannedRepo:    bannedRepo,
		streamRepo:    streamRepo,
		commentRepo:   commentRepo,
		kycRepo:       kycRepo,
		agencyRepo:    agencyRepo,
		logger:        logger,
	}
}

// === ADMIN DASHBOARD ===

func (u *dashboardUseCase) GetAdminStats(ctx context.Context) (*domain.AdminStats, error) {
	return u.dashboardRepo.GetAdminStats(ctx)
}

func (u *dashboardUseCase) GetAdminRevenue(ctx context.Context) (*domain.AdminRevenue, error) {
	return u.dashboardRepo.GetAdminRevenue(ctx)
}

func (u *dashboardUseCase) GetAdminGraph(ctx context.Context, period string) ([]*domain.GraphPoint, error) {
	if period != "daily" && period != "weekly" && period != "monthly" {
		period = "daily"
	}
	return u.dashboardRepo.GetAdminGraph(ctx, period)
}

func (u *dashboardUseCase) ListUsers(ctx context.Context, filter domain.UserListFilter) ([]*domain.AdminUserListItem, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}
	return u.dashboardRepo.ListUsers(ctx, filter)
}

func (u *dashboardUseCase) ListPendingKYC(ctx context.Context, limit, offset int) ([]*domain.KYCVerification, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.kycRepo.ListPendingKYC(ctx, limit, offset)
}

func (u *dashboardUseCase) ListReports(ctx context.Context, limit, offset int) ([]*domain.Report, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.dashboardRepo.ListReports(ctx, limit, offset)
}

func (u *dashboardUseCase) BanUser(ctx context.Context, adminID domain.UUID, targetUserID domain.UUID, reason string, isPermanent bool) error {
	// Buat data banned_user
	banned := &domain.BannedUser{
		ID:          domain.NewUUID(),
		UserID:      targetUserID,
		Reason:      reason,
		BannedAt:    time.Now(),
		IsPermanent: isPermanent,
		CanAppeal:   !isPermanent,
	}

	err := u.bannedRepo.BanUser(ctx, banned)
	if err != nil {
		return err
	}

	// Hentikan live stream jika user yang dibanned adalah host yang sedang streaming
	if stream, err := u.streamRepo.GetLiveByHost(ctx, targetUserID); err == nil && stream != nil {
		stream.Status = domain.StreamStatusEnded
		now := time.Now()
		stream.EndedAt = &now
		_ = u.streamRepo.Update(ctx, stream)
		u.logger.Info("BanUser: terminated active stream for banned host", zap.String("host_id", targetUserID.String()), zap.String("stream_id", stream.ID.String()))
	}

	return nil
}

func (u *dashboardUseCase) UnbanUser(ctx context.Context, targetUserID domain.UUID) error {
	return u.bannedRepo.UnbanUser(ctx, targetUserID)
}

func (u *dashboardUseCase) TerminateStream(ctx context.Context, adminID domain.UUID, streamID domain.UUID, reason string) error {
	stream, err := u.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if stream.Status != domain.StreamStatusLive {
		return errors.New("siaran ini tidak aktif atau sudah selesai")
	}

	stream.Status = domain.StreamStatusEnded
	now := time.Now()
	stream.EndedAt = &now

	err = u.streamRepo.Update(ctx, stream)
	if err != nil {
		return err
	}

	// Catat laporan penutupan paksa
	report := &domain.Report{
		ID:             domain.NewUUID(),
		ReporterID:     adminID,
		ReportedUserID: &stream.HostID,
		StreamID:       &stream.ID,
		ReportType:     "content_violation",
		Reason:         "Diberhentikan paksa oleh Admin: " + reason,
		Status:         "action_taken",
		ActionTaken:    "stream_terminated",
	}
	_ = u.dashboardRepo.CreateReport(ctx, report)

	return nil
}

func (u *dashboardUseCase) DeleteComment(ctx context.Context, adminID domain.UUID, commentID domain.UUID) error {
	return u.commentRepo.Delete(ctx, commentID)
}

func (u *dashboardUseCase) SubmitReport(ctx context.Context, reporterID domain.UUID, reportedUserID *domain.UUID, streamID *domain.UUID, chatMessageID *domain.UUID, reportType string, reason string) (*domain.Report, error) {
	if reportType == "" {
		reportType = "other"
	}

	// Deteksi otomatis jika konten/alasan mengandung LGBT untuk penindakan langsung (Zero Tolerance Policy)
	if reportType == "lgbt_content" || reason == "lgbt_policy" {
		if reportedUserID != nil {
			// Ban permanent langsung target user demi mematuhi kebijakan anti-LGBT nasional
			banned := &domain.BannedUser{
				ID:          domain.NewUUID(),
				UserID:      *reportedUserID,
				Reason:      "lgbt_policy",
				BannedAt:    time.Now(),
				IsPermanent: true,
				CanAppeal:   false,
			}
			_ = u.bannedRepo.BanUser(ctx, banned)

			// Tutup stream jika sedang aktif
			if streamID != nil {
				if stream, err := u.streamRepo.GetByID(ctx, *streamID); err == nil && stream != nil {
					stream.Status = domain.StreamStatusEnded
					now := time.Now()
					stream.EndedAt = &now
					_ = u.streamRepo.Update(ctx, stream)
				}
			}
		}
	}

	report := &domain.Report{
		ID:             domain.NewUUID(),
		ReporterID:     reporterID,
		ReportedUserID: reportedUserID,
		StreamID:       streamID,
		ChatMessageID:  chatMessageID,
		ReportType:     reportType,
		Reason:         reason,
		Status:         "pending",
	}

	err := u.dashboardRepo.CreateReport(ctx, report)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// === HOST DASHBOARD ===

func (u *dashboardUseCase) GetHostStats(ctx context.Context, hostID domain.UUID) (*domain.HostStats, error) {
	return u.dashboardRepo.GetHostStats(ctx, hostID)
}

func (u *dashboardUseCase) GetHostRevenue(ctx context.Context, hostID domain.UUID, period string) (*domain.HostRevenue, error) {
	if period != "daily" && period != "weekly" && period != "monthly" {
		period = "daily"
	}
	return u.dashboardRepo.GetHostRevenue(ctx, hostID, period)
}

func (u *dashboardUseCase) GetHostClips(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.StreamClip, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.dashboardRepo.GetHostClips(ctx, hostID, limit, offset)
}

func (u *dashboardUseCase) GetHostStreams(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.Stream, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.dashboardRepo.GetHostStreams(ctx, hostID, limit, offset)
}

func (u *dashboardUseCase) GetHostRequests(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*domain.ShowRequest, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.dashboardRepo.GetHostRequests(ctx, hostID, limit, offset)
}

func (u *dashboardUseCase) UpdateHostSettings(ctx context.Context, hostID domain.UUID, settings *domain.HostDashboardSettings) error {
	if settings == nil {
		return errors.New("pengaturan tidak boleh kosong")
	}
	return u.dashboardRepo.UpdateHostSettings(ctx, hostID, settings)
}

// === AGENCY DASHBOARD ===

func (u *dashboardUseCase) GetAgencyStats(ctx context.Context, ownerID domain.UUID) (*domain.AgencyStats, error) {
	return u.dashboardRepo.GetAgencyStats(ctx, ownerID)
}

func (u *dashboardUseCase) GetAgencyHosts(ctx context.Context, ownerID domain.UUID) ([]*domain.AgencyHostItem, error) {
	return u.dashboardRepo.GetAgencyHosts(ctx, ownerID)
}

func (u *dashboardUseCase) InviteHostToAgency(ctx context.Context, ownerID domain.UUID, hostUsername string, revenueShare int) error {
	agency, err := u.agencyRepo.GetByOwnerID(ctx, ownerID)
	if err != nil {
		return err
	}

	host, err := u.userRepo.GetByUsername(ctx, hostUsername)
	if err != nil {
		return err
	}

	// Periksa apakah host sudah terdaftar di hubungan agensi aktif lain
	if _, err := u.agencyRepo.GetHostRelation(ctx, host.ID); err == nil {
		return errors.New("host ini sudah terdaftar di agensi aktif lain")
	}

	// Insert status sebagai 'invited'
	ah := &domain.AgencyHost{
		AgencyID:     agency.ID,
		HostID:       host.ID,
		Status:       domain.AgencyHostInvited,
		RevenueShare: revenueShare,
	}

	return u.agencyRepo.AddHost(ctx, ah)
}

func (u *dashboardUseCase) RemoveHostFromAgency(ctx context.Context, ownerID domain.UUID, hostID domain.UUID) error {
	agency, err := u.agencyRepo.GetByOwnerID(ctx, ownerID)
	if err != nil {
		return err
	}

	return u.agencyRepo.RemoveHost(ctx, agency.ID, hostID)
}

func (u *dashboardUseCase) GetAgencyRevenue(ctx context.Context, ownerID domain.UUID, period string) (*domain.AgencyRevenue, error) {
	if period != "daily" && period != "weekly" && period != "monthly" {
		period = "daily"
	}
	return u.dashboardRepo.GetAgencyRevenue(ctx, ownerID, period)
}

func (u *dashboardUseCase) UpdateAgencySettings(ctx context.Context, ownerID domain.UUID, settings *domain.AgencyDashboardSettings) error {
	if settings == nil {
		return errors.New("pengaturan tidak boleh kosong")
	}
	return u.dashboardRepo.UpdateAgencySettings(ctx, ownerID, settings)
}
