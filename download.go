package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	probeTimeout    = 30 * time.Second
	downloadTimeout = 30 * time.Minute
	multipartMin    = 4 << 20 // below this size a single stream is cheaper
)

// downloader fetches a URL to a local file with concurrent ranged requests,
// resume support and a progress bar. It is a single-attempt worker; retry
// policy lives with the caller.
type downloader struct {
	client *http.Client
	parts  int
}

func newDownloader(client *http.Client) *downloader {
	return &downloader{client: client, parts: runtime.NumCPU()}
}

// fetch downloads url into dest. label is the progress bar caption.
func (d *downloader) fetch(ctx context.Context, url, dest, label string) error {
	size, ranged, err := d.probe(ctx, url)
	if err != nil {
		return err
	}
	if ranged && size >= multipartMin && d.parts > 1 {
		return d.fetchParts(ctx, url, dest, size, label)
	}
	return d.fetchSingle(ctx, url, dest, size, label)
}

// probe asks the server for the file size and range support.
func (d *downloader) probe(ctx context.Context, url string) (size int64, ranged bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, false, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("probe %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("server returned status %d for %s", resp.StatusCode, url)
	}
	return resp.ContentLength, resp.Header.Get("Accept-Ranges") == "bytes", nil
}

type partSpan struct {
	file       string
	start, end int64
}

func (d *downloader) fetchParts(ctx context.Context, url, dest string, size int64, label string) error {
	partDir := dest + ".parts"
	if err := os.MkdirAll(partDir, 0o755); err != nil {
		return err
	}

	// Split into spans and count bytes already present from a previous run.
	partSize := size / int64(d.parts)
	spans := make([]partSpan, d.parts)
	var resumed int64
	for i := range spans {
		start := int64(i) * partSize
		end := start + partSize - 1
		if i == d.parts-1 {
			end = size - 1
		}
		file := filepath.Join(partDir, fmt.Sprintf("part-%d", i))
		if info, err := os.Stat(file); err == nil {
			resumed += min(info.Size(), end-start+1)
		}
		spans[i] = partSpan{file: file, start: start, end: end}
	}

	bar := newBar(label, size)
	defer bar.Close()
	if resumed > 0 {
		bar.Add(resumed)
	}

	g, gctx := errgroup.WithContext(ctx)
	for i, sp := range spans {
		g.Go(func() error {
			if err := d.fetchPart(gctx, url, sp, bar); err != nil {
				return fmt.Errorf("part %d: %w", i, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	if err := mergeParts(dest, spans); err != nil {
		return err
	}
	bar.Finish()
	return os.RemoveAll(partDir)
}

func (d *downloader) fetchPart(ctx context.Context, url string, sp partSpan, bar *Bar) error {
	want := sp.end - sp.start + 1
	var have int64
	if info, err := os.Stat(sp.file); err == nil {
		have = info.Size()
	}
	if have >= want {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", sp.start+have, sp.end))

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("expected partial content, got status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(sp.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	return err
}

func (d *downloader) fetchSingle(ctx context.Context, url, dest string, size int64, label string) error {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d for %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := newBar(label, size)
	defer bar.Close()

	if _, err := io.Copy(io.MultiWriter(f, bar), resp.Body); err != nil {
		return err
	}
	bar.Finish()
	return nil
}

func mergeParts(dest string, spans []partSpan) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, sp := range spans {
		part, err := os.Open(sp.file)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, part)
		part.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// withRetry runs fn up to attempts times with jittered exponential backoff.
func withRetry(attempts int, fn func() error) error {
	attempts = max(attempts, 1)
	delay := time.Second

	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			wait := delay + rand.N(delay/2)
			warnf("attempt %d/%d failed: %v — retrying in %s", i+1, attempts, err, wait.Round(time.Millisecond))
			time.Sleep(wait)
			delay = min(delay*2, 10*time.Second)
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", attempts, err)
}
