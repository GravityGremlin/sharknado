// Package download manages download jobs for music from all providers.
// Calls streamrip (rip url) and tidal-dl-ng (dl) as subprocesses.
package download

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sharknado/backend/internal/events"
	"github.com/sharknado/backend/internal/library"
	"github.com/sharknado/backend/internal/models"
)

// Manager tracks and executes download jobs.
type Manager struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	dlDir   string
	broker  *events.EventBroker
	db      *library.DB
	workers chan struct{} // semaphore for concurrency
}

// Job represents an active or completed download.
type Job struct {
	models.DownloadJob
	CancelFunc  context.CancelFunc
	atomProgress atomic.Int64 // 0-10000 (fixed-point for atomic float)
}

// getProgress returns the atomic progress as a float64.
func (j *Job) getProgress() float64 {
	return float64(j.atomProgress.Load()) / 100.0
}

// Config for the download manager.
type Config struct {
	DownloadDir   string
	MaxConcurrent int
	Broker        *events.EventBroker
	DB            *library.DB
}

// NewManager creates a download manager.
func NewManager(cfg Config) *Manager {
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	return &Manager{
		jobs:    make(map[string]*Job),
		dlDir:   cfg.DownloadDir,
		broker:  cfg.Broker,
		db:      cfg.DB,
		workers: make(chan struct{}, maxConcurrent),
	}
}

// Submit creates a new download job and starts it.
func (m *Manager) Submit(url, service, quality string) *models.DownloadJob {
	// Auto-detect provider from URL if not specified
	if service == "" {
		service = DetectProvider(url)
	}

	job := &Job{
		DownloadJob: models.DownloadJob{
			ID:        generateID(),
			URL:       url,
			Service:   service,
			Status:    "queued",
			Quality:   quality,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Progress:  0,
		},
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	m.broadcast("job.created", job.DownloadJob)
	go m.execute(job)

	return &job.DownloadJob
}

// Get returns a job by ID.
func (m *Manager) Get(id string) *models.DownloadJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil
	}
	clone := job.DownloadJob
	return &clone
}

// List returns all jobs.
func (m *Manager) List() []*models.DownloadJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobs := make([]*models.DownloadJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		clone := j.DownloadJob
		jobs = append(jobs, &clone)
	}
	return jobs
}

// Pause pauses a running job.
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	if job.Status != "running" {
		return fmt.Errorf("job not running")
	}
	job.Status = "paused"
	if job.CancelFunc != nil {
		job.CancelFunc()
		job.CancelFunc = nil
	}
	job.Progress = job.getProgress()
	m.broadcast("job.updated", job.DownloadJob)
	m.persistJob(job)
	return nil
}

// Cancel cancels a job.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	job.Status = "cancelled"
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if job.CancelFunc != nil {
		job.CancelFunc()
		job.CancelFunc = nil
	}
	job.Progress = job.getProgress()
	m.broadcast("job.updated", job.DownloadJob)
	m.persistJob(job)
	return nil
}

// Delete removes a job from the list.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	if job.CancelFunc != nil {
		job.CancelFunc()
		job.CancelFunc = nil
	}
	delete(m.jobs, id)
	return nil
}

// Resume resumes a paused or failed job.
func (m *Manager) Resume(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	if job.Status != "paused" && job.Status != "failed" {
		return fmt.Errorf("job is not paused or failed")
	}
	job.Status = "queued"
	job.ErrorMsg = ""
	job.atomProgress.Store(0)
	m.broadcast("job.updated", job.DownloadJob)
	m.persistJob(job)
	go m.execute(job)
	return nil
}

