package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds runtime settings, each overridable via an SV_* environment
// variable. sv deliberately has no config file.
type Config struct {
	UpgradeAPIURL string        // release API used by `sv self upgrade`
	HTTPTimeout   time.Duration // timeout for API requests
	DownloadRetry int           // download attempts before giving up
}

func loadConfig() Config {
	return Config{
		UpgradeAPIURL: envStr("SV_UPGRADE_API_URL", "https://api.github.com/repos/voocel/sv/releases/latest"),
		HTTPTimeout:   envDuration("SV_HTTP_TIMEOUT", 30*time.Second),
		DownloadRetry: envInt("SV_DOWNLOAD_RETRY", 3),
	}
}

// Paths is the sv directory layout under ~/.sv.
type Paths struct {
	Home      string // ~/.sv
	Root      string // ~/.sv/go — symlink to the active version
	Bin       string // ~/.sv/bin — the sv binary itself
	Cache     string // ~/.sv/cache — installed versions, one dir per tag
	Downloads string // ~/.sv/downloads — archives in flight
}

func newPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}

	base := filepath.Join(home, ".sv")
	p := Paths{
		Home:      base,
		Root:      filepath.Join(base, "go"),
		Bin:       filepath.Join(base, "bin"),
		Cache:     filepath.Join(base, "cache"),
		Downloads: filepath.Join(base, "downloads"),
	}

	for _, dir := range []string{p.Bin, p.Cache, p.Downloads} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Paths{}, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return p, nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
