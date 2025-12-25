package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type Upgrade struct {
	force  bool
	client *http.Client
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewUpgrade(force bool) *Upgrade {
	return &Upgrade{
		force:  force,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (u *Upgrade) checkUpgrade() error {
	PrintCyan("Checking version...")

	release, err := u.fetchLatestRelease()
	if err != nil {
		return err
	}

	if !u.force && versionCompare(Ver) >= versionCompare(release.TagName) {
		return ErrAlreadyLatest(release.TagName)
	}

	asset := u.findMatchingAsset(release.Assets)
	if asset == nil {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	PrintBlue(fmt.Sprintf("Upgrading to %s (%s)", release.TagName, asset.Name))
	return u.downloadAndInstall(asset)
}

func (u *Upgrade) fetchLatestRelease() (*Release, error) {
	resp, err := u.client.Get(cfg.UpgradeAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, ErrLatestVersionFailed()
	}
	return &release, nil
}

func (u *Upgrade) findMatchingAsset(assets []Asset) *Asset {
	expected := fmt.Sprintf("sv-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expected += ".exe"
	}

	for i := range assets {
		if assets[i].Name == expected {
			return &assets[i]
		}
	}
	return nil
}

func (u *Upgrade) downloadAndInstall(asset *Asset) error {
	resp, err := u.client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	bar := NewBar(asset.Size)
	bar.SetName("sv[upgrade]", "cyan")

	tmpFile := filepath.Join(paths.Bin, ".sv.tmp")
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to download: %w", err)
	}

	targetPath := filepath.Join(paths.Bin, "sv")
	if runtime.GOOS == "windows" {
		targetPath += ".exe"
		oldPath := targetPath + ".old"
		os.Remove(oldPath)
		os.Rename(targetPath, oldPath)
	}

	if err := os.Rename(tmpFile, targetPath); err != nil {
		return fmt.Errorf("failed to install: %w", err)
	}

	PrintGreen("Upgrade successful!")
	return nil
}
