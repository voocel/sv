package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Version struct {
	Name     string
	Packages []*Package
}

type Package struct {
	Name      string
	Tag       string
	URL       string
	Kind      string
	OS        string
	Arch      string
	Size      string
	Checksum  string
	Algorithm string
}

func (p *Package) download() error {
	d := NewDownloader(runtime.NumCPU())
	if p.URL == "" || p.Name == "" {
		return errors.New("download URL is empty")
	}
	return d.Download(baseUrl+p.URL, p.Name)
}

func (p *Package) checkSum() (err error) {
	f, err := os.Open(svDownload + "/" + p.Name)
	if err != nil {
		return err
	}
	defer f.Close()

	var h hash.Hash
	switch p.Algorithm {
	case "SHA256":
		h = sha256.New()
	case "SHA1":
		h = sha1.New()
	default:
		return errors.New("unsupported checksum algorithm")
	}

	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	if p.Checksum != fmt.Sprintf("%x", h.Sum(nil)) {
		return errors.New("file checksum does not match the computed checksum")
	}
	return nil
}

func (p *Package) useCached() error {
	return execute(p.Tag)
}

func (p *Package) useDownloaded() error {
	if p.Checksum != "" {
		if err := p.checkSum(); err != nil {
			return err
		}
	}

	if err := Extract(svCache, filepath.Join(svDownload, p.Name)); err != nil {
		return err
	}
	GreenText("extract success")

	if err := os.Rename(filepath.Join(svCache, "go"), filepath.Join(svCache, p.Tag)); err != nil {
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
	if inCache(p.Tag) {
		return p.useCached()
	}
	if inDownload(p.Name) {
		return p.useDownloaded()
	}
	return errors.New("local does not exist")
}

func (p *Package) use() (err error) {
	if err := p.useLocal(); err != nil {
		return p.useRemote()
	}
	return
}

func (p *Package) install() error {
	if err := p.removeLocal(); err != nil {
		return err
	}
	return p.useRemote()
}

func (p *Package) remove() error {
	if err := p.removeLocal(); err != nil {
		return err
	}
	return os.RemoveAll(svRoot)
}

func (p *Package) removeLocal() (err error) {
	err = os.RemoveAll(filepath.Join(svCache, p.Tag))
	if err != nil {
		return
	}
	return os.RemoveAll(filepath.Join(svDownload, p.Name))
}

func (p *Package) getLocalVersion() (versions []string, err error) {
	folder := filepath.Join(svCache, "*")
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
	if err = os.RemoveAll(svRoot); err != nil {
		return err
	}
	if err = os.Symlink(filepath.Join(svCache, tag), svRoot); err != nil {
		return err
	}

	goBin := filepath.Join(svRoot, "bin", "go")
	cmd := exec.Command(goBin, "version")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	newPath := filepath.Join(svRoot, "bin")
	if p := os.Getenv("PATH"); p != "" {
		newPath += string(filepath.ListSeparator) + p
	}
	cmd.Env = setEnv(append(os.Environ(), "GOROOT="+svRoot, "PATH="+newPath))
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	return
}

// ExecCommand use shell /bin/bash -c to execute command
func ExecCommand(command string) (stdout, stderr string, err error) {
	var out bytes.Buffer
	var errout bytes.Buffer
	cmd := exec.Command("/bin/bash", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd")
	}
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		stderr = string(errout.Bytes())
	}
	stdout = string(out.Bytes())
	return
}

func inDownload(name string) bool {
	path := filepath.Join(svDownload, name)
	return Exists(path)
}

func inCache(tag string) bool {
	path := filepath.Join(svCache, tag)
	return Exists(path)
}

func setEnv(env []string) []string {
	out := make([]string, 0, len(env))
	saw := map[string]int{}
	for _, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 1 {
			out = append(out, kv)
			continue
		}
		k := kv[:eq]
		k = strings.ToLower(k)
		if dupIdx, isDup := saw[k]; isDup {
			out[dupIdx] = kv
		} else {
			saw[k] = len(out)
			out = append(out, kv)
		}
	}
	return out
}
