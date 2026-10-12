package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL  string
	Port         string
	SyncInterval time.Duration
	GithubToken  string
}

func Load() Config {
	return Config{
		DatabaseURL:  getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/github_sync"),
		Port:         getenv("PORT", "4000"),
		SyncInterval: time.Duration(getenvInt("SYNC_INTERVAL_MINUTES", 5)) * time.Minute,
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
