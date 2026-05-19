package delivery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// DRMHandler menangani request HTTP untuk DRM dan pemutaran video terenkripsi
type DRMHandler struct {
	useCase    domain.DRMUseCaseInterface
	vodUseCase domain.VODUseCaseInterface
	logger     *zap.Logger
}

// NewDRMHandler membuat instance baru dari DRMHandler
func NewDRMHandler(useCase domain.DRMUseCaseInterface, vodUseCase domain.VODUseCaseInterface, logger *zap.Logger) *DRMHandler {
	return &DRMHandler{
		useCase:    useCase,
		vodUseCase: vodUseCase,
		logger:     logger,
	}
}

func (h *DRMHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *DRMHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// GenerateToken handles POST /vods/{vod_id}/token
func (h *DRMHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	vars := mux.Vars(r)
	vodIDStr := vars["vod_id"]
	vodID, err := domain.FromString(vodIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "ID VOD tidak valid")
		return
	}

	accessKey, tokenStr, err := h.useCase.GeneratePlaybackToken(r.Context(), userID, vodID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	playbackURL := fmt.Sprintf("/api/v1/vods/%s/playlist.m3u8?token=%s", vodID.String(), tokenStr)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"playback_token": tokenStr,
		"expires_at":     accessKey.ExpiresAt,
		"manifest_url":   playbackURL,
	})
}

// ServePlaylist handles GET /vods/{vod_id}/playlist.m3u8
func (h *DRMHandler) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vodIDStr := vars["vod_id"]
	vodID, err := domain.FromString(vodIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "ID VOD tidak valid")
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token pemutaran diperlukan")
		return
	}

	// Validasi token
	_, err = h.useCase.ValidateToken(r.Context(), tokenStr)
	if err != nil {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	// Baca file playlist.m3u8 asli dari direktori lokal
	// Sesuai konfigurasi: ./uploads/vods/{vod_id}/hls/playlist.m3u8
	playlistPath := filepath.Join(".", "uploads", "vods", vodID.String(), "hls", "playlist.m3u8")
	playlistBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		h.logger.Error("Gagal membaca playlist M3U8", zap.String("path", playlistPath), zap.Error(err))
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Playlist video tidak ditemukan")
		return
	}

	playlistContent := string(playlistBytes)

	// Modifikasi manifest secara dinamis:
	// 1. Tambahkan token keamanan pada URI kunci DRM
	// Contoh asli: URI="/api/v1/vods/abc/key"
	// Menjadi: URI="/api/v1/vods/abc/key?token=xyz"
	oldKeyURI := fmt.Sprintf(`URI="/api/v1/vods/%s/key"`, vodID.String())
	newKeyURI := fmt.Sprintf(`URI="/api/v1/vods/%s/key?token=%s"`, vodID.String(), tokenStr)
	playlistContent = strings.ReplaceAll(playlistContent, oldKeyURI, newKeyURI)

	// 2. Arahkan segment .ts ke endpoint terproteksi yang menyuntikkan watermark dinamis
	// Cari setiap baris segment, misalnya segment_000.ts
	// Ubah menjadi /api/v1/vods/{vod_id}/segments/segment_000.ts?token={playback_token}
	lines := strings.Split(playlistContent, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && strings.HasSuffix(trimmed, ".ts") {
			segmentName := trimmed
			lines[i] = fmt.Sprintf("/api/v1/vods/%s/segments/%s?token=%s", vodID.String(), segmentName, tokenStr)
		}
	}
	playlistContent = strings.Join(lines, "\n")

	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(playlistContent))
}

// ServeKey handles GET /vods/{vod_id}/key
func (h *DRMHandler) ServeKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vodIDStr := vars["vod_id"]
	vodID, err := domain.FromString(vodIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "ID VOD tidak valid")
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token pemutaran diperlukan")
		return
	}

	// Validasi token
	_, err = h.useCase.ValidateToken(r.Context(), tokenStr)
	if err != nil {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	// Ambil kunci DRM 16-byte dari DB
	keyBytes, err := h.useCase.GetVODDRMKey(r.Context(), vodID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "KEY_NOT_FOUND", "Kunci DRM untuk video ini tidak ditemukan")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(keyBytes)
}

// ServeSegment handles GET /vods/{vod_id}/segments/{segment}
func (h *DRMHandler) ServeSegment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vodIDStr := vars["vod_id"]
	segmentName := vars["segment"]

	vodID, err := domain.FromString(vodIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "ID VOD tidak valid")
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token pemutaran diperlukan")
		return
	}

	// Validasi token dan ambil ID User penonton
	accessKey, err := h.useCase.ValidateToken(r.Context(), tokenStr)
	if err != nil {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	// Path segment asli di lokal
	segmentPath := filepath.Join(".", "uploads", "vods", vodID.String(), "hls", segmentName)
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Segmen video tidak ditemukan")
		return
	}

	// Terapkan Dynamic Watermarking ID User via FFmpeg secara on-the-fly!
	watermarkedPath, err := h.useCase.WatermarkSegment(r.Context(), segmentPath, accessKey.UserID)
	if err != nil {
		h.logger.Error("Gagal menyematkan watermark dinamis, menyajikan segmen mentah sebagai fallback aman", zap.Error(err))
		// Fallback menyajikan segment asli jika FFmpeg bermasalah agar pemutaran tidak macet
		h.serveLocalFile(w, segmentPath)
		return
	}

	h.serveLocalFile(w, watermarkedPath)
}

func (h *DRMHandler) serveLocalFile(w http.ResponseWriter, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "FILE_ERROR", "Gagal menyajikan segmen")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "video/MP2T")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}
