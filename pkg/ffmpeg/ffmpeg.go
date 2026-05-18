package ffmpeg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

// VideoMetadata contains basic video info including codecs
type VideoMetadata struct {
	Duration   float64
	Width      int
	Height     int
	Bitrate    int64
	Size       int64
	VideoCodec string
	AudioCodec string
}

// GetMetadata extracts metadata using ffprobe
func (f *FFmpeg) GetMetadata(ctx context.Context, inputPath string) (*VideoMetadata, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,bit_rate,codec_name:format=duration,size",
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
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			BitRate   string `json:"bit_rate"`
			CodecName string `json:"codec_name"`
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
		meta.VideoCodec = result.Streams[0].CodecName
	}

	// Fetch audio codec info separately
	audioCmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "json",
		inputPath,
	)
	if audioOutput, audioErr := audioCmd.Output(); audioErr == nil {
		var audioResult struct {
			Streams []struct {
				CodecName string `json:"codec_name"`
			} `json:"streams"`
		}
		if json.Unmarshal(audioOutput, &audioResult) == nil && len(audioResult.Streams) > 0 {
			meta.AudioCodec = audioResult.Streams[0].CodecName
		}
	}

	meta.Duration, _ = strconv.ParseFloat(result.Format.Duration, 64)
	meta.Size, _ = strconv.ParseInt(result.Format.Size, 10, 64)

	return meta, nil
}

// GenerateHLS transcodes input to HLS format (720p) with progress tracking
func (f *FFmpeg) GenerateHLS(ctx context.Context, inputPath, outputDir string, duration float64, progressCallback func(percent float64)) error {
	hlsOutputPath := filepath.Join(outputDir, "playlist.m3u8")
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-progress", "pipe:1",
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

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress from pipe
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			timeUsStr := strings.TrimPrefix(line, "out_time_us=")
			if timeUs, err := strconv.ParseFloat(timeUsStr, 64); err == nil && duration > 0 {
				percent := (timeUs / (duration * 1000000.0)) * 100.0
				if percent > 100.0 {
					percent = 100.0
				}
				if percent < 0.0 {
					percent = 0.0
				}
				if progressCallback != nil {
					progressCallback(percent)
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		f.logger.Error("ffmpeg HLS transcoding failed", zap.Error(err))
		return err
	}

	return nil
}

// GenerateThumbnail extracts a frame at the middle of the video
func (f *FFmpeg) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, duration float64) error {
	middleTime := duration / 2
	timeStr := fmt.Sprintf("%02d:%02d:%02d", int(middleTime)/3600, (int(middleTime)%3600)/60, int(middleTime)%60)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
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

// GenerateThreeThumbnails resizes a source thumbnail into small, medium, and large sizes
func (f *FFmpeg) GenerateThreeThumbnails(ctx context.Context, sourcePath, smallPath, mediumPath, largePath string) error {
	// Small: scale=320:-1
	cmdSmall := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", sourcePath, "-vf", "scale=320:-1", smallPath)
	if out, err := cmdSmall.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate small thumbnail: %s, %w", string(out), err)
	}

	// Medium: scale=640:-1
	cmdMedium := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", sourcePath, "-vf", "scale=640:-1", mediumPath)
	if out, err := cmdMedium.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate medium thumbnail: %s, %w", string(out), err)
	}

	// Large: scale=1280:-1
	cmdLarge := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", sourcePath, "-vf", "scale=1280:-1", largePath)
	if out, err := cmdLarge.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate large thumbnail: %s, %w", string(out), err)
	}

	return nil
}
