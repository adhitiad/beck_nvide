package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"nvide-live/internal/domain"
	"nvide-live/pkg/ffmpeg"
	"nvide-live/pkg/redis"
	"nvide-live/pkg/storage"
	"nvide-live/pkg/worker"
)

// VODTranscodePayload represents the background transcode job payload
type VODTranscodePayload struct {
	VODID            string `json:"vod_id"`
	TempFilePath     string `json:"temp_file_path"`
	OriginalFileName string `json:"original_file_name"`
	UserID           string `json:"user_id"`
}

type VODUseCase struct {
	vodRepo     domain.VODMediaRepository
	drmRepo     domain.DRMRepository
	ffmpeg      *ffmpeg.FFmpeg
	storage     storage.Storage
	redisClient *redis.Client
	workerPool  *worker.WorkerPool
	logger      *zap.Logger
	sf          singleflight.Group
}

func NewVODUseCase(
	vodRepo domain.VODMediaRepository,
	drmRepo domain.DRMRepository,
	ffmpeg *ffmpeg.FFmpeg,
	storage storage.Storage,
	redisClient *redis.Client,
	workerPool *worker.WorkerPool,
	logger *zap.Logger,
) *VODUseCase {
	uc := &VODUseCase{
		vodRepo:     vodRepo,
		drmRepo:     drmRepo,
		ffmpeg:      ffmpeg,
		storage:     storage,
		redisClient: redisClient,
		workerPool:  workerPool,
		logger:      logger,
	}

	// Start HLS auto-delete cleanup job
	uc.StartHLSCleanupJob(context.Background())

	return uc
}

