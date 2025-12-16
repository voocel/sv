package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func normalizeVersionTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return strings.Replace(tag, "v", "go", 1)
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

func generateDownloadURL(tag string) string {
	name := generateFileName(tag)
	if strings.HasPrefix(tag, "v") {
		name = strings.Replace(name, "v", "go", 1)
	}
	return fmt.Sprintf("/dl/%s", name)
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

// DefaultRetryConfig returns sensible defaults for retry behavior
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: cfg.DownloadRetry,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// retryFunc executes fn with exponential backoff retry
func retryFunc(fn func() error, maxRetries int) error {
	return retryWithConfig(fn, RetryConfig{
		MaxRetries: maxRetries,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	})
}

// retryWithConfig executes fn with configurable exponential backoff
func retryWithConfig(fn func() error, config RetryConfig) error {
	var lastErr error
	delay := config.BaseDelay

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err

			if attempt < config.MaxRetries-1 {
				waitTime := delay
				if config.Jitter {
					jitter := time.Duration(rand.Int63n(int64(delay)))
					waitTime = delay + jitter/2
				}

				Warnf("Attempt %d/%d failed: %v, retrying in %v...",
					attempt+1, config.MaxRetries, err, waitTime)
				time.Sleep(waitTime)

				delay = time.Duration(float64(delay) * config.Multiplier)
				if delay > config.MaxDelay {
					delay = config.MaxDelay
				}
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", config.MaxRetries, lastErr)
}

func Extract(dst, src string) error {
	PrintCyan("extracting...")
	switch {
	case strings.HasSuffix(src, ".tar.gz"), strings.HasSuffix(src, ".tgz"):
		return UnpackTar(dst, src)
	case strings.HasSuffix(src, ".zip"):
		return UnpackZip(dst, src)
	default:
		return fmt.Errorf("failed to extract %v, unhandled file type", src)
	}
}

// UnpackTar take a destination path and a reader
func UnpackTar(dst, src string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	if strings.HasSuffix(src, ".gz") {
		if fileReader, err = gzip.NewReader(file); err != nil {
			return err
		}
		defer fileReader.Close()
	}

	tr := tar.NewReader(fileReader)
	for {
		header, err := tr.Next()
		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		//fi := header.FileInfo()
		//mode := fi.Mode()
		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}
			outFile.Close()
		default:
			log.Fatalf("uknown type: %v in %s", header.Typeflag, header.Name)
		}
	}
}

// PackTar write each file found to the tar writer
func PackTar(src string, writers ...io.Writer) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files: %v", err.Error())
	}

	gzw := gzip.NewWriter(io.MultiWriter(writers...))
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		f.Close()

		return nil
	})
}

// PackZip a file or a directory
func PackZip(dst, src string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Method = zip.Deflate
		header.Name, err = filepath.Rel(filepath.Dir(src), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// UnpackZip will decompress a zip archived file
func UnpackZip(dst, src string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	err = os.MkdirAll(dst, 0755)
	if err != nil {
		return err
	}

	// closure to resolve file descriptors with defer close
	extract := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			err = os.MkdirAll(filepath.Dir(path), f.Mode())
			if err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extract(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// Exists reports whether the named file or directory exists.
func Exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func checkStringExistsFile(filename, value string) (bool, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == value {
			return true, nil
		}
	}

	return false, scanner.Err()
}

func versionCompare(version string) string {
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}

	// 如果版本字符串为空，返回空字符串
	if version == "" {
		return ""
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
		if j+1 < len(vo) && vo[j] == 1 && vo[j+1] == '0' {
			vo[j+1] = b
			continue
		}
		if vo[j]+1 > maxByte {
			// 不要panic，而是返回原始版本
			return version
		}
		vo = append(vo, b)
		vo[j]++
	}
	return string(vo)
}
