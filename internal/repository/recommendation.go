package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type recommendationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewRecommendationRepository membuat instance baru dari RecommendationRepository
func NewRecommendationRepository(db *pgxpool.Pool, logger *zap.Logger) domain.RecommendationRepository {
	return &recommendationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *recommendationRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *recommendationRepository) SaveInteraction(ctx context.Context, interaction *domain.UserInteraction) error {
	exec := r.getExecutor(ctx)

	// Cek mode incognito di database
	var isIncognito bool
	err := exec.QueryRow(ctx, "SELECT is_incognito FROM users WHERE id = $1", interaction.UserID).Scan(&isIncognito)
	if err == nil && isIncognito {
		r.logger.Info("Interaction not saved because user is in global Incognito mode", zap.String("user_id", interaction.UserID.String()))
		return nil
	}

	// Cek mode incognito di metadata
	if interaction.Metadata != nil {
		if incog, ok := interaction.Metadata["is_incognito"]; ok {
			if b, ok := incog.(bool); ok && b {
				r.logger.Info("Interaction not saved because is_incognito flag is true in metadata", zap.String("user_id", interaction.UserID.String()))
				return nil
			}
		}
	}

	query := `INSERT INTO user_interactions (id, user_id, stream_id, interaction_type, duration_seconds, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	
	metaJSON, err := json.Marshal(interaction.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	return exec.QueryRow(ctx, query, 
		interaction.ID, 
		interaction.UserID, 
		interaction.StreamID, 
		interaction.InteractionType, 
		interaction.DurationSeconds, 
		metaJSON,
	).Scan(&interaction.CreatedAt)
}

func (r *recommendationRepository) GetUserInteractions(ctx context.Context, userID domain.UUID, limit int) ([]*domain.UserInteraction, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, stream_id, interaction_type, duration_seconds, metadata, created_at 
		FROM user_interactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	
	rows, err := exec.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserInteraction
	for rows.Next() {
		var inter domain.UserInteraction
		var metaBytes []byte
		err := rows.Scan(&inter.ID, &inter.UserID, &inter.StreamID, &inter.InteractionType, &inter.DurationSeconds, &metaBytes, &inter.CreatedAt)
		if err != nil {
			return nil, err
		}
		
		_ = json.Unmarshal(metaBytes, &inter.Metadata)
		list = append(list, &inter)
	}

	return list, nil
}

func (r *recommendationRepository) GetHostPreferenceVector(ctx context.Context, userID domain.UUID) (map[domain.UUID]float64, error) {
	exec := r.getExecutor(ctx)
	// Query menghitung preferensi host menggunakan bobot tipe interaksi dan time-decay (laju peluruhan 10% per hari)
	query := `
		SELECT s.host_id, 
		       SUM(
		           CASE 
		               WHEN ui.interaction_type = 'gift' THEN 50
		               WHEN ui.interaction_type = 'comment' THEN 15
		               WHEN ui.interaction_type = 'like' THEN 10
		               WHEN ui.interaction_type = 'watch' THEN LEAST(ui.duration_seconds * 0.1, 100.0)
		               ELSE 5
		           END * EXP(-0.1 * EXTRACT(DAY FROM NOW() - ui.created_at))
		       ) as preference_score
		FROM user_interactions ui
		JOIN streams s ON ui.stream_id = s.id
		WHERE ui.user_id = $1
		GROUP BY s.host_id
		ORDER BY preference_score DESC
	`
	rows, err := exec.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vector := make(map[domain.UUID]float64)
	for rows.Next() {
		var hostID domain.UUID
		var score float64
		if err := rows.Scan(&hostID, &score); err != nil {
			return nil, err
		}
		vector[hostID] = score
	}

	return vector, nil
}

func (r *recommendationRepository) GetCategoryPreferenceVector(ctx context.Context, userID domain.UUID) (map[string]float64, error) {
	exec := r.getExecutor(ctx)
	// Query menghitung preferensi kategori stream menggunakan bobot interaksi dan time-decay
	query := `
		SELECT s.category, 
		       SUM(
		           CASE 
		               WHEN ui.interaction_type = 'gift' THEN 50
		               WHEN ui.interaction_type = 'comment' THEN 15
		               WHEN ui.interaction_type = 'like' THEN 10
		               WHEN ui.interaction_type = 'watch' THEN LEAST(ui.duration_seconds * 0.1, 100.0)
		               ELSE 5
		           END * EXP(-0.1 * EXTRACT(DAY FROM NOW() - ui.created_at))
		       ) as category_score
		FROM user_interactions ui
		JOIN streams s ON ui.stream_id = s.id
		WHERE ui.user_id = $1 AND s.category IS NOT NULL AND s.category <> ''
		GROUP BY s.category
		ORDER BY category_score DESC
	`
	rows, err := exec.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vector := make(map[string]float64)
	for rows.Next() {
		var category string
		var score float64
		if err := rows.Scan(&category, &score); err != nil {
			return nil, err
		}
		vector[category] = score
	}

	return vector, nil
}
