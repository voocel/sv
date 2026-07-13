package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// localVersions lists installed version tags, newest first.
func (a *App) localVersions() ([]string, error) {
	entries, err := os.ReadDir(a.paths.Cache)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tags []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "go") {
			tags = append(tags, e.Name())
		}
	}
	sortVersionsDesc(tags)
	return tags, nil
}

// currentVersion returns the active version tag, or "" if none.
func (a *App) currentVersion() string {
	link, err := os.Readlink(a.paths.Root)
	if err != nil {
		return ""
	}
	return filepath.Base(link)
}

func (a *App) versionPath(tag string) string {
	return filepath.Join(a.paths.Cache, tag)
}

func (a *App) isInstalled(tag string) bool {
	return fileExists(a.versionPath(tag))
}

// installRelease downloads, verifies and unpacks a release into the cache.
// An existing installation of the same version is replaced.
func (a *App) installRelease(ctx context.Context, release *GoRelease) error {
	file := release.archiveFile()
	if file == nil {
		return fmt.Errorf("no archive available for %s/%s in %s", runtime.GOOS, runtime.GOARCH, release.Version)
	}

	archive := filepath.Join(a.paths.Downloads, file.Filename)
	dl := newDownloader(a.client)
	label := cyan("sv[" + release.Version + "]")

	// Download and verify as one retryable unit: a corrupt archive is
	// deleted so the next attempt starts clean, while partial part files
	// survive network errors and resume.
	err := withRetry(a.cfg.DownloadRetry, func() error {
		if fileExists(archive) && verifyChecksum(archive, file.SHA256) == nil {
			return nil
		}
		if err := dl.fetch(ctx, file.url(), archive, label); err != nil {
			return err
		}
		if err := verifyChecksum(archive, file.SHA256); err != nil {
			os.Remove(archive)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := a.installFromArchive(release.Version, archive); err != nil {
		return err
	}
	successf("installed %s", release.Version)
	return nil
}

// installFromArchive unpacks a verified archive into the cache. It extracts
// to a temp dir first so the cache never holds a half-extracted version,
// then moves it into place and drops the archive.
func (a *App) installFromArchive(tag, archive string) error {
	tmp := filepath.Join(a.paths.Cache, ".extract-"+tag)
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	infof("extracting %s...", filepath.Base(archive))
	if err := extract(tmp, archive); err != nil {
		return err
	}

	target := a.versionPath(tag)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(tmp, "go"), target); err != nil {
		return err
	}

	os.Remove(archive)
	return nil
}

// localArchive returns the path of a previously downloaded archive for tag,
// or "" if none exists. It lets `sv use` work offline.
func (a *App) localArchive(tag string) string {
	archive := filepath.Join(a.paths.Downloads, archiveName(tag))
	if fileExists(archive) {
		return archive
	}
	return ""
}

// switchTo points ~/.sv/go at the given installed version.
func (a *App) switchTo(tag string) error {
	target := a.versionPath(tag)
	if !fileExists(target) {
		return fmt.Errorf("version %s is not installed", tag)
	}

	// Create the new link under a temp name first, then swap it in, so a
	// failed link never destroys the currently working one.
	tmp := a.paths.Root + ".new"
	os.RemoveAll(tmp)
	if err := link(target, tmp); err != nil {
		return err
	}
	if err := os.RemoveAll(a.paths.Root); err != nil {
		os.RemoveAll(tmp)
		return fmt.Errorf("remove current symlink: %w", err)
	}
	if err := os.Rename(tmp, a.paths.Root); err != nil {
		os.RemoveAll(tmp)
		return fmt.Errorf("activate new symlink: %w", err)
	}

	return a.printGoVersion()
}

// link points linkPath at target. On Windows, symlinks need Developer Mode
// or admin rights, so it falls back to a directory junction, which any user
// can create; os.Readlink resolves both the same way.
func link(target, linkPath string) error {
	symErr := os.Symlink(target, linkPath)
	if symErr == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("cmd", "/c", "mklink", "/J", linkPath, target).Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("create symlink: %w", symErr)
}

// printGoVersion runs `go version` with the freshly linked toolchain so the
// user sees the switch took effect.
func (a *App) printGoVersion() error {
	goBin := filepath.Join(a.paths.Root, "bin", "go")
	cmd := exec.Command(goBin, "version")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	path := filepath.Join(a.paths.Root, "bin")
	if p := os.Getenv("PATH"); p != "" {
		path += string(filepath.ListSeparator) + p
	}
	cmd.Env = dedupEnv(append(os.Environ(), "GOROOT="+a.paths.Root, "PATH="+path))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run go version: %w", err)
	}
	return nil
}

// removeVersion deletes an installed version; the active one is protected.
func (a *App) removeVersion(tag string) error {
	if tag == a.currentVersion() {
		return fmt.Errorf("version %s is currently active — switch to another version first", tag)
	}
	if !a.isInstalled(tag) {
		return fmt.Errorf("version %s is not installed", tag)
	}
	if err := os.RemoveAll(a.versionPath(tag)); err != nil {
		return fmt.Errorf("remove %s: %w", tag, err)
	}
	// Drop a leftover archive too, if any.
	os.Remove(filepath.Join(a.paths.Downloads, archiveName(tag)))
	return nil
}

// planPrune decides which versions to keep. versions must be sorted newest
// first; the current version is always kept and does not count against keep.
func planPrune(versions []string, current string, keep int, all bool) (kept, removed []string) {
	counted := 0
	for _, v := range versions {
		switch {
		case v == current:
			kept = append(kept, v)
		case all || counted >= keep:
			removed = append(removed, v)
		default:
			kept = append(kept, v)
			counted++
		}
	}
	return kept, removed
}

// verifyChecksum compares the file's SHA256 with want ("" skips the check).
func verifyChecksum(path, want string) error {
	if want == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != want {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

// dedupEnv keeps the last value per variable; keys fold case on Windows.
func dedupEnv(env []string) []string {
	out := make([]string, 0, len(env))
	seen := make(map[string]int, len(env))
	for _, kv := range env {
		if i := strings.Index(kv, "="); i > 0 {
			key := kv[:i]
			if runtime.GOOS == "windows" {
				key = strings.ToLower(key)
			}
			if at, ok := seen[key]; ok {
				out[at] = kv
				continue
			}
			seen[key] = len(out)
		}
		out = append(out, kv)
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
