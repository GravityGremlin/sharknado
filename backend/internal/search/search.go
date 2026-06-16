// Package search provides cross-provider music search.
// Makes direct API calls to Qobuz, Deezer, and Tidal.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sharknado/backend/internal/models"
)

const (
	defaultTimeout = 15 * time.Second
	qobuzSearchURL = "https://api.qobuz.com/v2.1/search/queries"
	deezerSearchURL = "https://api.deezer.com/search"
	tidalSearchURL = "https://api.tidal.com/v1/search"
)

// Config holds auth config paths.
type Config struct {
	StreamripConfigPath string // path to streamrip config.toml
	TidalTokenPath      string // path to tidal token.json
}

// Engine orchestrates parallel searches across all providers.
type Engine struct {
	config    Config
	cache     *searchCache
	http      *http.Client
	qobuzApp  string
	qobuzToken string
	tidalToken string
	tidalRefresh string
}

// NewEngine creates a search engine with all providers.
func NewEngine(cfg Config) *Engine {
	e := &Engine{
		config: cfg,
		cache:  newSearchCache(60 * time.Second),
		http:   &http.Client{Timeout: defaultTimeout},
	}

	// Load Qobuz auth from streamrip config.toml
	e.loadQobuzAuth()

	// Load Tidal auth from token.json
	e.loadTidalAuth()

	return e
}

// Search queries all (or specified) providers in parallel and returns merged results.
func (e *Engine) Search(ctx context.Context, query string, services []string) ([]models.SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	// Check cache
	if cached := e.cache.get(query, services); cached != nil {
		return cached, nil
	}

	// Filter providers
	targets := e.filterProviders(services)

	type providerResult struct {
		results []models.SearchResult
		err     error
		name    string
	}

	resultsCh := make(chan providerResult, len(targets))

	for _, name := range targets {
		go func(name string) {
			ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
			defer cancel()

			var results []models.SearchResult
			var err error

			switch name {
			case "qobuz":
				results, err = e.searchQobuz(ctx, query)
			case "deezer":
				results, err = e.searchDeezer(ctx, query)
			case "tidal":
				results, err = e.searchTidal(ctx, query)
			}

			resultsCh <- providerResult{results: results, err: err, name: name}
		}(name)
	}

	// Collect results
	var all []models.SearchResult
	var mu sync.Mutex
	var errs []string

	for i := 0; i < len(targets); i++ {
		r := <-resultsCh
		if r.err != nil {
			log.Printf("search %s failed: %v", r.name, r.err)
			errs = append(errs, fmt.Sprintf("%s: %s", r.name, r.err.Error()))
			continue
		}
		mu.Lock()
		all = append(all, r.results...)
		mu.Unlock()
	}

	// Sort by provider, then title
	sort.Slice(all, func(i, j int) bool {
		if all[i].Provider != all[j].Provider {
			return all[i].Provider < all[j].Provider
		}
		return all[i].Title < all[j].Title
	})

	// Cache results
	e.cache.set(query, services, all)

	if len(all) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all providers failed: [%s]", strings.Join(errs, " "))
	}

	return all, nil
}

// ── Qobuz Search ───────────────────────────────────────────────────

