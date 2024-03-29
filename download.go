package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Downloader struct {
	concurrency int
	resume      bool
	tag         string
	bar         *Bar
	client      *http.Client
}

// PartMeta the file meta info
type PartMeta struct {
	Index float32
	Start int64
	End   int64
	Cur   int64
}

func NewDownloader(concurrency int, tag string) *Downloader {
	return &Downloader{
		tag:         tag,
		concurrency: concurrency,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (d *Downloader) Download(strURL, filename string) error {
	if filename == "" {
		filename = filepath.Base(strURL)
	}

	resp, err := d.client.Head(strURL)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		return d.multiDownload(strURL, filename, resp.ContentLength)
	}

	return d.singleDownload(strURL, filename)
}

func (d *Downloader) multiDownload(strURL, filename string, contentLen int64) error {
	d.bar = NewBar(contentLen)
	d.bar.SetName("sv["+d.tag+"]", "pink")

	partSize := int(contentLen) / d.concurrency
	partDir := d.getPartDir(filename)
	err := os.MkdirAll(partDir, 0777)
	if err != nil {
		return err
	}
	defer os.RemoveAll(partDir)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)
	rangeStart := 0
	for i := 0; i < d.concurrency; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()
			rangeEnd := rangeStart + partSize
			// in the last part, the total length cannot exceed ContentLength
			if i == d.concurrency-1 {
				rangeEnd = int(contentLen)
			}

			downloaded := 0
			if d.resume {
				partFileName := d.getPartFilename(filename, i)
				content, err := ioutil.ReadFile(partFileName)
				if err == nil {
					downloaded = len(content)
				}
			}

			err := d.downloadPartial(strURL, filename, rangeStart+downloaded, rangeEnd, i)
			if err != nil {
				PrintRed(fmt.Sprintf("downloadPartial err: %v", err.Error()))
			}
		}(i, rangeStart)

		rangeStart += partSize + 1
	}
	wg.Wait()

	return d.merge(filename)
}

func (d *Downloader) singleDownload(strURL, filename string) error {
	resp, err := d.client.Get(strURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d.bar = NewBar(resp.ContentLength)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	//wt := bufio.NewWriter(f)
	defer f.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	//wt.Flush()
	return err
}

func (d *Downloader) downloadPartial(strURL, filename string, rangeStart, rangeEnd, i int) (err error) {
	if rangeStart >= rangeEnd {
		return
	}

	req, err := http.NewRequest("GET", strURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if d.resume {
		flags |= os.O_APPEND
	}

	partFile, err := os.OpenFile(d.getPartFilename(filename, i), flags, 0666)
	if err != nil {
		return
	}
	defer partFile.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(partFile, d.bar), resp.Body, buf)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return
}

func (d *Downloader) merge(filename string) error {
	dstFile, err := os.OpenFile(SVDownload+"/"+filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	for i := 0; i < d.concurrency; i++ {
		partFileName := d.getPartFilename(filename, i)
		partFile, err := os.Open(partFileName)
		if err != nil {
			return err
		}
		io.Copy(dstFile, partFile)
		partFile.Close()
		os.Remove(partFileName)
	}

	return nil
}

func (d *Downloader) getPartDir(filename string) string {
	return SVDownload + "/" + strings.SplitN(filepath.Base(filename), ".", 2)[0]
}

func (d *Downloader) getPartFilename(filename string, partNum int) string {
	partDir := d.getPartDir(filename)
	return fmt.Sprintf("%s/%s-%d", partDir, filename, partNum)
}
