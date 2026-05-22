package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type RoyalFamilyUseCase struct {
	familyRepo domain.RoyalFamilyRepository
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	redis      *redis.Client
	logger     *zap.Logger
}

func NewRoyalFamilyUseCase(
	familyRepo domain.RoyalFamilyRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *RoyalFamilyUseCase {
	return &RoyalFamilyUseCase{
		familyRepo: familyRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		redis:      redis,
		logger:     logger,
	}
}

// CreateFamily creates a new royal family for a host
func (uc *RoyalFamilyUseCase) CreateFamily(ctx context.Context, hostID domain.UUID, name, description string) (*domain.RoyalFamily, error) {
	// Check if host already has a family
	existing, _ := uc.familyRepo.GetByHostID(ctx, hostID)
	if existing != nil {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "host already has a family", nil)
	}

	family := &domain.RoyalFamily{
		ID:          domain.NewUUID(),
		HostID:      hostID,
		Name:        name,
		Description: description,
		Level:       1,
		MaxMembers:  50,
	}

	if err := uc.familyRepo.Create(ctx, family); err != nil {
		return nil, err
	}

	// Host becomes owner
	member := &domain.RoyalFamilyMember{
		ID:       domain.NewUUID(),
		FamilyID: family.ID,
		UserID:   hostID,
		Role:     domain.RFRoleOwner,
	}
	_ = uc.familyRepo.AddMember(ctx, member)

	return family, nil
}

// JoinFamily allows a user to join a royal family
func (uc *RoyalFamilyUseCase) JoinFamily(ctx context.Context, userID, familyID domain.UUID) error {
	// Check if user is already in a family
	existing, _ := uc.familyRepo.GetUserFamily(ctx, userID)
	if existing != nil {
		return domain.NewDomainError(domain.ErrCodeConflict, "user already belongs to a family", nil)
	}

	family, err := uc.familyRepo.GetByID(ctx, familyID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "family not found", err)
	}

	// Check member count
	count, _ := uc.familyRepo.CountMembers(ctx, familyID)
	if count >= family.MaxMembers {
		return domain.NewDomainError(domain.ErrCodeConflict, "family is full", nil)
	}

	member := &domain.RoyalFamilyMember{
		ID:       domain.NewUUID(),
		FamilyID: familyID,
		UserID:   userID,
		Role:     domain.RFRoleMember,
	}
	return uc.familyRepo.AddMember(ctx, member)
}

// LeaveFamily removes a user from their family
func (uc *RoyalFamilyUseCase) LeaveFamily(ctx context.Context, userID domain.UUID) error {
	membership, err := uc.familyRepo.GetUserFamily(ctx, userID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "user is not in any family", err)
	}
	if membership.Role == domain.RFRoleOwner {
		return domain.NewDomainError(domain.ErrCodeForbidden, "owner cannot leave, transfer ownership first", nil)
	}
	return uc.familyRepo.RemoveMember(ctx, membership.FamilyID, userID)
}

// Contribute donates coins to the family
func (uc *RoyalFamilyUseCase) Contribute(ctx context.Context, userID domain.UUID, amount int64) (*domain.RoyalFamilyContribution, error) {
	if amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "amount must be positive", nil)
	}

	membership, err := uc.familyRepo.GetUserFamily(ctx, userID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "user is not in any family", err)
	}

	// Debit wallet
	if err := uc.walletRepo.DebitBalance(ctx, userID, amount); err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance", err)
	}

	// Record transaction
	tx := &domain.Transaction{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Type:        "family_contribution",
		Amount:      amount,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: "family_" + string(membership.FamilyID),
	}
	_ = uc.txRepo.Create(ctx, tx)

	// Add contribution
	contrib := &domain.RoyalFamilyContribution{
		ID:       domain.NewUUID(),
		FamilyID: membership.FamilyID,
		UserID:   userID,
		Amount:   amount,
		Source:   domain.RFContribDirect,
	}
	if err := uc.familyRepo.AddContribution(ctx, contrib); err != nil {
		return nil, err
	}

	// Update member contribution
	_ = uc.familyRepo.UpdateMemberContribution(ctx, membership.FamilyID, userID, amount)

	// Update family total
	family, _ := uc.familyRepo.GetByID(ctx, membership.FamilyID)
	if family != nil {
		family.TotalContribution += amount
		// Level up family based on total contribution
		newLevel := calculateFamilyLevel(family.TotalContribution)
		if newLevel > family.Level {
			family.Level = newLevel
			family.MaxMembers = 50 + (newLevel-1)*25 // Each level adds 25 member slots
		}
		_ = uc.familyRepo.Update(ctx, family)
	}

	return contrib, nil
}

func calculateFamilyLevel(totalContribution int64) int {
	switch {
	case totalContribution >= 100000000: // 100M
		return 10
	case totalContribution >= 50000000:
		return 8
	case totalContribution >= 20000000:
		return 6
	case totalContribution >= 10000000:
		return 5
	case totalContribution >= 5000000:
		return 4
	case totalContribution >= 2000000:
		return 3
	case totalContribution >= 500000:
		return 2
	default:
		return 1
	}
}

// GetFamily returns family details
func (uc *RoyalFamilyUseCase) GetFamily(ctx context.Context, familyID domain.UUID) (*domain.RoyalFamily, error) {
	return uc.familyRepo.GetByID(ctx, familyID)
}

// GetMyFamily returns the user's current family
func (uc *RoyalFamilyUseCase) GetMyFamily(ctx context.Context, userID domain.UUID) (*domain.RoyalFamily, *domain.RoyalFamilyMember, error) {
	membership, err := uc.familyRepo.GetUserFamily(ctx, userID)
	if err != nil {
		return nil, nil, nil
	}
	family, err := uc.familyRepo.GetByID(ctx, membership.FamilyID)
	if err != nil {
		return nil, nil, err
	}
	return family, membership, nil
}

// GetMembers returns members of a family
func (uc *RoyalFamilyUseCase) GetMembers(ctx context.Context, familyID domain.UUID, limit, offset int) ([]*domain.RoyalFamilyMember, error) {
	return uc.familyRepo.ListMembers(ctx, familyID, limit, offset)
}

// GetContributionLeaderboard returns top contributors in a family
func (uc *RoyalFamilyUseCase) GetContributionLeaderboard(ctx context.Context, familyID domain.UUID, limit int) ([]*domain.RoyalFamilyMember, error) {
	return uc.familyRepo.GetContributionLeaderboard(ctx, familyID, limit)
}

// GetTopFamilies returns top families by total contribution
func (uc *RoyalFamilyUseCase) GetTopFamilies(ctx context.Context, limit int) ([]*domain.RoyalFamily, error) {
	return uc.familyRepo.GetTopFamilies(ctx, limit)
}

// PromoteMember promotes a member to elder
func (uc *RoyalFamilyUseCase) PromoteMember(ctx context.Context, ownerID, memberUserID, familyID domain.UUID) error {
	owner, err := uc.familyRepo.GetMember(ctx, familyID, ownerID)
	if err != nil || owner.Role != domain.RFRoleOwner {
		return domain.NewDomainError(domain.ErrCodeForbidden, "only owner can promote members", nil)
	}
	return uc.familyRepo.UpdateMemberRole(ctx, familyID, memberUserID, domain.RFRoleElder)
}
