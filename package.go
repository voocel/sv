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

func (p *Package) Download() error {
	d := NewDownloader(runtime.NumCPU())
	return d.Download(baseUrl+p.URL, p.Name)
}

func (p *Package) CheckSum() (err error) {
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

func (p *Package) install() (err error) {
	srcPath := svDownload + "/" + p.Name
	if !Exists(filepath.Join(svCache, p.Tag)) {
		if err = Extract(svCache, srcPath); err != nil {
			return err
		}
		if err = os.Rename(filepath.Join(svCache, "go"), filepath.Join(svCache, p.Tag)); err != nil {
			return err
		}
	}

	return execute(p.Tag)
}

func (p *Package) uninstall() error {
	return os.RemoveAll(filepath.Join(svRoot))
}

func (p *Package) use() (err error) {
	if inCache(p.Tag) {
		return execute(p.Tag)
	}
	if inDownload(p.Tag) {
		if err = Extract(svCache, filepath.Join(svDownload, p.Name)); err != nil {
			return err
		}
		if err = os.Rename(filepath.Join(svCache, "go"), filepath.Join(svCache, p.Tag)); err != nil {
			return err
		}
		return execute(p.Tag)
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

func inDownload(tag string) bool {
	path := filepath.Join(svDownload, tag)
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