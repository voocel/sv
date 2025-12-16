package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Upgrade struct {
	force       bool
	latestTag   string
	downloadURL string
	client      *http.Client
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewUpgrade(force bool) *Upgrade {
	return &Upgrade{
		force: force,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
}

func (u *Upgrade) checkUpgrade() error {
	PrintCyan("Checking version...")
	resp, err := u.client.Get(cfg.UpgradeAPIURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	latest := &Release{}
	err = json.NewDecoder(resp.Body).Decode(latest)
	if err != nil {
		return ErrLatestVersionFailed()
	}

	if !u.force && versionCompare(Ver) >= versionCompare(latest.TagName) {
		return ErrAlreadyLatest(latest.TagName)
	}

	// Detect current system architecture and find matching binary
	// Binary naming convention: sv-{os}-{arch} where arch uses hyphen (e.g., arm-64, amd-64)
	osName := runtime.GOOS
	archName := runtime.GOARCH
	// Convert GOARCH format to release asset format
	switch archName {
	case "amd64":
		archName = "amd-64"
	case "arm64":
		archName = "arm-64"
	}
	currentArch := fmt.Sprintf("%s-%s", osName, archName)
	var matchedAsset *Asset

	for _, asset := range latest.Assets {
		if strings.Contains(asset.Name, currentArch) {
			matchedAsset = &asset
			break
		}
	}

	if matchedAsset == nil {
		return fmt.Errorf("no binary found for current architecture (%s)", currentArch)
	}

	u.downloadURL = matchedAsset.BrowserDownloadURL
	PrintBlue(fmt.Sprintf("Found matching version: %s", matchedAsset.Name))

	return u.upgrade()
}

func (u *Upgrade) upgrade() error {
	filename := filepath.Base(u.downloadURL)
	downloadPath := filepath.Join(paths.Bin, filename)

	PrintBlue("Downloading the newest version...")
	resp, err := u.client.Get(u.downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download upgrade: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err = io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Ensure file is closed before rename
	f.Close()

	targetPath := filepath.Join(paths.Bin, "sv")
	if err = os.Rename(downloadPath, targetPath); err != nil {
		return fmt.Errorf("failed to rename binary: %w", err)
	}

	if err = os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	PrintGreen("Upgrade successful!")
	return nil
}
