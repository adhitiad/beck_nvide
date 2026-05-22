package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type inventoryRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewInventoryRepository(db *pgxpool.Pool, logger *zap.Logger) domain.InventoryRepository {
	return &inventoryRepository{db: db, logger: logger}
}

func (r *inventoryRepository) CreateItem(ctx context.Context, item *domain.InventoryItem) error {
	query := `INSERT INTO inventory_items (id, name, type, icon_url, description, is_tradeable, is_active, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, item.ID, item.Name, item.Type, item.IconURL,
		item.Description, item.IsTradeable, item.IsActive, item.Metadata).Scan(&item.CreatedAt)
}

func (r *inventoryRepository) GetItemByID(ctx context.Context, id domain.UUID) (*domain.InventoryItem, error) {
	query := `SELECT id, name, type, icon_url, description, is_tradeable, is_active, metadata, created_at
		FROM inventory_items WHERE id=$1`
	var item domain.InventoryItem
	err := r.db.QueryRow(ctx, query, id).Scan(&item.ID, &item.Name, &item.Type, &item.IconURL,
		&item.Description, &item.IsTradeable, &item.IsActive, &item.Metadata, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *inventoryRepository) ListActiveItems(ctx context.Context, itemType string) ([]*domain.InventoryItem, error) {
	query := `SELECT id, name, type, icon_url, description, is_tradeable, is_active, metadata, created_at
		FROM inventory_items WHERE is_active=true`
	args := []interface{}{}
	if itemType != "" {
		query += ` AND type=$1`
		args = append(args, itemType)
	}
	query += ` ORDER BY created_at`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.InventoryItem
	for rows.Next() {
		var item domain.InventoryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.IconURL,
			&item.Description, &item.IsTradeable, &item.IsActive, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	return list, nil
}

func (r *inventoryRepository) AddToInventory(ctx context.Context, inv *domain.UserInventory) error {
	query := `INSERT INTO user_inventory (id, user_id, item_id, quantity, source, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, inv.ID, inv.UserID, inv.ItemID, inv.Quantity,
		inv.Source, inv.ExpiresAt).Scan(&inv.CreatedAt)
}

func (r *inventoryRepository) GetUserInventory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.UserInventory, error) {
	query := `SELECT ui.id, ui.user_id, ui.item_id, ui.quantity, ui.source, ui.expires_at, ui.created_at,
		ii.id, ii.name, ii.type, ii.icon_url, ii.description, ii.is_tradeable, ii.is_active, ii.metadata, ii.created_at
		FROM user_inventory ui
		JOIN inventory_items ii ON ui.item_id = ii.id
		WHERE ui.user_id=$1 AND ui.quantity > 0 AND (ui.expires_at IS NULL OR ui.expires_at > NOW())
		ORDER BY ui.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserInventory
	for rows.Next() {
		var ui domain.UserInventory
		var ii domain.InventoryItem
		if err := rows.Scan(&ui.ID, &ui.UserID, &ui.ItemID, &ui.Quantity, &ui.Source, &ui.ExpiresAt, &ui.CreatedAt,
			&ii.ID, &ii.Name, &ii.Type, &ii.IconURL, &ii.Description, &ii.IsTradeable, &ii.IsActive, &ii.Metadata, &ii.CreatedAt); err != nil {
			return nil, err
		}
		ui.Item = &ii
		list = append(list, &ui)
	}
	return list, nil
}

func (r *inventoryRepository) GetUserItem(ctx context.Context, userID, itemID domain.UUID) (*domain.UserInventory, error) {
	query := `SELECT id, user_id, item_id, quantity, source, expires_at, created_at
		FROM user_inventory WHERE user_id=$1 AND item_id=$2 AND quantity > 0
		AND (expires_at IS NULL OR expires_at > NOW()) LIMIT 1`
	var ui domain.UserInventory
	err := r.db.QueryRow(ctx, query, userID, itemID).Scan(&ui.ID, &ui.UserID, &ui.ItemID,
		&ui.Quantity, &ui.Source, &ui.ExpiresAt, &ui.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ui, nil
}

func (r *inventoryRepository) UseItem(ctx context.Context, userID, itemID domain.UUID, quantity int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_inventory SET quantity = quantity - $1
		WHERE user_id=$2 AND item_id=$3 AND quantity >= $1
		AND (expires_at IS NULL OR expires_at > NOW())`,
		quantity, userID, itemID)
	return err
}

func (r *inventoryRepository) RemoveExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_inventory WHERE expires_at IS NOT NULL AND expires_at < NOW()`)
	return err
}
