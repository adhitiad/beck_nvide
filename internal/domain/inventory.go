package domain

import (
	"context"
	"time"
)

// Inventory item types
const (
	InventoryTypeGift       = "gift"
	InventoryTypeVoucher    = "voucher"
	InventoryTypeEffect     = "effect"
	InventoryTypeBadgeFrame = "badge_frame"
	InventoryTypeChatBubble = "chat_bubble"
)

// Inventory sources
const (
	InventorySourcePurchase = "purchase"
	InventorySourceWheel    = "wheel"
	InventorySourceMission  = "mission"
	InventorySourceGift     = "gift"
	InventorySourceAdmin    = "admin"
)

// InventoryItem represents a catalog item that can be stored in user's backpack
type InventoryItem struct {
	ID          UUID      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // gift, voucher, effect, badge_frame, chat_bubble
	IconURL     string    `json:"icon_url"`
	Description string    `json:"description"`
	IsTradeable bool      `json:"is_tradeable"`
	IsActive    bool      `json:"is_active"`
	Metadata    string    `json:"metadata"` // JSONB
	CreatedAt   time.Time `json:"created_at"`
}

// UserInventory represents an item owned by a user in their backpack
type UserInventory struct {
	ID        UUID       `json:"id"`
	UserID    UUID       `json:"user_id"`
	ItemID    UUID       `json:"item_id"`
	Quantity  int        `json:"quantity"`
	Source    string     `json:"source"` // purchase, wheel, mission, gift, admin
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// Relations
	Item *InventoryItem `json:"item,omitempty"`
}

// IsExpired checks if the inventory item has expired
func (ui *UserInventory) IsExpired() bool {
	if ui.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ui.ExpiresAt)
}

// InventoryRepository defines the contract for inventory data access
type InventoryRepository interface {
	// Items catalog
	CreateItem(ctx context.Context, item *InventoryItem) error
	GetItemByID(ctx context.Context, id UUID) (*InventoryItem, error)
	ListActiveItems(ctx context.Context, itemType string) ([]*InventoryItem, error)

	// User inventory
	AddToInventory(ctx context.Context, inv *UserInventory) error
	GetUserInventory(ctx context.Context, userID UUID, limit, offset int) ([]*UserInventory, error)
	GetUserItem(ctx context.Context, userID, itemID UUID) (*UserInventory, error)
	UseItem(ctx context.Context, userID, itemID UUID, quantity int) error
	RemoveExpired(ctx context.Context) error
}
