package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type InventoryUseCase struct {
	inventoryRepo domain.InventoryRepository
	logger        *zap.Logger
}

func NewInventoryUseCase(inventoryRepo domain.InventoryRepository, logger *zap.Logger) *InventoryUseCase {
	return &InventoryUseCase{inventoryRepo: inventoryRepo, logger: logger}
}

func (uc *InventoryUseCase) GetMyInventory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.UserInventory, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return uc.inventoryRepo.GetUserInventory(ctx, userID, limit, offset)
}

func (uc *InventoryUseCase) UseItem(ctx context.Context, userID, itemID domain.UUID, quantity int) error {
	if quantity <= 0 {
		return domain.NewDomainError(domain.ErrCodeValidation, "quantity must be positive", nil)
	}
	inv, err := uc.inventoryRepo.GetUserItem(ctx, userID, itemID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "item not in inventory", err)
	}
	if inv.Quantity < quantity {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient quantity", nil)
	}
	return uc.inventoryRepo.UseItem(ctx, userID, itemID, quantity)
}

func (uc *InventoryUseCase) ListCatalog(ctx context.Context, itemType string) ([]*domain.InventoryItem, error) {
	return uc.inventoryRepo.ListActiveItems(ctx, itemType)
}
