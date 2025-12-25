package main

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	UpgradeAPIURL string
	HTTPTimeout   time.Duration
	DownloadRetry int
	Debug         bool
}

var defaultConfig = &Config{
	UpgradeAPIURL: "https://api.github.com/repos/voocel/sv/releases/latest",
	HTTPTimeout:   30 * time.Second,
	DownloadRetry: 3,
	Debug:         false,
}

var cfg *Config

func init() {
	cfg = loadConfig()
}

func loadConfig() *Config {
	config := &Config{
		UpgradeAPIURL: getEnv("SV_UPGRADE_API_URL", defaultConfig.UpgradeAPIURL),
		HTTPTimeout:   getEnvDuration("SV_HTTP_TIMEOUT", defaultConfig.HTTPTimeout),
		DownloadRetry: getEnvInt("SV_DOWNLOAD_RETRY", defaultConfig.DownloadRetry),
		Debug:         getEnvBool("SV_DEBUG", defaultConfig.Debug),
	}

	if config.Debug {
		SetLogLevel("debug")
	}

	return config
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
