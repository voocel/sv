package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	upgradeApi = "https://api.github.com/repos/voocel/sv/releases/latest"
)

type Upgrade struct {
	latestTag   string
	downloadURL string
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

func NewUpgrade() *Upgrade {
	return &Upgrade{}
}

func (u *Upgrade) checkUpgrade() error {
	fmt.Println("check version...")
	resp, err := http.Get(upgradeApi)
	if err != nil {
		return err
	}

	latest := &Release{}
	err = json.NewDecoder(resp.Body).Decode(latest)
	if err != nil {
		return err
	}

	if versionCompare(Ver) >= versionCompare(latest.TagName) {
		return errors.New("it's already the latest version")
	}
	u.downloadURL = latest.Assets[0].BrowserDownloadURL

	return u.upgrade()
}

func (u *Upgrade) upgrade() error {
	filename := filepath.Base(u.downloadURL)
	path := filepath.Join(svBin, filename)

	fmt.Println("downloading the newest version...")
	resp, err := http.Get(u.downloadURL)
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

	fmt.Println("upgrade success!")
	return os.Rename(filepath.Join(svBin, filename), filepath.Join(svBin, "sv"))
}
