package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	u.downloadURL = latest.Assets[0].BrowserDownloadURL

	return u.upgrade()
}

func (u *Upgrade) upgrade() error {
	filename := filepath.Base(u.downloadURL)
	path := filepath.Join(SVBin, filename)

	PrintBlue("Downloading the newest version...")
	resp, err := u.client.Get(u.downloadURL)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	PrintGreen("upgrade success!")
	if err = os.Rename(filepath.Join(SVBin, filename), filepath.Join(SVBin, "sv")); err != nil {
		return err
	}
	return os.Chmod(filepath.Join(SVBin, "sv"), 0755)
}
