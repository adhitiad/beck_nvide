package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type drmUseCase struct {
	drmRepo domain.DRMRepository
	vodRepo domain.VODMediaRepository
	logger  *zap.Logger
}

// NewDRMUseCase membuat instance baru dari DRMUseCase
func NewDRMUseCase(
	drmRepo domain.DRMRepository,
	vodRepo domain.VODMediaRepository,
	logger *zap.Logger,
) domain.DRMUseCaseInterface {
	return &drmUseCase{
		drmRepo: drmRepo,
		vodRepo: vodRepo,
		logger:  logger,
	}
}

func (uc *drmUseCase) GeneratePlaybackToken(ctx context.Context, userID, vodID domain.UUID) (*domain.VODAccessKey, string, error) {
	// Verifikasi apakah VOD ada
	_, err := uc.vodRepo.GetByID(ctx, vodID)
	if err != nil {
		return nil, "", err
	}

	// Generate secure token (high-entropy key)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}
	tokenStr := fmt.Sprintf("%x", tokenBytes)

	expiresAt := time.Now().Add(15 * time.Minute) // Berlaku selama 15 menit

	accessKey := &domain.VODAccessKey{
		ID:          domain.NewUUID(),
		VODID:       vodID,
		UserID:      userID,
		AccessToken: tokenStr,
		ExpiresAt:   expiresAt,
	}

	if err := uc.drmRepo.SaveAccessKey(ctx, accessKey); err != nil {
		uc.logger.Error("Gagal menyimpan access token VOD ke DB", zap.Error(err))
		return nil, "", err
	}

	return accessKey, tokenStr, nil
}

func (uc *drmUseCase) ValidateToken(ctx context.Context, token string) (*domain.VODAccessKey, error) {
	accessKey, err := uc.drmRepo.GetAccessKey(ctx, token)
	if err != nil {
		return nil, err
	}

	if time.Now().After(accessKey.ExpiresAt) {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "token akses pemutaran video telah kadaluwarsa", nil)
	}

	return accessKey, nil
}

func (uc *drmUseCase) GetVODDRMKey(ctx context.Context, vodID domain.UUID) ([]byte, error) {
	return uc.drmRepo.GetDRMKey(ctx, vodID)
}

func (uc *drmUseCase) GenerateDRMKeysForVOD(ctx context.Context, vodID domain.UUID) ([]byte, error) {
	// Buat kunci enkripsi 16-byte acak (AES-128)
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	if err := uc.drmRepo.SaveDRMKey(ctx, vodID, key); err != nil {
		uc.logger.Error("Gagal menyimpan kunci DRM VOD ke DB", zap.Error(err))
		return nil, err
	}

	return key, nil
}

// WatermarkSegment memproses segment HLS secara on-the-fly membakar watermark berisi ID user ke dalam video
func (uc *drmUseCase) WatermarkSegment(ctx context.Context, segmentPath string, userID domain.UUID) (string, error) {
	// Pastikan file segmen input ada
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		return "", fmt.Errorf("segmen video tidak ditemukan: %s", segmentPath)
	}

	// Tentukan direktori temp khusus watermark
	tempDir := filepath.Join(os.TempDir(), "nvide_watermarks")
	_ = os.MkdirAll(tempDir, 0755)

	outPath := filepath.Join(tempDir, fmt.Sprintf("wm_%s_%s", userID.String()[:8], filepath.Base(segmentPath)))

	// Jalankan perintah FFmpeg untuk membakar teks watermark
	// Filter drawtext menempatkan teks buram di sudut kanan bawah: x=w-220, y=h-40
	watermarkText := fmt.Sprintf("USER ID\\: %s", userID.String())
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", segmentPath,
		"-vf", fmt.Sprintf("drawtext=text='%s':x=w-300:y=h-40:fontsize=14:fontcolor=white@0.4", watermarkText),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-c:a", "copy",
		"-f", "mpegts",
		outPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		uc.logger.Error("Gagal menambahkan watermark dinamis via FFmpeg",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.String("cmd", cmd.String()),
		)
		return "", err
	}

	// Jalankan goroutine pembersihan otomatis setelah 30 detik untuk menghemat ruang disk temp
	go func(filePath string) {
		time.Sleep(30 * time.Second)
		_ = os.Remove(filePath)
	}(outPath)

	return outPath, nil
}
