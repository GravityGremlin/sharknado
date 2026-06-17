package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	mux.HandleFunc("POST /api/downloads/{id}/resume", s.handleDownloadResume)
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
			cleanPath := filepath.Clean(r.URL.Path)
			if strings.Contains(cleanPath, "..") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			filePath := filepath.Join(frontendDir, cleanPath)
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
		"grouped": groupSearchResults(results),
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
		"grouped": groupSearchResults(results),
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
	if unescaped, err := url.PathUnescape(id); err == nil {
		id = unescaped
	}
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

	// Check DB first for file path
	var sourcePath string
	if t, err := s.db.GetTrack(id); err == nil && t.FilePath != "" {
		sourcePath = t.FilePath
	} else {
		// Fallback to slower walk over filesystem
		sourcePath, err = s.player.FindSourceFile(provider, providerID)
	}

	if sourcePath == "" {
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
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10) // 16KB limit
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

func (s *Server) handleDownloadResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.download.Resume(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "queued"})
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

type addTrackRequest struct {
	TrackID string `json:"track_id"`
}

func (s *Server) handlePlaylistList(w http.ResponseWriter, r *http.Request) {
	playlists, err := s.db.ListPlaylists()
	if err != nil {
		log.Printf("list playlists: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list playlists"})
		return
	}
	if playlists == nil {
		playlists = []models.Playlist{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"playlists": playlists})
}

func (s *Server) handlePlaylistCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10) // 16KB limit
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	p := &models.Playlist{
		ID:   fmt.Sprintf("pl-%d", time.Now().UnixNano()),
		Name: req.Name,
		Description: req.Description,
	}
	if err := s.db.InsertPlaylist(p); err != nil {
		log.Printf("create playlist: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create playlist"})
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handlePlaylistGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	playlist, err := s.db.GetPlaylist(id)
	if err != nil {
		log.Printf("get playlist %s: %v", id, err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "playlist not found"})
		return
	}
	writeJSON(w, http.StatusOK, playlist)
}

func (s *Server) handlePlaylistUpdate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10) // 16KB limit
	id := r.PathValue("id")
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if err := s.db.UpdatePlaylist(id, &models.Playlist{Name: req.Name, Description: req.Description}); err != nil {
		log.Printf("update playlist %s: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update playlist"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handlePlaylistDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeletePlaylist(id); err != nil {
		log.Printf("delete playlist %s: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete playlist"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePlaylistAddTrack(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10) // 16KB limit
	id := r.PathValue("id")
	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	// Get current track count for position
	pl, err := s.db.GetPlaylist(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "playlist not found"})
		return
	}
	if err := s.db.AddTrackToPlaylist(id, req.TrackID, len(pl.Tracks)); err != nil {
		log.Printf("add track to playlist: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add track"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "added", "track_id": req.TrackID})
}

func (s *Server) handlePlaylistRemoveTrack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	trackID := r.PathValue("trackId")
	if err := s.db.RemoveTrackFromPlaylist(id, trackID); err != nil {
		log.Printf("remove track from playlist: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove track"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Library ─────────────────────────────────────────────────────────

func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.db.ListTracks()
	if err != nil {
		log.Printf("list tracks: %v", err)
		tracks = nil
	}
	albums, err := s.db.ListAlbums()
	if err != nil {
		log.Printf("list albums: %v", err)
		albums = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracks": tracks, "albums": albums})
}

func (s *Server) handleLibraryScan(w http.ResponseWriter, r *http.Request) {
	// Scan downloads directory
	dlTracks, err := download.ScanDir(s.cfg.DownloadDir)
	if err != nil {
		log.Printf("scan downloads: %v", err)
	}
	// Scan library directory
	libTracks, err := download.ScanDir(s.cfg.LibraryDir)
	if err != nil {
		log.Printf("scan library: %v", err)
	}

	allTracks := append(dlTracks, libTracks...)

	// Persist to DB
	imported := 0
	for i := range allTracks {
		if err := s.db.InsertTrack(&allTracks[i]); err != nil {
			log.Printf("import track: %v", err)
		} else {
			imported++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "scanned",
		"found":    len(allTracks),
		"imported": imported,
	})
}

// ── Helpers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// albumGroup is an album with its tracks in a grouped search response.
type albumGroup struct {
	AlbumID  string                `json:"album_id"`
	Album    string                `json:"album"`
	Artist   string                `json:"artist"`
	Provider string                `json:"provider"`
	CoverURL string                `json:"cover_url,omitempty"`
	Tracks   []models.SearchResult `json:"tracks"`
}

// artistGroup is an artist with their albums in a grouped search response.
type artistGroup struct {
	Artist string       `json:"artist"`
	Albums []albumGroup `json:"albums"`
}

// groupSearchResults groups flat search results by artist then album.
func groupSearchResults(results []models.SearchResult) []artistGroup {
	// Build maps preserving insertion order
	type artistAlbumKey struct{ artist, album, provider string }
	artistOrder := make([]string, 0)
	artistSeen := make(map[string]bool)
	albumOrder := make(map[string][]artistAlbumKey)
	albumSeen := make(map[artistAlbumKey]bool)
	albumMap := make(map[artistAlbumKey]*albumGroup)

	for _, r := range results {
		artist := r.Artist
		if artist == "" {
			artist = "Unknown Artist"
		}
		album := r.Album
		if album == "" {
			album = "Unknown Album"
		}

		if !artistSeen[artist] {
			artistSeen[artist] = true
			artistOrder = append(artistOrder, artist)
		}

		key := artistAlbumKey{artist, album, r.Provider}
		if !albumSeen[key] {
			albumSeen[key] = true
			albumOrder[artist] = append(albumOrder[artist], key)
			albumMap[key] = &albumGroup{
				AlbumID:  r.AlbumID,
				Album:    album,
				Artist:   artist,
				Provider: r.Provider,
				CoverURL: r.CoverURL,
			}
		}
		albumMap[key].Tracks = append(albumMap[key].Tracks, r)
	}

	grouped := make([]artistGroup, 0, len(artistOrder))
	for _, artist := range artistOrder {
		ag := artistGroup{Artist: artist}
		for _, key := range albumOrder[artist] {
			ag.Albums = append(ag.Albums, *albumMap[key])
		}
		grouped = append(grouped, ag)
	}
	return grouped
}
