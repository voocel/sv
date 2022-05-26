package main

import (
	"encoding/json"
	"errors"
	"net/http"
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

	return err
}


