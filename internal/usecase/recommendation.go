package usecase

import (
	"context"
	"sort"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type recommendationUseCase struct {
	recoRepo   domain.RecommendationRepository
	streamRepo domain.StreamRepository
	vodRepo    domain.VODMediaRepository
	logger     *zap.Logger
}

// NewRecommendationUseCase membuat instance baru dari RecommendationUseCase
func NewRecommendationUseCase(
	recoRepo domain.RecommendationRepository,
	streamRepo domain.StreamRepository,
	vodRepo domain.VODMediaRepository,
	logger *zap.Logger,
) domain.RecommendationUseCaseInterface {
	return &recommendationUseCase{
		recoRepo:   recoRepo,
		streamRepo: streamRepo,
		vodRepo:    vodRepo,
		logger:     logger,
	}
}

func (uc *recommendationUseCase) TrackInteraction(ctx context.Context, userID domain.UUID, streamID *domain.UUID, interactionType string, duration int, metadata map[string]interface{}) error {
	interaction := &domain.UserInteraction{
		ID:              domain.NewUUID(),
		UserID:          userID,
		StreamID:        streamID,
		InteractionType: interactionType,
		DurationSeconds: duration,
		Metadata:        metadata,
	}

	if err := uc.recoRepo.SaveInteraction(ctx, interaction); err != nil {
		uc.logger.Error("Gagal menyimpan interaksi user", zap.Error(err))
		return err
	}

	return nil
}

func (uc *recommendationUseCase) GetRecommendedStreams(ctx context.Context, userID domain.UUID, limit int) ([]*domain.Stream, error) {
	// 1. Dapatkan daftar live stream aktif saat ini
	activeStreams, err := uc.streamRepo.ListLive(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	if len(activeStreams) == 0 {
		return activeStreams, nil
	}

	// 2. Ambil preferensi Host dan Kategori user dari DB
	hostVector, err := uc.recoRepo.GetHostPreferenceVector(ctx, userID)
	if err != nil {
		uc.logger.Warn("Gagal mendapatkan vektor preferensi host user, fallback ke popularitas", zap.Error(err))
		hostVector = make(map[domain.UUID]float64)
	}

	categoryVector, err := uc.recoRepo.GetCategoryPreferenceVector(ctx, userID)
	if err != nil {
		uc.logger.Warn("Gagal mendapatkan vektor preferensi kategori user, fallback ke popularitas", zap.Error(err))
		categoryVector = make(map[string]float64)
	}

	// 3. Hitung skor rekomendasi hibrida (Popularitas Global + Afinitas Personal)
	type streamWithScore struct {
		stream *domain.Stream
		score  float64
	}

	scoredStreams := make([]streamWithScore, len(activeStreams))
	for i, s := range activeStreams {
		// A. Skor Popularitas Global (Baseline): penonton, likes, hadiah (maksimal 50 poin agar tidak memonopoli rekomendasi)
		viewerScore := float64(s.ViewerCount) * 0.5
		likeScore := float64(s.LikeCount) * 0.1
		giftScore := s.TotalGiftValueIDR * 0.0001
		baselinePopularity := viewerScore + likeScore + giftScore
		if baselinePopularity > 50.0 {
			baselinePopularity = 50.0
		}

		// B. Skor Afinitas Personal (Personal Affinity Weight)
		hostAffinity := hostVector[s.HostID] * 2.5     // Bobot afinitas host yang disukai
		categoryAffinity := categoryVector[s.Category] * 1.5 // Bobot kesukaan kategori stream

		totalScore := baselinePopularity + hostAffinity + categoryAffinity

		scoredStreams[i] = streamWithScore{
			stream: s,
			score:  totalScore,
		}
	}

	// 4. Urutkan berdasarkan skor tertinggi (descending)
	sort.Slice(scoredStreams, func(i, j int) bool {
		return scoredStreams[i].score > scoredStreams[j].score
	})

	// 5. Kembalikan data sesuai limit yang diminta
	resultLimit := limit
	if resultLimit > len(scoredStreams) {
		resultLimit = len(scoredStreams)
	}

	result := make([]*domain.Stream, resultLimit)
	for i := 0; i < resultLimit; i++ {
		result[i] = scoredStreams[i].stream
	}

	return result, nil
}

func (uc *recommendationUseCase) GetRecommendedVODs(ctx context.Context, userID domain.UUID, limit int) ([]*domain.VODMedia, error) {
	// 1. Dapatkan daftar VOD Publik saat ini
	publicVODs, err := uc.vodRepo.ListPublic(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	if len(publicVODs) == 0 {
		return publicVODs, nil
	}

	// 2. Ambil preferensi Host user dari DB
	hostVector, err := uc.recoRepo.GetHostPreferenceVector(ctx, userID)
	if err != nil {
		uc.logger.Warn("Gagal mendapatkan vektor preferensi host user, fallback ke popularitas", zap.Error(err))
		hostVector = make(map[domain.UUID]float64)
	}

	// 3. Hitung skor rekomendasi hibrida untuk VOD
	type vodWithScore struct {
		vod   *domain.VODMedia
		score float64
	}

	scoredVODs := make([]vodWithScore, len(publicVODs))
	for i, v := range publicVODs {
		// A. Skor Baseline (Ukuran file & Durasi sebagai sinyal kelengkapan konten)
		baseline := float64(v.Duration) * 0.05
		if baseline > 20.0 {
			baseline = 20.0
		}

		// B. Skor Afinitas Personal (Rekomendasikan VOD yang diunggah oleh host favorit user)
		hostAffinity := hostVector[v.UserID] * 3.0

		totalScore := baseline + hostAffinity

		scoredVODs[i] = vodWithScore{
			vod:   v,
			score: totalScore,
		}
	}

	// 4. Urutkan berdasarkan skor tertinggi (descending)
	sort.Slice(scoredVODs, func(i, j int) bool {
		return scoredVODs[i].score > scoredVODs[j].score
	})

	// 5. Kembalikan data sesuai limit
	resultLimit := limit
	if resultLimit > len(scoredVODs) {
		resultLimit = len(scoredVODs)
	}

	result := make([]*domain.VODMedia, resultLimit)
	for i := 0; i < resultLimit; i++ {
		result[i] = scoredVODs[i].vod
	}

	return result, nil
}
