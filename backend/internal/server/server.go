package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sharknado/backend/internal/config"
	"github.com/sharknado/backend/internal/download"
	"github.com/sharknado/backend/internal/events"
	"github.com/sharknado/backend/internal/library"
	"github.com/sharknado/backend/internal/player"
	"github.com/sharknado/backend/internal/search"
)

// Server holds dependencies and state for the HTTP server.
type Server struct {
	cfg      *config.Config
	db       *library.DB
	broker   *events.EventBroker
	search   *search.Engine
	download *download.Manager
	player   *player.Streamer
	mux      *http.ServeMux
}

// NewServer creates a new Server with the given dependencies.
func NewServer(cfg *config.Config, db *library.DB, broker *events.EventBroker) *Server {
	s := &Server{
		cfg:    cfg,
		db:     db,
		broker: broker,
	}

	// Initialize search engine
	s.search = search.NewEngine(search.Config{
		StreamripConfigPath: cfg.StreamripConfigDir + "/config.toml",
		TidalTokenPath:      cfg.TidalConfigDir + "/token.json",
	})

	// Initialize download manager
	s.download = download.NewManager(download.Config{
		DownloadDir:   cfg.DownloadDir,
		MaxConcurrent: 3,
		Broker:        broker,
		DB:            db,
	})

	// Initialize player/streamer
	s.player = player.NewStreamer(player.StreamConfig{
		CacheDir:   cfg.CacheDir,
		LibraryDir: cfg.LibraryDir,
		FFmpegPath: "ffmpeg",
	})

	s.registerRoutes()
	return s
}

// Start begins serving HTTP requests and blocks until shutdown.
func (s *Server) Start() error {
	addr := ":" + s.cfg.Port

	handler := corsMiddleware(s.cfg.AllowedOrigins, s.mux)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Sharknado listening on :%s", s.cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for signal
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(ctx)
}

// corsMiddleware adds CORS headers.
func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins == "*" || allowedOrigins == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && stringsContains(strings.Split(allowedOrigins, ","), origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func stringsContains(list []string, target string) bool {
	for _, s := range list {
		if strings.TrimSpace(s) == target {
			return true
		}
	}
	return false
}
