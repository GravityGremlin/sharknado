package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sharknado/backend/internal/download"
	"github.com/sharknado/backend/internal/models"
)

// registerRoutes sets up all API and static file routes.
func (s *Server) registerRoutes() {
	mux := http.NewServeMux()

	// ── API Routes ────────────────────────────────────────────────

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/search/{service}", s.handleSearchService)
	mux.HandleFunc("GET /api/track/{id}/info", s.handleTrackInfo)
	mux.HandleFunc("GET /api/track/{id}/stream", s.handleTrackStream)
	mux.HandleFunc("GET /api/track/{id}/download", s.handleTrackDownload)
	mux.HandleFunc("POST /api/download", s.handleDownloadCreate)
	mux.HandleFunc("GET /api/downloads", s.handleDownloadList)
	mux.HandleFunc("GET /api/downloads/{id}", s.handleDownloadGet)
	mux.HandleFunc("POST /api/downloads/{id}/pause", s.handleDownloadPause)
	mux.HandleFunc("POST /api/downloads/{id}/cancel", s.handleDownloadCancel)
	mux.HandleFunc("DELETE /api/downloads/{id}", s.handleDownloadDelete)
	mux.HandleFunc("GET /api/playlists", s.handlePlaylistList)
	mux.HandleFunc("POST /api/playlists", s.handlePlaylistCreate)
	mux.HandleFunc("GET /api/playlists/{id}", s.handlePlaylistGet)
	mux.HandleFunc("PUT /api/playlists/{id}", s.handlePlaylistUpdate)
	mux.HandleFunc("DELETE /api/playlists/{id}", s.handlePlaylistDelete)
	mux.HandleFunc("POST /api/playlists/{id}/tracks", s.handlePlaylistAddTrack)
	mux.HandleFunc("DELETE /api/playlists/{id}/tracks/{trackId}", s.handlePlaylistRemoveTrack)
	mux.HandleFunc("GET /api/library", s.handleLibraryList)
	mux.HandleFunc("POST /api/library/scan", s.handleLibraryScan)
	mux.HandleFunc("GET /api/events", s.broker.ServeHTTP)

	// ── Static frontend ───────────────────────────────────────────
	frontendDir := s.cfg.FrontendDir
	if fi, err := os.Stat(frontendDir); err == nil && fi.IsDir() {
		log.Printf("serving frontend from %s", frontendDir)
		assetsDir := filepath.Join(frontendDir, "assets")
		if _, err := os.Stat(assetsDir); err == nil {
			fs := http.FileServer(http.Dir(frontendDir))
			mux.Handle("GET /assets/", fs)
		}
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			filePath := filepath.Join(frontendDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, filePath)
				return
			}
			index := filepath.Join(frontendDir, "index.html")
			if _, err := os.Stat(index); err == nil {
				http.ServeFile(w, r, index)
				return
			}
			http.NotFound(w, r)
		})
	} else {
		log.Printf("frontend dist directory not found at %s, serving API only", frontendDir)
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "info": "API only"})
		})
	}

	s.mux = mux
}

// ── Health ──────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "0.1.0",
	})
}

// ── Search ──────────────────────────────────────────────────────────

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	var services []string
	if svcs := r.URL.Query().Get("services"); svcs != "" {
		services = strings.Split(svcs, ",")
	}

	results, err := s.search.Search(r.Context(), query, services)
	if err != nil {
		log.Printf("search error: %v", err)
		writeJSON(w, http.StatusOK, map[string]any{
			"query":   query,
			"results": []any{},
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
	})
}

func (s *Server) handleSearchService(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	results, err := s.search.Search(r.Context(), query, []string{service})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"service": service,
			"query":   query,
			"results": []any{},
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"service": service,
		"query":   query,
		"results": results,
	})
}

// ── Track ───────────────────────────────────────────────────────────

func (s *Server) handleTrackInfo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// For now, return the ID — real implementation will query DB
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"title": "",
	})
}

func (s *Server) handleTrackStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	format := models.StreamFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = models.StreamFormatOpus
	}

	// Parse provider:provider_id from ID
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid track id format, expected provider:id"})
		return
	}
	provider, providerID := parts[0], parts[1]

	// Find the source file
	sourcePath, err := s.player.FindSourceFile(provider, providerID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "track not downloaded yet",
			"id":    id,
		})
		return
	}

	contentType := "audio/ogg"
	switch format {
	case models.StreamFormatMP3:
		contentType = "audio/mpeg"
	case models.StreamFormatFlac:
		contentType = "audio/flac"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if _, err := s.player.Stream(r.Context(), sourcePath, format, w); err != nil {
		log.Printf("stream error for %s: %v", id, err)
	}
}

func (s *Server) handleTrackDownload(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error": "direct track download not yet implemented",
	})
}

// ── Downloads ───────────────────────────────────────────────────────

type downloadRequest struct {
	URL     string `json:"url"`
	Service string `json:"service"`
	Quality string `json:"quality"`
}

func (s *Server) handleDownloadCreate(w http.ResponseWriter, r *http.Request) {
	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}

	job := s.download.Submit(req.URL, req.Service, req.Quality)
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleDownloadList(w http.ResponseWriter, r *http.Request) {
	jobs := s.download.List()
	if jobs == nil {
		jobs = []*models.DownloadJob{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleDownloadGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job := s.download.Get(id)
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleDownloadPause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.download.Pause(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "paused"})
}

func (s *Server) handleDownloadCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.download.Cancel(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "cancelled"})
}

func (s *Server) handleDownloadDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.download.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Playlists ───────────────────────────────────────────────────────

type createPlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (s *Server) handlePlaylistList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"playlists": []any{}})
}

func (s *Server) handlePlaylistCreate(w http.ResponseWriter, r *http.Request) {
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": "placeholder", "name": req.Name})
}

func (s *Server) handlePlaylistGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "tracks": []any{}})
}

func (s *Server) handlePlaylistUpdate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handlePlaylistDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type addTrackRequest struct {
	TrackID string `json:"track_id"`
}

func (s *Server) handlePlaylistAddTrack(w http.ResponseWriter, r *http.Request) {
	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, http.StatusCreated, map[string]any{"status": "added", "track_id": req.TrackID})
}

func (s *Server) handlePlaylistRemoveTrack(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// ── Library ─────────────────────────────────────────────────────────

func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tracks": []any{}, "albums": []any{}})
}

func (s *Server) handleLibraryScan(w http.ResponseWriter, r *http.Request) {
	tracks, err := download.ScanDir(s.cfg.LibraryDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "scanned", "count": len(tracks), "tracks": tracks})
}

// ── Helpers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Download returns a reader for the given file path. Used for streaming.
func Download(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}