// execute runs the actual download subprocess.
func (m *Manager) execute(job *Job) {
	// Early check if someone canceled before token was granted
	m.mu.RLock()
	if job.Status == "cancelled" || job.Status == "paused" {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	ctx, cancel := context.WithCancel(context.Background())

	// Watch context for cancellation inside workers semaphore acquire
	select {
	case m.workers <- struct{}{}:
		// acquired
	case <-ctx.Done():
		cancel()
		return
	}
	defer func() { <-m.workers }()

	m.mu.Lock()
	// Fix 1.1: Re-check status after acquiring lock to prevent TOCTOU race
	if job.Status == "cancelled" || job.Status == "paused" {
		m.mu.Unlock()
		cancel()
		return
	}
	job.CancelFunc = cancel
	job.Status = "running"
	job.StartedAt = time.Now().UTC().Format(time.RFC3339)
	m.mu.Unlock()
	m.broadcast("job.updated", job.DownloadJob)

	var cmd *exec.Cmd

	switch job.Service {
	case "tidal":
		// Fix 1.3: Use -- to prevent argument injection
		cmd = exec.CommandContext(ctx, "tidal-dl-ng", "dl", "--", job.URL)
	case "qobuz", "deezer":
		// Fix 1.3: Use -- to prevent argument injection
		cmd = exec.CommandContext(ctx, "rip", "url", "--", job.URL)
	default:
		// Try streamrip as fallback
		cmd = exec.CommandContext(ctx, "rip", "url", "--", job.URL)
	}

	cmd.Dir = m.dlDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"STREAMRIP_CONFIG_DIR=/app/data/streamrip-config",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.mu.Lock()
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("failed to create pipe: %v", err)
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		m.mu.Unlock()
		m.broadcast("job.updated", job.DownloadJob)
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		m.mu.Lock()
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("failed to start: %v", err)
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		m.mu.Unlock()
		m.broadcast("job.updated", job.DownloadJob)
		return
	}

	// Parse progress from output
	go parseProgress(stdout, job, m)

	err = cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		if job.Status != "paused" && job.Status != "cancelled" {
			job.Status = "cancelled"
		}
	} else if err != nil {
		job.Status = "failed"
		job.ErrorMsg = err.Error()
	} else {
		job.Status = "completed"
		job.atomProgress.Store(10000)
		// Auto-import downloaded files
		go m.importDownloads()
	}
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	job.Progress = job.getProgress()
	m.broadcast("job.updated", job.DownloadJob)
	m.persistJob(job)
}

// DetectProvider guesses the provider from a URL.
func DetectProvider(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "tidal.com") || strings.Contains(lower, "listen.tidal.com"):
		return "tidal"
	case strings.Contains(lower, "qobuz.com") || strings.Contains(lower, "play.qobuz.com"):
		return "qobuz"
	case strings.Contains(lower, "deezer.com") || strings.Contains(lower, "deezer.page.link"):
		return "deezer"
	}
	return "unknown"
}

// progressRegex matches download progress indicators from CLI tools.
// Matches: "10%", "[####....] 45%", "Track 1/5 - 20%", etc.
var progressRegex = regexp.MustCompile(`(\d+)%|(\d+)/(\d+)`)
var albumRegex = regexp.MustCompile(`(?i)album|track|downloading|converting|finish|complete|downloaded`)

func parseProgress(r interface{ Read([]byte) (int, error) }, job *Job, m *Manager) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}

		// Try to extract percentage (X% format)
		if matches := progressRegex.FindStringSubmatch(text); len(matches) > 1 {
			// First capture group is percentage
			if matches[1] != "" {
				if pct, e := strconv.Atoi(matches[1]); e == nil {
					job.atomProgress.Store(int64(pct * 100))
					progressUpdate := job.DownloadJob
					progressUpdate.Progress = job.getProgress()
					m.broadcast("job.updated", progressUpdate)
				}
			}
			// Second capture group is X/Y format (current/total)
			if matches[2] != "" && matches[3] != "" {
				if current, e1 := strconv.Atoi(matches[2]); e1 == nil {
					if total, e2 := strconv.Atoi(matches[3]); e2 == nil && total > 0 {
						pct := (current * 100) / total
						job.atomProgress.Store(int64(pct * 100))
						progressUpdate := job.DownloadJob
						progressUpdate.Progress = job.getProgress()
						m.broadcast("job.updated", progressUpdate)
					}
				}
			}
		}

		// Log progress line for debugging
		if albumRegex.MatchString(text) {
			m.broadcast("job.log", map[string]string{
				"job_id": job.ID,
				"text":   text,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("parseProgress scanner error: %v", err)
	}
}

func (m *Manager) broadcast(event string, data any) {
	if m.broker != nil {
		m.broker.Broadcast(event, data)
	}
}

// persistJob writes the job state to the database.
func (m *Manager) persistJob(job *Job) {
	if m.db == nil {
		return
	}
	_, err := m.db.Exec(`INSERT INTO download_jobs
		(id, url, service, status, quality, progress, error_msg, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			progress = excluded.progress,
			error_msg = excluded.error_msg,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at`,
		job.ID, job.URL, job.Service, job.Status, job.Quality,
		job.getProgress(), job.ErrorMsg, job.CreatedAt, job.StartedAt, job.FinishedAt)
	if err != nil {
		log.Printf("persist job %s: %v", job.ID, err)
	}
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("dl-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// ScanDir finds downloaded audio files in a directory and extracts metadata via ffprobe.
func ScanDir(dir string) ([]models.Track, error) {
	var tracks []models.Track
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("ScanDir walk error at %s: %v", path, err)
			return nil // Continue walking
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".flac", ".mp3", ".ogg", ".opus", ".m4a", ".aac", ".wav":
			t := models.Track{
				ID:         fmt.Sprintf("local:%s", path),
				Provider:   "local",
				ProviderID: path,
				FilePath:   path,
				FileFormat: strings.TrimPrefix(ext, "."),
				FileSize:   info.Size(),
				Downloaded: true,
			}
			probeMetadata(&t, path)
			tracks = append(tracks, t)
		}
		return nil
	})
	return tracks, err
}

