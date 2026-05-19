package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type monetizationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewMonetizationRepository creates a new instance of MonetizationRepository
func NewMonetizationRepository(db *pgxpool.Pool, logger *zap.Logger) domain.MonetizationRepository {
	return &monetizationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *monetizationRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

// Paid Room
func (r *monetizationRepository) CreatePaidRoom(ctx context.Context, room *domain.PaidRoom) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO paid_rooms (id, host_id, name, entry_fee_idr, created_at)
		VALUES ($1, $2, $3, $4, NOW()) RETURNING created_at`
	
	return exec.QueryRow(ctx, query, room.ID, room.HostID, room.Name, room.EntryFeeIDR).Scan(&room.CreatedAt)
}

func (r *monetizationRepository) GetPaidRoomByID(ctx context.Context, id domain.UUID) (*domain.PaidRoom, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, name, entry_fee_idr, created_at FROM paid_rooms WHERE id = $1`
	
	var room domain.PaidRoom
	err := exec.QueryRow(ctx, query, id).Scan(&room.ID, &room.HostID, &room.Name, &room.EntryFeeIDR, &room.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

func (r *monetizationRepository) ListPaidRoomsByHost(ctx context.Context, hostID domain.UUID) ([]*domain.PaidRoom, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, name, entry_fee_idr, created_at FROM paid_rooms WHERE host_id = $1 ORDER BY created_at DESC`
	
	rows, err := exec.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.PaidRoom
	for rows.Next() {
		var room domain.PaidRoom
		err := rows.Scan(&room.ID, &room.HostID, &room.Name, &room.EntryFeeIDR, &room.CreatedAt)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}
	return rooms, nil
}

// Interactive Toys
func (r *monetizationRepository) SaveHostDevice(ctx context.Context, device *domain.HostDevice) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO host_devices (id, host_id, device_name, device_id, api_token, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (host_id, device_id) 
		DO UPDATE SET device_name = $3, api_token = $5 RETURNING created_at`
	
	return exec.QueryRow(ctx, query, device.ID, device.HostID, device.DeviceName, device.DeviceID, device.APIToken).Scan(&device.CreatedAt)
}

func (r *monetizationRepository) GetHostDevices(ctx context.Context, hostID domain.UUID) ([]*domain.HostDevice, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, device_name, device_id, api_token, created_at FROM host_devices WHERE host_id = $1`
	
	rows, err := exec.Query(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*domain.HostDevice
	for rows.Next() {
		var d domain.HostDevice
		err := rows.Scan(&d.ID, &d.HostID, &d.DeviceName, &d.DeviceID, &d.APIToken, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &d)
	}
	return devices, nil
}

// Show Request
func (r *monetizationRepository) CreateShowRequest(ctx context.Context, req *domain.ShowRequest) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO show_requests (id, stream_id, user_id, description, tips_amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	
	return exec.QueryRow(ctx, query, req.ID, req.StreamID, req.UserID, req.Description, req.TipsAmount, req.Status).Scan(&req.CreatedAt)
}

func (r *monetizationRepository) GetShowRequestByID(ctx context.Context, id domain.UUID) (*domain.ShowRequest, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, user_id, description, tips_amount, status, created_at FROM show_requests WHERE id = $1`
	
	var req domain.ShowRequest
	err := exec.QueryRow(ctx, query, id).Scan(&req.ID, &req.StreamID, &req.UserID, &req.Description, &req.TipsAmount, &req.Status, &req.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *monetizationRepository) UpdateShowRequestStatus(ctx context.Context, id domain.UUID, status string) error {
	exec := r.getExecutor(ctx)
	query := `UPDATE show_requests SET status = $2 WHERE id = $1`
	
	_, err := exec.Exec(ctx, query, id, status)
	return err
}

// AI Companion
func (r *monetizationRepository) CreateAIChatSession(ctx context.Context, sess *domain.AIChatSession) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO ai_chat_sessions (id, user_id, host_id, created_at)
		VALUES ($1, $2, $3, NOW()) RETURNING created_at`
	
	return exec.QueryRow(ctx, query, sess.ID, sess.UserID, sess.HostID).Scan(&sess.CreatedAt)
}

func (r *monetizationRepository) GetAIChatSession(ctx context.Context, userID, hostID domain.UUID) (*domain.AIChatSession, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, host_id, created_at FROM ai_chat_sessions WHERE user_id = $1 AND host_id = $2`
	
	var sess domain.AIChatSession
	err := exec.QueryRow(ctx, query, userID, hostID).Scan(&sess.ID, &sess.UserID, &sess.HostID, &sess.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sess, nil
}

func (r *monetizationRepository) SaveAIChatMessage(ctx context.Context, msg *domain.AIChatMessage) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO ai_chat_messages (id, session_id, sender_type, content, created_at)
		VALUES ($1, $2, $3, $4, NOW()) RETURNING created_at`
	
	return exec.QueryRow(ctx, query, msg.ID, msg.SessionID, msg.SenderType, msg.Content).Scan(&msg.CreatedAt)
}

func (r *monetizationRepository) GetAIChatHistory(ctx context.Context, sessionID domain.UUID, limit int) ([]*domain.AIChatMessage, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, session_id, sender_type, content, created_at FROM ai_chat_messages WHERE session_id = $1 ORDER BY created_at ASC LIMIT $2`
	
	rows, err := exec.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.AIChatMessage
	for rows.Next() {
		var msg domain.AIChatMessage
		err := rows.Scan(&msg.ID, &msg.SessionID, &msg.SenderType, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &msg)
	}
	return list, nil
}

func (r *monetizationRepository) GetHostChatHistory(ctx context.Context, hostID domain.UUID, limit int) ([]string, error) {
	exec := r.getExecutor(ctx)
	// Query public live stream chat history sent by the host
	query := `SELECT content FROM live_messages WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	
	rows, err := exec.Query(ctx, query, hostID, limit)
	if err != nil {
		// Fallback to table messages if live_messages doesn't yield anything
		query = `SELECT content FROM messages WHERE sender_id = $1 AND content IS NOT NULL ORDER BY created_at DESC LIMIT $2`
		rows, err = exec.Query(ctx, query, hostID, limit)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var content sql.NullString
		if err := rows.Scan(&content); err == nil && content.Valid && content.String != "" {
			history = append(history, content.String)
		}
	}
	return history, nil
}
