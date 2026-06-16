package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port               string
	DownloadDir        string
	LibraryDir         string
	CacheDir           string
	DBPath             string
	TidalConfigDir     string
	StreamripConfigDir string
	FrontendDir        string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8000"),
		DownloadDir:        getEnv("DOWNLOAD_DIR", "./downloads"),
		LibraryDir:         getEnv("LIBRARY_DIR", "./library"),
		CacheDir:           getEnv("CACHE_DIR", "./cache"),
		DBPath:             getEnv("DB_PATH", "./data/sharknado.db"),
		TidalConfigDir:     getEnv("TIDAL_CONFIG_DIR", "./data/tidal-config"),
		StreamripConfigDir: getEnv("STREAMRIP_CONFIG_DIR", "./data/streamrip-config"),
		FrontendDir:        getEnv("FRONTEND_DIR", "../frontend/dist"),
	}
}

// PortInt returns the port as an integer.
func (c *Config) PortInt() int {
	p, err := strconv.Atoi(c.Port)
	if err != nil {
		return 8000
	}
	return p
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
