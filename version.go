package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

const (
	//OriginURL = "https://go.dev/dl"
	OriginURL = "https://studygolang.com/dl"
)

type Version struct {
	Url string
	Tag string
}

func NewVersion(url, tag string) *Version {
	return &Version{
		Url: url,
		Tag: tag,
	}
}

func (v *Version) download() error {
	rawUrl := fmt.Sprintf("%s/golang/%s.darwin-amd64.tar.gz", v.Url, v.Tag)
	uri, err := url.ParseRequestURI(rawUrl)
	if err != nil {
		return err
	}
	filename := path.Base(uri.Path)
	log.Println("[*] Filename " + filename)

	client := http.DefaultClient
	request, err := http.NewRequest("GET", rawUrl, nil)
	if err != nil {
		return err
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.ContentLength <= 0 {
		log.Println("[*] Destination server does not support breakpoint download.")
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	buf := make([]byte, 1024)

	written := 0
	go func() {
		for {
			nr, errR := reader.Read(buf)
			if nr > 0 {
				nw, errW := writer.Write(buf[0:nr])
				if nw > 0 {
					written += nw
				}
				if errW != nil {
					err = errW
					break
				}
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
			}
			if errR != nil {
				if err != io.EOF {
					err = errR
				}
				break
			}

		}
	}()

	spaceTime := time.Second * 1
	ticker := time.NewTicker(spaceTime)
	//last read data size
	lastWtn := 0
	stop := false

	for {
		select {
		case <-ticker.C:
			//file size read this time - data size read last time = speed
			speed := written - lastWtn
			log.Printf("[*] Speed %s / %s \n", bytesToSize(speed), spaceTime.String())
			if written-lastWtn == 0 {
				ticker.Stop()
				stop = true
				break
			}
			lastWtn = written
		}
		if stop {
			break
		}
	}
	return err
}

func bytesToSize(length int) string {
	var k = 1024
	var sizes = []string{"Bytes", "KB", "MB", "GB", "TB"}
	if length == 0 {
		return "0 Bytes"
	}
	i := math.Floor(math.Log(float64(length)) / math.Log(float64(k)))
	r := float64(length) / math.Pow(float64(k), i)
	return strconv.FormatFloat(r, 'f', 3, 64) + " " + sizes[int(i)]
}
