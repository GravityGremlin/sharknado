// Package player provides audio streaming via FFmpeg transcoding.
// Transcodes source audio to opus/MP3/FLAC and streams to the client.
package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sharknado/backend/internal/models"
)

// StreamConfig for the streaming pipeline.
type StreamConfig struct {
	CacheDir  string
	LibraryDir string
	FFmpegPath string
}

// Streamer handles audio transcoding and streaming.
type Streamer struct {
	cacheDir   string
	libraryDir string
	ffmpegPath string
	cache      *streamCache
}

// NewStreamer creates a new audio streamer.
func NewStreamer(cfg StreamConfig) *Streamer {
	ffmpeg := cfg.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	return &Streamer{
		cacheDir:   cfg.CacheDir,
		libraryDir: cfg.LibraryDir,
		ffmpegPath: ffmpeg,
		cache:      newStreamCache(24 * time.Hour),
	}
}

// Stream transcodes audio from sourcePath and writes to writer.
// Returns the content type and any error.
func (s *Streamer) Stream(ctx context.Context, sourcePath string, format models.StreamFormat, w io.Writer) (contentType string, err error) {
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source file not found: %w", err)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", sourcePath, string(format))
	if cached, ok := s.cache.get(cacheKey); ok {
		contentType = formatContentType(format)
		_, err = io.Copy(w, bytes.NewReader(cached))
		return contentType, err
	}

	// Build FFmpeg command
	contentType = formatContentType(format)
	args := s.buildFFmpegArgs(sourcePath, format)

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	// Stream output to writer
	_, err = io.Copy(w, stdout)
	if err != nil {
		cmd.Process.Kill()
		return "", fmt.Errorf("stream output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("ffmpeg finished: %w", err)
	}

	return contentType, nil
}

// StreamToFile transcodes and saves to a cache file. Returns the file path.
func (s *Streamer) StreamToFile(ctx context.Context, sourcePath string, format models.StreamFormat) (string, error) {
	cacheKey := fmt.Sprintf("%s_%s", filepath.Base(sourcePath), string(format))
	cachePath := filepath.Join(s.cacheDir, cacheKey)

	// Check if already cached
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	// Ensure cache dir exists
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	args := s.buildFFmpegArgs(sourcePath, format)
	// Add output file
	args = append(args, cachePath)

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg transcode failed: %w (output: %s)", err, string(output))
	}

	return cachePath, nil
}

// FindSourceFile looks for a source file for a given track ID.
func (s *Streamer) FindSourceFile(provider, providerID string) (string, error) {
	// Search in downloads and library directories
	dirs := []string{s.libraryDir, s.cacheDir}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		var found string
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".flac", ".mp3", ".ogg", ".opus", ".m4a", ".aac", ".wav":
				// Check if filename contains the provider ID
				if strings.Contains(filepath.Base(path), providerID) {
					found = path
				}
			}
			return nil
		})

		if err == nil && found != "" {
			return found, nil
		}
	}

	return "", fmt.Errorf("source file not found for %s:%s", provider, providerID)
}

// buildFFmpegArgs builds the FFmpeg command arguments for transcoding.
func (s *Streamer) buildFFmpegArgs(sourcePath string, format models.StreamFormat) []string {
	base := []string{
		"-i", sourcePath,
		"-f", string(format),
	}

	switch format {
	case models.StreamFormatOpus:
		return append(base,
			"-c:a", "libopus",
			"-b:a", "192k",
			"-vbr", "on",
			"-application", "audio",
			"-ar", "48000",
			"-ac", "2",
			"pipe:1",
		)
	case models.StreamFormatMP3:
		return append(base,
			"-c:a", "libmp3lame",
			"-b:a", "320k",
			"-q:a", "0",
			"-ar", "44100",
			"-ac", "2",
			"pipe:1",
		)
	case models.StreamFormatFlac:
		return append(base,
			"-c:a", "flac",
			"pipe:1",
		)
	default:
		// Default to opus
		return append(base,
			"-c:a", "libopus",
			"-b:a", "192k",
			"pipe:1",
		)
	}
}

func formatContentType(format models.StreamFormat) string {
	switch format {
	case models.StreamFormatOpus:
		return "audio/ogg"
	case models.StreamFormatMP3:
		return "audio/mpeg"
	case models.StreamFormatFlac:
		return "audio/flac"
	default:
		return "audio/ogg"
	}
}

// streamCache caches transcoded audio data.
type streamCache struct {
	mu      sync.RWMutex
	entries map[string][]byte
	ttl     time.Duration
}

func newStreamCache(ttl time.Duration) *streamCache {
	return &streamCache{
		entries: make(map[string][]byte),
		ttl:     ttl,
	}
}

func (c *streamCache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	// Simple TTL not implemented for in-memory cache (use file-based cache instead)
	return data, true
}

func (c *streamCache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if too many entries
	if len(c.entries) > 100 {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	c.entries[key] = data
}

// FindTrackFile searches for any audio file matching the given query in a directory.
func FindTrackFile(dir, query string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("no directory specified")
	}

	query = strings.ToLower(query)
	var best string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".flac", ".mp3", ".ogg", ".opus", ".m4a", ".aac", ".wav":
			name := strings.ToLower(filepath.Base(path))
			if strings.Contains(name, query) {
				best = path
			}
		}
		return nil
	})

	if best == "" {
		return "", fmt.Errorf("no matching file found for %q in %s", query, dir)
	}
	return best, err
}