func (e *Engine) searchQobuz(ctx context.Context, query string) ([]models.SearchResult, error) {
	if e.qobuzApp == "" {
		return nil, fmt.Errorf("qobuz app_id not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", qobuzSearchURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("query", query)
	q.Set("limit", "15")
	q.Set("type", "tracks")
	q.Set("app_id", e.qobuzApp)
	req.URL.RawQuery = q.Encode()

	if e.qobuzToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.qobuzToken)
	}

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qobuz request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qobuz returned %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Tracks struct {
			Items []struct {
				ID        int    `json:"id"`
				Title     string `json:"title"`
				Performers []struct {
					Name string `json:"name"`
				} `json:"performers"`
				Album struct {
					Title string `json:"title"`
				} `json:"album"`
				Duration int `json:"duration"`
				Cover    struct {
					Small string `json:"small"`
				} `json:"cover"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode qobuz response: %w", err)
	}

	results := make([]models.SearchResult, 0, len(data.Tracks.Items))
	for _, t := range data.Tracks.Items {
		artist := ""
		if len(t.Performers) > 0 {
			artist = t.Performers[0].Name
		}
		results = append(results, models.SearchResult{
			ID:       fmt.Sprintf("qobuz:%d", t.ID),
			Provider: "qobuz",
			Title:    t.Title,
			Artist:   artist,
			Album:    t.Album.Title,
			Duration: float64(t.Duration),
			CoverURL: t.Cover.Small,
			Type:     "track",
		})
	}

	return results, nil
}

// ── Deezer Search ──────────────────────────────────────────────────

func (e *Engine) searchDeezer(ctx context.Context, query string) ([]models.SearchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", deezerSearchURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("q", query)
	q.Set("limit", "15")
	req.URL.RawQuery = q.Encode()

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deezer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deezer returned %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Data []struct {
			ID       int    `json:"id"`
			Title    string `json:"title"`
			Artist   struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int    `json:"duration"`
			AlbumCover string `json:"album_cover"` // not standard but sometimes present
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode deezer response: %w", err)
	}

	results := make([]models.SearchResult, 0, len(data.Data))
	for _, t := range data.Data {
		cover := fmt.Sprintf("https://e-cdns-images.dzcdn.net/images/artist/%s/250x250.jpg", t.Artist.Name)
		results = append(results, models.SearchResult{
			ID:       fmt.Sprintf("deezer:%d", t.ID),
			Provider: "deezer",
			Title:    t.Title,
			Artist:   t.Artist.Name,
			Album:    t.Album.Title,
			Duration: float64(t.Duration),
			CoverURL: cover,
			Type:     "track",
		})
	}

	return results, nil
}

// ── Tidal Search ───────────────────────────────────────────────────

func (e *Engine) searchTidal(ctx context.Context, query string) ([]models.SearchResult, error) {
	if e.tidalToken == "" {
		return nil, fmt.Errorf("tidal token not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", tidalSearchURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("query", query)
	q.Set("limit", "15")
	q.Set("types", "TRACKS")
	q.Set("countryCode", "US")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+e.tidalToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Sharknado/0.1.0")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tidal request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Try to refresh token
		if refreshed, refreshErr := e.refreshTidalToken(); refreshed && refreshErr == nil {
			return e.searchTidal(ctx, query) // retry once
		}
		return nil, fmt.Errorf("tidal unauthorized (token expired?)")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tidal returned %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Tracks struct {
			Items []struct {
				ID       int    `json:"id"`
				Title    string `json:"title"`
				Artist   struct {
					Name string `json:"name"`
				} `json:"artist"`
				Album struct {
					Title string `json:"title"`
					Cover string `json:"cover"`
				} `json:"album"`
				Duration int `json:"duration"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode tidal response: %w", err)
	}

	results := make([]models.SearchResult, 0, len(data.Tracks.Items))
	for _, t := range data.Tracks.Items {
		results = append(results, models.SearchResult{
			ID:       fmt.Sprintf("tidal:%d", t.ID),
			Provider: "tidal",
			Title:    t.Title,
			Artist:   t.Artist.Name,
			Album:    t.Album.Title,
			Duration: float64(t.Duration),
			CoverURL: t.Album.Cover,
			Type:     "track",
		})
	}

	return results, nil
}

// refreshTidalToken attempts to refresh the Tidal access token.
func (e *Engine) refreshTidalToken() (bool, error) {
	if e.tidalRefresh == "" {
		return false, fmt.Errorf("no refresh token")
	}

	// Read client_id from settings.json
	settingsPath := strings.Replace(e.config.TidalTokenPath, "token.json", "settings.json", 1)
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		return false, err
	}
	var settings struct {
		QualityAudio string `json:"quality_audio"`
	}
	json.Unmarshal(settingsData, &settings)

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", e.tidalRefresh)
	data.Set("client_id", "zU4XHVVkc2tDPo4t") // tidal-dl-ng default client_id

	req, err := http.NewRequest("POST", "https://auth.tidal.com/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := e.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("token refresh failed: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return false, err
	}

	e.tidalToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		e.tidalRefresh = tokenResp.RefreshToken
	}

	return true, nil
}

// ── Config Loading ─────────────────────────────────────────────────

func (e *Engine) loadQobuzAuth() {
	path := e.config.StreamripConfigPath
	if path == "" {
		path = defaultStreamripConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("qobuz config not found at %s: %v", path, err)
		return
	}

	// Parse TOML manually (avoid adding a dependency)
	lines := strings.Split(string(data), "\n")
	inQobuz := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[qobuz]" {
			inQobuz = true
			continue
		}
		if strings.HasPrefix(line, "[") && inQobuz {
			inQobuz = false
			continue
		}
		if inQobuz && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"")

			switch key {
			case "app_id":
				e.qobuzApp = val
			case "password_or_token":
				e.qobuzToken = val
			}
		}
	}

	if e.qobuzApp != "" {
		log.Printf("qobuz auth loaded (app_id=%s)", e.qobuzApp)
	}
}

func (e *Engine) loadTidalAuth() {
	path := e.config.TidalTokenPath
	if path == "" {
		path = defaultTidalTokenPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("tidal token not found at %s: %v", path, err)
		return
	}

	var token struct {
		TokenType    string  `json:"token_type"`
		AccessToken  string  `json:"access_token"`
		RefreshToken string  `json:"refresh_token"`
		ExpiryTime   float64 `json:"expiry_time"`
	}
	if err := json.Unmarshal(data, &token); err != nil {
		log.Printf("parse tidal token: %v", err)
		return
	}

	e.tidalToken = token.AccessToken
	e.tidalRefresh = token.RefreshToken

	if e.tidalToken != "" {
		log.Printf("tidal auth loaded (token present, expiry=%.0f)", token.ExpiryTime)
	}
}

func (e *Engine) filterProviders(names []string) []string {
	if len(names) == 0 {
		return []string{"qobuz", "deezer", "tidal"}
	}

	var filtered []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		switch n {
		case "qobuz", "deezer", "tidal":
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func defaultStreamripConfigPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.config/streamrip/config.toml"
}

func defaultTidalTokenPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.config/tidal_dl_ng/token.json"
}

// ParseID splits "provider:id" into provider and id parts.
func ParseID(id string) (provider, providerID string) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", id
}

// FormatDuration formats seconds as m:ss.
func FormatDuration(d float64) string {
	if d <= 0 {
		return "0:00"
	}
	m := int(d) / 60
	s := int(d) % 60
	return strconv.Itoa(m) + ":" + strconv.Itoa(s)
}
