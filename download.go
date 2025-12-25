package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Downloader struct {
	concurrency int
	tag         string
	bar         *Bar
	client      *http.Client
}

func NewDownloader(concurrency int, tag string) *Downloader {
	return &Downloader{
		tag:         tag,
		concurrency: concurrency,
		client:      &http.Client{},
	}
}

func (d *Downloader) Download(strURL, filename string) error {
	if strURL == "" {
		return NewError("download URL is empty")
	}

	if filename == "" {
		filename = filepath.Base(strURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, strURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d for %s", resp.StatusCode, strURL)
	}

	if resp.Header.Get("Accept-Ranges") == "bytes" && resp.ContentLength > 0 {
		return d.multiDownload(strURL, filename, resp.ContentLength)
	}

	return d.singleDownload(strURL, filename)
}

func (d *Downloader) multiDownload(strURL, filename string, contentLen int64) error {
	partSize := contentLen / int64(d.concurrency)
	partDir := d.getPartDir(filename)
	if err := os.MkdirAll(partDir, 0755); err != nil {
		return err
	}

	var downloaded int64
	for i := 0; i < d.concurrency; i++ {
		if info, err := os.Stat(d.getPartFilename(filename, i)); err == nil {
			downloaded += info.Size()
		}
	}

	d.bar = NewBar(contentLen)
	d.bar.SetName("sv["+d.tag+"]", "pink")
	if downloaded > 0 {
		d.bar.Add(downloaded)
	}

	var (
		wg    sync.WaitGroup
		errCh = make(chan error, d.concurrency)
	)

	var rangeStart int64
	for i := 0; i < d.concurrency; i++ {
		wg.Add(1)
		rangeEnd := rangeStart + partSize - 1
		if i == d.concurrency-1 {
			rangeEnd = contentLen - 1
		}

		go func(i int, start, end int64) {
			defer wg.Done()
			if err := d.downloadPartial(strURL, filename, start, end, i); err != nil {
				select {
				case errCh <- fmt.Errorf("part %d: %w", i, err):
				default:
				}
			}
		}(i, rangeStart, rangeEnd)

		rangeStart = rangeEnd + 1
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return err
	}

	if err := d.merge(filename); err != nil {
		return err
	}

	os.RemoveAll(partDir)
	return nil
}

func (d *Downloader) singleDownload(strURL, filename string) error {
	resp, err := d.client.Get(strURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d for %s", resp.StatusCode, strURL)
	}

	d.bar = NewBar(resp.ContentLength)
	d.bar.SetName("sv["+d.tag+"]", "pink")

	f, err := os.OpenFile(filepath.Join(paths.Download, filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (d *Downloader) downloadPartial(strURL, filename string, rangeStart, rangeEnd int64, i int) error {
	partFile := d.getPartFilename(filename, i)
	expectedSize := rangeEnd - rangeStart + 1

	var downloaded int64
	if info, err := os.Stat(partFile); err == nil {
		downloaded = info.Size()
	}

	if downloaded >= expectedSize {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, strURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart+downloaded, rangeEnd))

	// dynamically calculate the timeout period based on the remaining size
	remaining := expectedSize - downloaded
	timeout := time.Duration(remaining/1024/10+30) * time.Second
	if timeout > 10*time.Minute {
		timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	f, err := os.OpenFile(partFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	return err
}

func (d *Downloader) merge(filename string) error {
	dstFile, err := os.OpenFile(filepath.Join(paths.Download, filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	for i := 0; i < d.concurrency; i++ {
		partFileName := d.getPartFilename(filename, i)
		partFile, err := os.Open(partFileName)
		if err != nil {
			return fmt.Errorf("failed to open part %d: %w", i, err)
		}
		_, err = io.Copy(dstFile, partFile)
		partFile.Close()
		if err != nil {
			return fmt.Errorf("failed to merge part %d: %w", i, err)
		}
	}

	return nil
}

func (d *Downloader) getPartDir(filename string) string {
	return filepath.Join(paths.Download, strings.SplitN(filepath.Base(filename), ".", 2)[0])
}

func (d *Downloader) getPartFilename(filename string, partNum int) string {
	return filepath.Join(d.getPartDir(filename), fmt.Sprintf("%s-%d", filename, partNum))
}
