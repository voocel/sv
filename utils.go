package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func normalizeVersionTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return "go" + tag[1:]
	}
	if !strings.HasPrefix(tag, "go") {
		return "go" + tag
	}
	return tag
}

func generateFileName(tag string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s.%s-%s%s", tag, runtime.GOOS, runtime.GOARCH, ext)
}

// retryFunc executes fn with exponential backoff retry
func retryFunc(fn func() error, maxRetries int) error {
	var lastErr error
	delay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				// Add jitter to prevent thundering herd
				jitter := time.Duration(rand.Int63n(int64(delay / 2)))
				waitTime := delay + jitter

				Warnf("Attempt %d/%d failed: %v, retrying in %v...",
					attempt+1, maxRetries, err, waitTime)
				time.Sleep(waitTime)

				// Exponential backoff with cap
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// Extract extracts an archive to the destination directory
func Extract(dst, src string) error {
	PrintCyan("extracting...")
	switch {
	case strings.HasSuffix(src, ".tar.gz"), strings.HasSuffix(src, ".tgz"):
		return unpackTar(dst, src)
	case strings.HasSuffix(src, ".zip"):
		return unpackZip(dst, src)
	default:
		return fmt.Errorf("unsupported archive format: %s", src)
	}
}

func unpackTar(dst, src string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header == nil {
			continue
		}

		target := filepath.Join(dst, header.Name)

		// Security: prevent path traversal attacks (zip slip)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := extractFile(target, tr, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Skip symlinks for security
			continue
		}
	}
}

func unpackZip(dst, src string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		target := filepath.Join(dst, f.Name)

		// Security: prevent path traversal attacks (zip slip)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		if err := extractFile(target, rc, f.Mode()); err != nil {
			rc.Close()
			return err
		}
		rc.Close()
	}

	return nil
}

func extractFile(target string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Exists reports whether the named file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// versionCompare returns a comparable string for version sorting
// Handles: go1.21, go1.21.5, go1.22rc1, go1.22beta1
func versionCompare(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "go")

	if version == "" {
		return ""
	}

	// Remove any suffix like " (2024-01-01)"
	if idx := strings.Index(version, " "); idx != -1 {
		version = version[:idx]
	}

	// Parse version parts
	var major, minor, patch int
	var preRelease string

	// Try to parse "1.21.5", "1.21", "1.22rc1", "1.22beta2"
	n, _ := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	if n < 3 {
		// Try "1.21" or "1.21rc1"
		n, _ = fmt.Sscanf(version, "%d.%d", &major, &minor)
		patch = 0
	}

	// Extract pre-release suffix (rc, beta, alpha)
	for _, pre := range []string{"rc", "beta", "alpha"} {
		if idx := strings.Index(strings.ToLower(version), pre); idx != -1 {
			preRelease = strings.ToLower(version[idx:])
			break
		}
	}

	// Build comparable string
	// Format: MAJOR.MINOR.PATCH.PRERELEASE
	// Pre-release: "~" for release (sorts last), "!alpha", "!beta", "!rc" (sort before release)
	result := fmt.Sprintf("%08d.%08d.%08d", major, minor, patch)

	if preRelease == "" {
		result += ".~" // release version sorts after pre-release
	} else {
		// Parse pre-release number
		var preNum int
		for _, pre := range []string{"rc", "beta", "alpha"} {
			if strings.HasPrefix(preRelease, pre) {
				fmt.Sscanf(preRelease[len(pre):], "%d", &preNum)
				// alpha < beta < rc < release
				// Use prefix to ensure correct ordering
				switch pre {
				case "alpha":
					result += fmt.Sprintf(".!a%08d", preNum)
				case "beta":
					result += fmt.Sprintf(".!b%08d", preNum)
				case "rc":
					result += fmt.Sprintf(".!r%08d", preNum)
				}
				break
			}
		}
	}

	return result
}
