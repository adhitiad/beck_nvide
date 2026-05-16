package usecase

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/ffmpeg"
	"nvide-live/pkg/storage"
)

type VODUseCase struct {
	vodRepo domain.VODMediaRepository
	ffmpeg  *ffmpeg.FFmpeg
	storage storage.Storage
	logger  *zap.Logger
}

func NewVODUseCase(
	vodRepo domain.VODMediaRepository,
	ffmpeg *ffmpeg.FFmpeg,
	storage storage.Storage,
	logger *zap.Logger,
) domain.VODUseCaseInterface {
	return &VODUseCase{
		vodRepo: vodRepo,
		ffmpeg:  ffmpeg,
		storage: storage,
		logger:  logger,
	}
}

// UploadVideo handles video upload, metadata extraction, and transcoding
func (uc *VODUseCase) UploadVideo(ctx context.Context, userID domain.UUID, title, description, visibility, tempFilePath string, originalFileName string) (*domain.VODMedia, error) {
	vodID := domain.NewUUID()

	// Upload original video
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
		return nil, err
	}

	// Start async processing
	go uc.processVideo(vod, tempFilePath)

	return vod, nil
}

func (uc *VODUseCase) processVideo(vod *domain.VODMedia, tempFilePath string) {
	ctx := context.Background()
	defer os.Remove(tempFilePath) // Cleanup temp file

	// Extract metadata
	meta, err := uc.ffmpeg.GetMetadata(ctx, tempFilePath)
	if err != nil {
		uc.markAsFailed(ctx, vod, err)
		return
	}

	vod.Duration = int(meta.Duration)
	vod.FileSize = meta.Size

	// Generate Thumbnail
	thumbTemp := tempFilePath + ".jpg"
	if err := uc.ffmpeg.GenerateThumbnail(ctx, tempFilePath, thumbTemp, meta.Duration); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return
	}
	defer os.Remove(thumbTemp)

	thumbFile, err := os.Open(thumbTemp)
	if err == nil {
		thumbKey := filepath.Join("vods", vod.ID.String(), "thumbnail.jpg")
		if thumbURL, err := uc.storage.Upload(ctx, thumbKey, thumbFile); err == nil {
			vod.ThumbnailURL = thumbURL
		}
		thumbFile.Close()
	}

	// Generate HLS
	hlsOutDir := filepath.Join(os.TempDir(), "hls_"+vod.ID.String())
	os.MkdirAll(hlsOutDir, 0755)
	defer os.RemoveAll(hlsOutDir)

	if err := uc.ffmpeg.GenerateHLS(ctx, tempFilePath, hlsOutDir); err != nil {
		uc.markAsFailed(ctx, vod, err)
		return
	}

	// Upload HLS directory to storage
	// For local storage, we can just copy. For S3, we'd iterate over files.
	// Since storage interface takes an io.Reader per key, let's iterate.
	files, _ := os.ReadDir(hlsOutDir)
	var hlsURL string
	for _, f := range files {
		fPath := filepath.Join(hlsOutDir, f.Name())
		fFile, _ := os.Open(fPath)
		sKey := filepath.Join("vods", vod.ID.String(), "hls", f.Name())
		url, _ := uc.storage.Upload(ctx, sKey, fFile)
		fFile.Close()

		if strings.HasSuffix(f.Name(), ".m3u8") {
			hlsURL = url
		}
	}

	vod.HLSURL = hlsURL
	vod.Status = domain.VODStatusReady

	// Update DB
	if err := uc.vodRepo.Update(ctx, vod); err != nil {
		uc.logger.Error("Failed to update VOD to ready status", zap.Error(err))
	} else {
		uc.logger.Info("VOD processing completed", zap.String("vod_id", vod.ID.String()))
	}
}

func (uc *VODUseCase) markAsFailed(ctx context.Context, vod *domain.VODMedia, err error) {
	uc.logger.Error("VOD processing failed", zap.String("vod_id", vod.ID.String()), zap.Error(err))
	vod.Status = domain.VODStatusFailed
	uc.vodRepo.Update(ctx, vod)
}

func (uc *VODUseCase) GetVODDetail(ctx context.Context, id domain.UUID) (*domain.VODMedia, error) {
	return uc.vodRepo.GetByID(ctx, id)
}

func (uc *VODUseCase) ListUserVODs(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.VODMedia, error) {
	return uc.vodRepo.ListByUser(ctx, userID, limit, offset)
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
	return uc.vodRepo.Update(ctx, vod)
}

func (uc *VODUseCase) DeleteVOD(ctx context.Context, id domain.UUID, userID domain.UUID) error {
	vod, err := uc.vodRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if vod.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "not your video", nil)
	}
	return uc.vodRepo.Delete(ctx, id)
}
