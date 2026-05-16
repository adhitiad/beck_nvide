package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type AgencyUseCase struct {
	hostAppRepo domain.HostApplicationRepository
	agencyRepo  domain.AgencyRepository
	walletUC    *WalletUseCase
	logger      *zap.Logger
}

func NewAgencyUseCase(
	hostAppRepo domain.HostApplicationRepository,
	agencyRepo domain.AgencyRepository,
	walletUC *WalletUseCase,
	logger *zap.Logger,
) *AgencyUseCase {
	return &AgencyUseCase{
		hostAppRepo: hostAppRepo,
		agencyRepo:  agencyRepo,
		walletUC:    walletUC,
		logger:      logger,
	}
}

// ApplyHost submits a host application
func (uc *AgencyUseCase) ApplyHost(ctx context.Context, userID domain.UUID, bio, idCardURL, bankName, bankAccountName, bankAccountNumber string) (*domain.HostApplication, error) {
	// Check if already applied
	existing, err := uc.hostAppRepo.GetByUserID(ctx, userID)
	if err == nil && existing != nil && existing.Status == domain.ApplicationPending {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "you already have a pending application", nil)
	}

	app := &domain.HostApplication{
		ID:                domain.NewUUID(),
		UserID:            userID,
		Bio:               bio,
		IDCardURL:         idCardURL,
		BankAccountName:   bankAccountName,
		BankAccountNumber: bankAccountNumber,
		BankName:          bankName,
		Status:            domain.ApplicationPending,
	}

	if err := uc.hostAppRepo.Create(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// ApproveHost approves a host application
func (uc *AgencyUseCase) ApproveHost(ctx context.Context, applicationID, reviewerID domain.UUID) error {
	app, err := uc.hostAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != domain.ApplicationPending {
		return domain.NewDomainError(domain.ErrCodeValidation, "application is not pending", nil)
	}

	now := time.Now()
	app.Status = domain.ApplicationApproved
	app.ReviewedBy = &reviewerID
	app.ReviewedAt = &now

	if err := uc.hostAppRepo.Update(ctx, app); err != nil {
		return err
	}

	// Also create wallet for the new host
	uc.walletUC.GetOrCreateWallet(ctx, app.UserID)
	// NOTE: Role update (user -> host) should be done via RBAC system by the caller

	return nil
}

// RejectHost rejects a host application
func (uc *AgencyUseCase) RejectHost(ctx context.Context, applicationID, reviewerID domain.UUID, reason string) error {
	app, err := uc.hostAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != domain.ApplicationPending {
		return domain.NewDomainError(domain.ErrCodeValidation, "application is not pending", nil)
	}

	now := time.Now()
	app.Status = domain.ApplicationRejected
	app.ReviewedBy = &reviewerID
	app.ReviewedAt = &now
	app.RejectionReason = reason

	return uc.hostAppRepo.Update(ctx, app)
}

// ListHostApplications returns host applications by status
func (uc *AgencyUseCase) ListHostApplications(ctx context.Context, status string, limit, offset int) ([]*domain.HostApplication, error) {
	return uc.hostAppRepo.ListByStatus(ctx, status, limit, offset)
}

// CreateAgency creates a new agency profile
func (uc *AgencyUseCase) CreateAgency(ctx context.Context, ownerID domain.UUID, name, description, logoURL string, commissionRate int) (*domain.Agency, error) {
	// Check if owner already has an agency
	existing, err := uc.agencyRepo.GetByOwnerID(ctx, ownerID)
	if err == nil && existing != nil {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "you already own an agency", nil)
	}

	if commissionRate < 10 || commissionRate > 30 {
		commissionRate = 20 // default
	}

	agency := &domain.Agency{
		ID:             domain.NewUUID(),
		OwnerID:        ownerID,
		Name:           name,
		Description:    description,
		LogoURL:        logoURL,
		CommissionRate: commissionRate,
		Status:         domain.AgencyStatusActive,
	}

	if err := uc.agencyRepo.Create(ctx, agency); err != nil {
		return nil, err
	}
	return agency, nil
}

// InviteHost sends an invitation to a host
func (uc *AgencyUseCase) InviteHost(ctx context.Context, agencyID, hostID domain.UUID, revenueShare int) error {
	// Check host isn't already in an agency
	existing, err := uc.agencyRepo.GetHostRelation(ctx, hostID)
	if err == nil && existing != nil {
		return domain.NewDomainError(domain.ErrCodeConflict, "host is already in an agency", nil)
	}

	if revenueShare < 40 || revenueShare > 80 {
		revenueShare = 60
	}

	ah := &domain.AgencyHost{
		AgencyID:     agencyID,
		HostID:       hostID,
		Status:       domain.AgencyHostInvited,
		RevenueShare: revenueShare,
	}
	return uc.agencyRepo.AddHost(ctx, ah)
}

// AcceptInvitation accepts an agency invitation
func (uc *AgencyUseCase) AcceptInvitation(ctx context.Context, agencyID, hostID domain.UUID) error {
	// For simplicity, just update status to active
	ah := &domain.AgencyHost{
		AgencyID:     agencyID,
		HostID:       hostID,
		Status:       domain.AgencyHostActive,
		RevenueShare: 60,
	}
	return uc.agencyRepo.AddHost(ctx, ah)
}

// RemoveHost removes a host from the agency
func (uc *AgencyUseCase) RemoveHost(ctx context.Context, agencyID, hostID domain.UUID) error {
	return uc.agencyRepo.RemoveHost(ctx, agencyID, hostID)
}

// ListAgencyHosts lists all active hosts in an agency
func (uc *AgencyUseCase) ListAgencyHosts(ctx context.Context, agencyID domain.UUID) ([]*domain.AgencyHost, error) {
	return uc.agencyRepo.ListHosts(ctx, agencyID)
}