// probeMetadata uses ffprobe to extract audio metadata from a file.
func probeMetadata(t *models.Track, path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "quiet",
		"-print_format", "json", "-show_format", path)
	out, err := cmd.Output()
	if err != nil {
		// Fall back to filename parsing
		parseFilename(t, path)
		return
	}

	var probe struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		parseFilename(t, path)
		return
	}

	tags := probe.Format.Tags
	t.Title = tagValue(tags, "title", "TITLE")
	t.Artist = tagValue(tags, "artist", "ARTIST")
	t.Album = tagValue(tags, "album", "ALBUM")
	t.AlbumArtist = tagValue(tags, "album_artist", "ALBUMARTIST", "ALBUM_ARTIST")
	t.Genre = tagValue(tags, "genre", "GENRE")

	if v := tagValue(tags, "track", "TRACKNUMBER"); v != "" {
		// Handle "3/12" format
		parts := strings.SplitN(v, "/", 2)
		if n, e := strconv.Atoi(strings.TrimSpace(parts[0])); e == nil {
			t.TrackNumber = n
		}
	}
	if v := tagValue(tags, "disc", "DISCNUMBER"); v != "" {
		parts := strings.SplitN(v, "/", 2)
		if n, e := strconv.Atoi(strings.TrimSpace(parts[0])); e == nil {
			t.DiscNumber = n
		}
	}
	if v := tagValue(tags, "date", "DATE", "year", "YEAR"); v != "" {
		// Take first 4 chars as year
		if len(v) >= 4 {
			v = v[:4]
		}
		if n, e := strconv.Atoi(v); e == nil {
			t.Year = n
		}
	}
	if probe.Format.Duration != "" {
		if d, e := strconv.ParseFloat(probe.Format.Duration, 64); e == nil {
			t.Duration = d
		}
	}

	// If title is still empty, fall back to filename
	if t.Title == "" {
		parseFilename(t, path)
	}
}

// tagValue returns the first non-empty value for any of the given tag keys.
func tagValue(tags map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := tags[k]; v != "" {
			return v
		}
	}
	return ""
}

// parseFilename tries to extract track info from a filename like "01. Artist - Title.flac"
func parseFilename(t *models.Track, path string) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	// Try "NN. Artist - Title" pattern
	re := regexp.MustCompile(`^(\d+)\.\s*(.+?)\s*-\s*(.+)$`)
	if m := re.FindStringSubmatch(base); len(m) == 4 {
		if n, e := strconv.Atoi(m[1]); e == nil {
			t.TrackNumber = n
		}
		if t.Artist == "" {
			t.Artist = m[2]
		}
		if t.Title == "" {
			t.Title = m[3]
		}
		return
	}

	// Try "Artist - Title" pattern
	if parts := strings.SplitN(base, " - ", 2); len(parts) == 2 {
		if t.Artist == "" {
			t.Artist = parts[0]
		}
		if t.Title == "" {
			t.Title = parts[1]
		}
		return
	}

	// Use filename as title
	if t.Title == "" {
		t.Title = base
	}

	// Try to get album from parent directory name
	if t.Album == "" {
		dir := filepath.Base(filepath.Dir(path))
		if dir != "." && dir != "/" {
			t.Album = dir
		}
	}
}

// importDownloads scans the download directory and imports tracks into the DB.
func (m *Manager) importDownloads() {
	tracks, err := ScanDir(m.dlDir)
	if err != nil {
		log.Printf("import scan error: %v", err)
		// continue with partial tracks
	}
	if m.db == nil {
		return
	}
	imported := 0
	for i := range tracks {
		if err := m.db.InsertTrack(&tracks[i]); err != nil {
			log.Printf("import track %s: %v", tracks[i].FilePath, err)
		} else {
			imported++
		}
	}
	if imported > 0 {
		log.Printf("imported %d tracks from downloads", imported)
		// Request a cache clear to update search results with "downloaded: true"
		m.broadcast("library.updated", nil)
	}
}
