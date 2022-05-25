package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	upgradeApi = "https://api.github.com/repos/voocel/sv/releases/latest"
)

type Upgrade struct {
	latestTag string
	DownloadURL string
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

func checkUpgrade() (bool, error) {
	resp, err := http.Get(upgradeApi)
	if err != nil {
		return false, err
	}
	latest := &Release{}
	err = json.NewDecoder(resp.Body).Decode(latest)
	if err != nil {
		return false, err
	}
	if versionCompare(latest.TagName) > versionCompare(Ver) {
		return true, nil
	}
	return false, nil
}

func versionCompare(version string) string {
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}
	const maxByte = 1<<8 - 1
	vo := make([]byte, 0, len(version)+8)
	j := -1
	for i := 0; i < len(version); i++ {
		b := version[i]
		if '0' > b || b > '9' {
			vo = append(vo, b)
			j = -1
			continue
		}
		if j == -1 {
			vo = append(vo, 0x00)
			j = len(vo) - 1
		}
		if vo[j] == 1 && vo[j+1] == '0' {
			vo[j+1] = b
			continue
		}
		if vo[j]+1 > maxByte {
			panic("invalid version")
		}
		vo = append(vo, b)
		vo[j]++
	}
	return string(vo)
}