// UploadVideo handles video upload, creates VOD record, and submits to worker pool
func (uc *VODUseCase) UploadVideo(ctx context.Context, userID domain.UUID, title, description, visibility, tempFilePath string, originalFileName string) (*domain.VODMedia, error) {
	vodID := domain.NewUUID()

	// Upload original video to storage first so it is available
	originalFile, err := os.Open(tempFilePath)
	if err != nil {
		return nil, err
	}
	defer originalFile.Close()

	originalKey := filepath.Join("vods", vodID.String(), "original", originalFileName)
	originalURL, err := uc.storage.Upload(ctx, originalKey, originalFile)
	if err != nil {
		return nil, err
	}

	vod := &domain.VODMedia{
		ID:          vodID,
		UserID:      userID,
		Title:       title,
		Description: description,
		OriginalURL: originalURL,
		Status:      domain.VODStatusProcessing,
		Visibility:  visibility,
	}

	if err := uc.vodRepo.Create(ctx, vod); err != nil {
		// Clean up original upload on DB failure
		_ = uc.storage.Delete(ctx, originalKey)
		return nil, err
	}

	// Submit transcoding to worker pool instead of direct goroutine!
	payload := VODTranscodePayload{
		VODID:            vodID.String(),
		TempFilePath:     tempFilePath,
		OriginalFileName: originalFileName,
		UserID:           userID.String(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	job := &worker.Job{
		ID:        vodID.String(),
		Type:      worker.JobVideoTranscode,
		Payload:   json.RawMessage(payloadBytes),
		CreatedAt: time.Now(),
	}

	// Set initial progress in Redis to 5%
	progressKey := fmt.Sprintf("vod:job:%s:progress", vodID.String())
	uc.redisClient.GetClient().Set(ctx, progressKey, 5, 24*time.Hour)

	if err := uc.workerPool.Submit(job); err != nil {
		uc.logger.Error("Failed to submit VOD transcode job to worker pool, falling back to async goroutine", zap.Error(err))
		// Fallback to direct async processing if worker pool queue is full or closed
		go func() {
			_ = uc.ProcessVideo(context.Background(), vodID, tempFilePath, originalFileName)
		}()
	}

	return vod, nil
}

// ProcessVideo executes metadata extraction, 3-size thumbnails, and HLS transcoding with Redis progress tracking
func (uc *VODUseCase) ProcessVideo(ctx context.Context, vodID domain.UUID, tempFilePath string, originalFileName string) error {
	defer os.Remove(tempFilePath) // Cleanup local temp file

	progressKey := fmt.Sprintf("vod:job:%s:progress", vodID.String())

	vod, err := uc.vodRepo.GetByID(ctx, vodID)
	if err != nil {
		uc.logger.Error("Failed to find VOD in DB for processing", zap.String("vod_id", vodID.String()), zap.Error(err))
		return err
	}

	// Extract metadata (Update progress to 10%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 10, 24*time.Hour)
	meta, err := uc.ffmpeg.GetMetadata(ctx, tempFilePath)
	if err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	vod.Duration = int(meta.Duration)
	vod.FileSize = meta.Size

	// Generate base Thumbnail (Update progress to 15%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 15, 24*time.Hour)
	baseThumbTemp := tempFilePath + "_base.jpg"
	if err := uc.ffmpeg.GenerateThumbnail(ctx, tempFilePath, baseThumbTemp, meta.Duration); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}
	defer os.Remove(baseThumbTemp)

	// Generate 3 sizes of thumbnails: small, medium, large (Update progress to 20%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 20, 24*time.Hour)
	smallThumbTemp := tempFilePath + "_small.jpg"
	mediumThumbTemp := tempFilePath + "_medium.jpg"
	largeThumbTemp := tempFilePath + "_large.jpg"

	err = uc.ffmpeg.GenerateThreeThumbnails(ctx, baseThumbTemp, smallThumbTemp, mediumThumbTemp, largeThumbTemp)
	if err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}
	defer func() {
		os.Remove(smallThumbTemp)
		os.Remove(mediumThumbTemp)
		os.Remove(largeThumbTemp)
	}()

	// Upload all 3 thumbnails to storage
	smallFile, _ := os.Open(smallThumbTemp)
	smallKey := filepath.Join("vods", vod.ID.String(), "thumbnail_small.jpg")
	_, _ = uc.storage.Upload(ctx, smallKey, smallFile)
	smallFile.Close()

	mediumFile, _ := os.Open(mediumThumbTemp)
	mediumKey := filepath.Join("vods", vod.ID.String(), "thumbnail_medium.jpg")
	mediumURL, _ := uc.storage.Upload(ctx, mediumKey, mediumFile)
	mediumFile.Close()

	largeFile, _ := os.Open(largeThumbTemp)
	largeKey := filepath.Join("vods", vod.ID.String(), "thumbnail_large.jpg")
	_, _ = uc.storage.Upload(ctx, largeKey, largeFile)
	largeFile.Close()

	vod.ThumbnailURL = mediumURL

	// Generate HLS playlist and segments with progress updates (Update progress based on ffmpeg: 25% -> 85%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 25, 24*time.Hour)
	hlsOutDir := filepath.Join(os.TempDir(), "hls_"+vod.ID.String())
	os.MkdirAll(hlsOutDir, 0755)
	defer os.RemoveAll(hlsOutDir)

	// DRM AES-128 Setup
	drmKey := make([]byte, 16)
	if _, err := rand.Read(drmKey); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	// Simpan kunci DRM ke DB
	if err := uc.drmRepo.SaveDRMKey(ctx, vod.ID, drmKey); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	// Tulis file video.key dan video.keyinfo untuk FFmpeg HLS encryption
	keyFilePath := filepath.Join(hlsOutDir, "video.key")
	if err := os.WriteFile(keyFilePath, drmKey, 0644); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	keyInfoPath := filepath.Join(hlsOutDir, "video.keyinfo")
	// Format keyinfo:
	// line 1: URL kunci (dipanggil player)
	// line 2: Path lokal ke video.key
	keyURL := fmt.Sprintf("/api/v1/vods/%s/key", vod.ID.String())
	keyInfoContent := fmt.Sprintf("%s\n%s\n", keyURL, keyFilePath)
	if err := os.WriteFile(keyInfoPath, []byte(keyInfoContent), 0644); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	err = uc.ffmpeg.GenerateEncryptedHLS(ctx, tempFilePath, hlsOutDir, keyInfoPath, meta.Duration, func(hlsPercent float64) {
		// HLS progress accounts for 60% of total job progress (from 25% to 85%)
		totalPercent := 25 + int(hlsPercent*0.60)
		uc.redisClient.GetClient().Set(ctx, progressKey, totalPercent, 24*time.Hour)
	})
	if err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	// Upload HLS directory to storage (Update progress to 90%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 90, 24*time.Hour)
	files, err := os.ReadDir(hlsOutDir)
	if err != nil {
		uc.markAsFailed(ctx, vod, err)
		return err
	}

	var hlsURL string
	for _, f := range files {
		fPath := filepath.Join(hlsOutDir, f.Name())
		fFile, err := os.Open(fPath)
		if err != nil {
			continue
		}
		sKey := filepath.Join("vods", vod.ID.String(), "hls", f.Name())
		url, err := uc.storage.Upload(ctx, sKey, fFile)
		fFile.Close()

		if err == nil && strings.HasSuffix(f.Name(), ".m3u8") {
			hlsURL = url
		}
	}
	vod.HLSURL = hlsURL

	// Delete the original high-resolution uploaded video after HLS transcode success (Update progress to 95%)
	uc.redisClient.GetClient().Set(ctx, progressKey, 95, 24*time.Hour)
	originalKey := filepath.Join("vods", vod.ID.String(), "original", originalFileName)
	if err := uc.storage.Delete(ctx, originalKey); err != nil {
		uc.logger.Warn("Failed to delete original video upload from storage", zap.String("key", originalKey), zap.Error(err))
	} else {
		uc.logger.Info("Original video upload deleted successfully after transcode", zap.String("vod_id", vod.ID.String()))
		vod.OriginalURL = "" // Mark as deleted in domain URL but actual file is cleaned
	}

	// Save and complete (Update progress to 100%)
	vod.Status = domain.VODStatusReady
	if err := uc.vodRepo.Update(ctx, vod); err != nil {
		uc.logger.Error("Failed to update VOD to ready status", zap.Error(err))
		return err
	}

	uc.redisClient.GetClient().Set(ctx, progressKey, 100, 24*time.Hour)
	uc.logger.Info("VOD processing completed successfully", zap.String("vod_id", vod.ID.String()))
	return nil
}

