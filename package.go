package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Package struct {
	Name      string
	Tag       string
	URL       string
	Kind      string
	OS        string
	Arch      string
	Checksum  string // SHA256
	Algorithm string
}

func (p *Package) download() error {
	if p.URL == "" || p.Name == "" {
		return ErrURLEmpty()
	}

	d := NewDownloader(runtime.NumCPU(), p.Tag)

	return retryFunc(func() error {
		return d.Download(p.URL, p.Name)
	}, cfg.DownloadRetry)
}

func (p *Package) verifyChecksum() error {
	if p.Checksum == "" {
		return nil
	}

	filePath := filepath.Join(paths.Download, p.Name)
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to read file for checksum: %w", err)
	}

	computed := fmt.Sprintf("%x", h.Sum(nil))
	if p.Checksum != computed {
		return ErrChecksumMismatch()
	}
	return nil
}

func (p *Package) useCached() error {
	tag := normalizeVersionTag(p.Tag)
	return execute(tag)
}

func (p *Package) useDownloaded() error {
	if err := p.verifyChecksum(); err != nil {
		return err
	}

	if err := Extract(paths.Cache, filepath.Join(paths.Download, p.Name)); err != nil {
		return err
	}
	PrintGreen("extract success")

	normalizedTag := normalizeVersionTag(p.Tag)
	if err := os.Rename(filepath.Join(paths.Cache, "go"), filepath.Join(paths.Cache, normalizedTag)); err != nil {
		return err
	}

	return p.useCached()
}

func (p *Package) useRemote() error {
	if err := p.download(); err != nil {
		return err
	}
	return p.useDownloaded()
}

// useLocal contain cached and downloaded
func (p *Package) useLocal() error {
	normalizedTag := normalizeVersionTag(p.Tag)
	if inCache(normalizedTag) {
		return p.useCached()
	}
	if inDownload(p.Name) {
		return p.useDownloaded()
	}
	return ErrLocalNotExist()
}

func (p *Package) use() (err error) {
	if err := p.useLocal(); err != nil {
		return p.useRemote()
	}
	return
}

func (p *Package) install() error {
	if err := p.download(); err != nil {
		return err
	}

	tag := normalizeVersionTag(p.Tag)
	os.RemoveAll(filepath.Join(paths.Cache, tag))

	return p.useDownloaded()
}

func (p *Package) remove() error {
	return p.removeLocal()
}

func (p *Package) removeLocal() error {
	tag := normalizeVersionTag(p.Tag)

	linkPath, err := os.Readlink(paths.Root)
	if err == nil && filepath.Base(linkPath) == tag {
		return ErrVersionInUse(tag)
	}

	if err := os.RemoveAll(filepath.Join(paths.Cache, tag)); err != nil {
		return fmt.Errorf("failed to remove cached version: %w", err)
	}

	os.RemoveAll(filepath.Join(paths.Download, p.Name))

	return nil
}

func (p *Package) getLocalVersion() (versions []string, err error) {
	folder := filepath.Join(paths.Cache, "*")
	versions, err = filepath.Glob(folder)
	if err != nil {
		return
	}
	for i, v := range versions {
		versions[i] = filepath.Base(v)
	}
	return
}

func execute(tag string) (err error) {
	if err = os.RemoveAll(paths.Root); err != nil {
		return fmt.Errorf("failed to remove existing Go installation: %w", err)
	}
	if err = os.Symlink(filepath.Join(paths.Cache, tag), paths.Root); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	goBin := filepath.Join(paths.Root, "bin", "go")
	cmd := exec.Command(goBin, "version")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	newPath := filepath.Join(paths.Root, "bin")
	if p := os.Getenv("PATH"); p != "" {
		newPath += string(filepath.ListSeparator) + p
	}
	cmd.Env = dedupEnv(append(os.Environ(), "GOROOT="+paths.Root, "PATH="+newPath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute go version: %w", err)
	}
	return nil
}

func inDownload(name string) bool {
	return Exists(filepath.Join(paths.Download, name))
}

func inCache(tag string) bool {
	return Exists(filepath.Join(paths.Cache, tag))
}

// dedupEnv removes duplicate environment variables, keeping the last value
func dedupEnv(env []string) []string {
	out := make([]string, 0, len(env))
	seen := make(map[string]int, len(env))
	for _, kv := range env {
		if idx := strings.Index(kv, "="); idx > 0 {
			key := strings.ToLower(kv[:idx])
			if i, ok := seen[key]; ok {
				out[i] = kv
				continue
			}
			seen[key] = len(out)
		}
		out = append(out, kv)
	}
	return out
}

// getCurrentVersion returns the currently active Go version
func getCurrentVersion() string {
	linkPath, err := os.Readlink(paths.Root)
	if err != nil {
		return ""
	}
	return filepath.Base(linkPath)
}
