package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
)

const (
	goDevAPI    = "https://go.dev/dl/?mode=json"
	goDevAPIAll = "https://go.dev/dl/?mode=json&include=all"
	goDevDL     = "https://go.dev/dl/"
)

// GoRelease represents a Go version release from the official API
type GoRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}

// GoFile represents a downloadable file for a Go release
type GoFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"` // "source", "archive", "installer"
}

// FetchReleases fetches Go releases from the official API
func FetchReleases(client *http.Client, includeAll bool) ([]GoRelease, error) {
	url := goDevAPI
	if includeAll {
		url = goDevAPIAll
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var releases []GoRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	return releases, nil
}

// FindMatchingFile finds the appropriate file for the current OS/Arch
func (r *GoRelease) FindMatchingFile() *GoFile {
	for i := range r.Files {
		f := &r.Files[i]
		if f.OS == runtime.GOOS && f.Arch == runtime.GOARCH && f.Kind == "archive" {
			return f
		}
	}
	return nil
}

// DownloadURL returns the full download URL for a file
func (f *GoFile) DownloadURL() string {
	return goDevDL + f.Filename
}

// ToPackage converts a GoFile to a Package for compatibility
func (f *GoFile) ToPackage(version string) *Package {
	return &Package{
		Name:      f.Filename,
		Tag:       version,
		URL:       f.DownloadURL(),
		Kind:      f.Kind,
		OS:        f.OS,
		Arch:      f.Arch,
		Checksum:  f.SHA256,
		Algorithm: "SHA256",
	}
}

// GetLatestVersion returns the latest stable Go version
func GetLatestVersion(client *http.Client) (string, error) {
	releases, err := FetchReleases(client, false)
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Sort by version (newest first)
	sort.Slice(releases, func(i, j int) bool {
		return versionCompare(releases[i].Version) > versionCompare(releases[j].Version)
	})

	return releases[0].Version, nil
}

// FindRelease finds a specific release by version tag
func FindRelease(releases []GoRelease, tag string) *GoRelease {
	normalized := normalizeVersionTag(tag)
	for i := range releases {
		if releases[i].Version == normalized {
			return &releases[i]
		}
	}
	return nil
}

// GetVersionList returns a sorted list of version strings
func GetVersionList(releases []GoRelease) []string {
	versions := make([]string, len(releases))
	for i, r := range releases {
		versions[i] = r.Version
	}

	sort.Slice(versions, func(i, j int) bool {
		return versionCompare(versions[i]) > versionCompare(versions[j])
	})

	return versions
}
