package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

const (
	releaseAPI    = "https://go.dev/dl/?mode=json"
	releaseAPIAll = "https://go.dev/dl/?mode=json&include=all"
	downloadBase  = "https://go.dev/dl/"
)

// GoRelease is a Go release as reported by the official go.dev API.
type GoRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}

// GoFile is one downloadable artifact of a release.
type GoFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"` // "source", "archive", "installer"
}

// getJSON fetches url within the API timeout and decodes the response into v.
func (a *App) getJSON(ctx context.Context, url string, v any) error {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status %d", req.Host, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("parse response from %s: %w", req.Host, err)
	}
	return nil
}

// fetchReleases queries go.dev for available releases; all includes
// unstable and archived versions.
func (a *App) fetchReleases(ctx context.Context, all bool) ([]GoRelease, error) {
	url := releaseAPI
	if all {
		url = releaseAPIAll
	}
	var releases []GoRelease
	if err := a.getJSON(ctx, url, &releases); err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	return releases, nil
}

// latestStable returns the newest stable Go version tag.
func (a *App) latestStable(ctx context.Context) (string, error) {
	releases, err := a.fetchReleases(ctx, false)
	if err != nil {
		return "", err
	}
	if tag := latestStableTag(releases); tag != "" {
		return tag, nil
	}
	return "", fmt.Errorf("no stable release found")
}

func latestStableTag(releases []GoRelease) string {
	var best string
	for _, r := range releases {
		if r.Stable && (best == "" || compareTags(r.Version, best) > 0) {
			best = r.Version
		}
	}
	return best
}

func findRelease(releases []GoRelease, tag string) *GoRelease {
	tag = normalizeTag(tag)
	for i := range releases {
		if releases[i].Version == tag {
			return &releases[i]
		}
	}
	return nil
}

// releaseTags returns all version tags, newest first.
func releaseTags(releases []GoRelease) []string {
	tags := make([]string, len(releases))
	for i, r := range releases {
		tags[i] = r.Version
	}
	sortVersionsDesc(tags)
	return tags
}

// archiveFile picks the binary archive for the current platform.
func (r *GoRelease) archiveFile() *GoFile {
	for i := range r.Files {
		f := &r.Files[i]
		if f.OS == runtime.GOOS && f.Arch == runtime.GOARCH && f.Kind == "archive" {
			return f
		}
	}
	return nil
}

func (f *GoFile) url() string {
	return downloadBase + f.Filename
}
