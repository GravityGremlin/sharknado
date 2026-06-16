// Package download manages download jobs for music from all providers.
// Calls streamrip (rip url) and tidal-dl-ng (dl) as subprocesses.
package download

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sharknado/backend/internal/events"
	"github.com/sharknado/backend/internal/models"
)

// Manager tracks and executes download jobs.
type Manager struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	dlDir    string
	broker   *events.EventBroker
	workers  chan struct{} // semaphore for concurrency
}

// Job represents an active or completed download.
type Job struct {
	models.DownloadJob
	CancelFunc context.CancelFunc
}

// Config for the download manager.
type Config struct {
	DownloadDir   string
	MaxConcurrent int
	Broker        *events.EventBroker
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
		workers: make(chan struct{}, maxConcurrent),
	}
}

// Submit creates a new download job and starts it.
func (m *Manager) Submit(url, service, quality string) *models.DownloadJob {
	job := &Job{
		DownloadJob: models.DownloadJob{
			ID:        generateID(),
			URL:       url,
			Service:   service,
			Status:    "queued",
			Quality:   quality,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	// Broadcast job created
	m.broadcast("job.created", job.DownloadJob)

	// Start download in goroutine
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
	return &job.DownloadJob
}

// List returns all jobs.
func (m *Manager) List() []*models.DownloadJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*models.DownloadJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		jobs = append(jobs, &j.DownloadJob)
	}
	return jobs
}

// Pause pauses a running job.
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("job not found: %s", id)
	}
	if job.Status != "running" {
		m.mu.Unlock()
		return fmt.Errorf("job not running")
	}
	if job.CancelFunc != nil {
		job.CancelFunc()
	}
	job.Status = "paused"
	job.StartedAt = time.Now().UTC().Format(time.RFC3339)
	m.mu.Unlock()

	m.broadcast("job.updated", job.DownloadJob)
	return nil
}

// Cancel cancels a job.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("job not found: %s", id)
	}
	if job.CancelFunc != nil {
		job.CancelFunc()
	}
	job.Status = "cancelled"
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	m.mu.Unlock()

	m.broadcast("job.updated", job.DownloadJob)
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
	if job.Status == "running" && job.CancelFunc != nil {
		job.CancelFunc()
	}
	delete(m.jobs, id)
	return nil
}

// execute runs the actual download subprocess.
func (m *Manager) execute(job *Job) {
	// Acquire worker slot
	m.workers <- struct{}{}
	defer func() { <-m.workers }()

	ctx, cancel := context.WithCancel(context.Background())
	job.CancelFunc = cancel

	m.mu.Lock()
	job.Status = "running"
	job.StartedAt = time.Now().UTC().Format(time.RFC3339)
	m.mu.Unlock()

	m.broadcast("job.updated", job.DownloadJob)

	var cmd *exec.Cmd
	provider := detectProvider(job.URL)

	switch provider {
	case "tidal":
		cmd = exec.CommandContext(ctx, "tidal-dl-ng", "dl", job.URL)
	case "qobuz", "deezer":
		cmd = exec.CommandContext(ctx, "rip", "url", job.URL)
	default:
		// Try streamrip as default
		cmd = exec.CommandContext(ctx, "rip", "url", job.URL)
	}

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
	)

	// Capture output for progress parsing
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		m.mu.Lock()
		job.Status = "failed"
		job.ErrorMsg = err.Error()
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		m.mu.Unlock()
		m.broadcast("job.updated", job.DownloadJob)
		return
	}

	// Parse progress from output
	go parseProgress(stdout, job, m)

	err := cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		// Context cancelled
		if job.Status != "paused" && job.Status != "cancelled" {
			job.Status = "cancelled"
		}
	} else if err != nil {
		job.Status = "failed"
		job.ErrorMsg = err.Error()
	} else {
		job.Status = "completed"
		job.Progress = 100
	}
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)

	m.broadcast("job.updated", job.DownloadJob)
}

// detectProvider guesses the provider from a URL.
func detectProvider(url string) string {
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

// parseProgress tries to extract download progress from subprocess output.
var progressRegex = regexp.MustCompile(`(\d+)%`)

func parseProgress(r interface{ Read([]byte) (int, error) }, job *Job, m *Manager) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			text := string(buf[:n])
			if matches := progressRegex.FindStringSubmatch(text); len(matches) > 1 {
				if pct, e := strconv.Atoi(matches[1]); e == nil {
					m.mu.Lock()
					job.Progress = float64(pct)
					m.mu.Unlock()
					m.broadcast("job.updated", job.DownloadJob)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

func (m *Manager) broadcast(event string, data models.DownloadJob) {
	if m.broker != nil {
		m.broker.Broadcast(event, data)
	}
}

func generateID() string {
	return fmt.Sprintf("dl-%d", time.Now().UnixNano())
}

func init() {
	// Ensure download directory exists
	_ = os.MkdirAll(".", 0o755)
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// ScanDir finds downloaded files in a directory.
func ScanDir(dir string) ([]models.Track, error) {
	var tracks []models.Track

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".flac", ".mp3", ".ogg", ".opus", ".m4a", ".aac", ".wav":
			// Looks like an audio file
			tracks = append(tracks, models.Track{
				ID:         fmt.Sprintf("local:%s", path),
				Provider:   "local",
				FilePath:   path,
				FileFormat: strings.TrimPrefix(ext, "."),
				FileSize:   info.Size(),
				Downloaded: true,
			})
		}
		return nil
	})

	return tracks, err
}
