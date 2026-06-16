package models

// Track represents a music track from any provider or local file.
type Track struct {
	ID          string  `json:"id"`
	Provider    string  `json:"provider"`
	ProviderID  string  `json:"provider_id"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	AlbumArtist string  `json:"album_artist,omitempty"`
	TrackNumber int     `json:"track_number,omitempty"`
	DiscNumber  int     `json:"disc_number,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	Genre       string  `json:"genre,omitempty"`
	Year        int     `json:"year,omitempty"`
	CoverURL    string  `json:"cover_url,omitempty"`
	FilePath    string  `json:"file_path,omitempty"`
	FileFormat  string  `json:"file_format,omitempty"`
	FileSize    int64   `json:"file_size,omitempty"`
	Downloaded  bool    `json:"downloaded"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

// Album groups tracks by album.
type Album struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Year       int     `json:"year,omitempty"`
	CoverURL   string  `json:"cover_url,omitempty"`
	TrackCount int     `json:"track_count"`
	Tracks     []Track `json:"tracks,omitempty"`
}

// Playlist is a user-created collection of tracks.
type Playlist struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	CoverURL    string  `json:"cover_url,omitempty"`
	TrackCount  int     `json:"track_count"`
	CreatedAt   string  `json:"created_at,omitempty"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
	Tracks      []Track `json:"tracks,omitempty"`
}

// DownloadJob tracks a provider download operation.
type DownloadJob struct {
	ID         string  `json:"id"`
	URL        string  `json:"url"`
	Service    string  `json:"service"`
	Status     string  `json:"status"` // queued, running, paused, completed, failed
	Quality    string  `json:"quality,omitempty"`
	Progress   float64 `json:"progress"`
	ErrorMsg   string  `json:"error_msg,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	StartedAt  string  `json:"started_at,omitempty"`
	FinishedAt string  `json:"finished_at,omitempty"`
}

// SearchResult is a normalized result from any provider.
type SearchResult struct {
	ID       string  `json:"id"`
	Provider string  `json:"provider"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album,omitempty"`
	AlbumID  string  `json:"album_id,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	CoverURL string  `json:"cover_url,omitempty"`
	Type     string  `json:"type"` // track, album, playlist
	Quality  string  `json:"quality,omitempty"`
}

// StreamFormat represents an audio transcoding target.
type StreamFormat string

const (
	StreamFormatOpus StreamFormat = "opus"
	StreamFormatFlac StreamFormat = "flac"
	StreamFormatMP3  StreamFormat = "mp3"
)
