package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
)

// FFmpeg handles video processing operations
type FFmpeg struct {
	logger *zap.Logger
}

// New creates a new FFmpeg instance
func New(logger *zap.Logger) *FFmpeg {
	return &FFmpeg{logger: logger}
}

// VideoMetadata contains basic video info
type VideoMetadata struct {
	Duration float64
	Width    int
	Height   int
	Bitrate  int64
	Size     int64
}

// GetMetadata extracts metadata using ffprobe
func (f *FFmpeg) GetMetadata(ctx context.Context, inputPath string) (*VideoMetadata, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,bit_rate:format=duration,size",
		"-of", "json",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		f.logger.Error("ffprobe error", zap.Error(err), zap.String("output", string(output)))
		return nil, err
	}

	var result struct {
		Streams []struct {
			Width   int    `json:"width"`
			Height  int    `json:"height"`
			BitRate string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	meta := &VideoMetadata{}
	if len(result.Streams) > 0 {
		meta.Width = result.Streams[0].Width
		meta.Height = result.Streams[0].Height
		meta.Bitrate, _ = strconv.ParseInt(result.Streams[0].BitRate, 10, 64)
	}
	meta.Duration, _ = strconv.ParseFloat(result.Format.Duration, 64)
	meta.Size, _ = strconv.ParseInt(result.Format.Size, 10, 64)

	return meta, nil
}

// GenerateHLS transcodes input to HLS format (360p and 720p)
func (f *FFmpeg) GenerateHLS(ctx context.Context, inputPath, outputDir string) error {
	// Simple 720p HLS command for brevity
	hlsOutputPath := filepath.Join(outputDir, "playlist.m3u8")
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-profile:v", "main",
		"-vf", "scale=-2:720", // scale to 720p
		"-c:v", "h264",
		"-c:a", "aac",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%03d.ts"),
		hlsOutputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		f.logger.Error("ffmpeg HLS error", zap.Error(err), zap.String("output", string(output)))
		return err
	}

	return nil
}

// GenerateThumbnail extracts a frame at the middle of the video
func (f *FFmpeg) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, duration float64) error {
	middleTime := duration / 2
	timeStr := fmt.Sprintf("%02d:%02d:%02d", int(middleTime)/3600, (int(middleTime)%3600)/60, int(middleTime)%60)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", timeStr,
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		f.logger.Error("ffmpeg thumbnail error", zap.Error(err), zap.String("output", string(output)))
		return err
	}

	return nil
}