func (uc *VODUseCase) markAsFailed(ctx context.Context, vod *domain.VODMedia, err error) {
	uc.logger.Error("VOD processing failed", zap.String("vod_id", vod.ID.String()), zap.Error(err))
	vod.Status = domain.VODStatusFailed
	uc.vodRepo.Update(ctx, vod)

	progressKey := fmt.Sprintf("vod:job:%s:progress", vod.ID.String())
	uc.redisClient.GetClient().Set(ctx, progressKey, 0, 24*time.Hour)
}

// StartHLSCleanupJob scans local uploads directory and deletes HLS directories older than 30 days
func (uc *VODUseCase) StartHLSCleanupJob(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				uc.logger.Info("Starting VOD HLS auto-delete lifecycle cleanup...")
				uploadsVodsDir := "./uploads/vods"
				entries, err := os.ReadDir(uploadsVodsDir)
				if err != nil {
					continue
				}

				now := time.Now()
				cutoff := now.AddDate(0, 0, -30) // 30 days ago

				for _, entry := range entries {
					if entry.IsDir() {
						hlsDir := filepath.Join(uploadsVodsDir, entry.Name(), "hls")
						info, err := os.Stat(hlsDir)
						if err == nil {
							if info.ModTime().Before(cutoff) {
								uc.logger.Info("HLS directory expired (30+ days old), auto-deleting", zap.String("path", hlsDir))
								_ = os.RemoveAll(filepath.Join(uploadsVodsDir, entry.Name()))
							}
						}
					}
				}
			}
		}
	}()
}

// GetVODDetail returns VOD details (Cached for 1 hour with Singleflight protection)
func (uc *VODUseCase) GetVODDetail(ctx context.Context, id domain.UUID) (*domain.VODMedia, error) {
	cacheKey := fmt.Sprintf("vod:detail:%s", id.String())

	// 1. Try cache first
	if uc.redisClient != nil {
		cached, err := uc.redisClient.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var vod domain.VODMedia
			if err := json.Unmarshal([]byte(cached), &vod); err == nil {
				return &vod, nil
			}
		}
	}

	// 2. Singleflight protection
	val, err, _ := uc.sf.Do(id.String(), func() (interface{}, error) {
		vod, err := uc.vodRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		// Save to cache (1 hour TTL)
		if uc.redisClient != nil {
			data, err := json.Marshal(vod)
			if err == nil {
				_ = uc.redisClient.Set(ctx, cacheKey, string(data), 1*time.Hour)
			}
		}

		return vod, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*domain.VODMedia), nil
}

func (uc *VODUseCase) ListUserVODs(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.VODMedia, error) {
	return uc.vodRepo.ListByUser(ctx, userID, limit, offset)
}

func (uc *VODUseCase) ListPublicVODs(ctx context.Context, limit, offset int) ([]*domain.VODMedia, error) {
	return uc.vodRepo.ListPublic(ctx, limit, offset)
}

func (uc *VODUseCase) UpdateVisibility(ctx context.Context, id domain.UUID, userID domain.UUID, visibility string) error {
	vod, err := uc.vodRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if vod.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "not your video", nil)
	}
	vod.Visibility = visibility
	if err := uc.vodRepo.Update(ctx, vod); err != nil {
		return err
	}
	if uc.redisClient != nil {
		_ = uc.redisClient.GetClient().Del(ctx, fmt.Sprintf("vod:detail:%s", id.String()))
	}
	return nil
}

func (uc *VODUseCase) DeleteVOD(ctx context.Context, id domain.UUID, userID domain.UUID) error {
	vod, err := uc.vodRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if vod.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "not your video", nil)
	}
	if err := uc.vodRepo.Delete(ctx, id); err != nil {
		return err
	}
	if uc.redisClient != nil {
		_ = uc.redisClient.GetClient().Del(ctx, fmt.Sprintf("vod:detail:%s", id.String()))
	}
	return nil
}
