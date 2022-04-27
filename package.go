package main

import (
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

func (p *Package) Download() {
	d := NewDownloader(runtime.NumCPU())
	err := d.Download(baseUrl+p.URL, p.Name)
	if err != nil {
		panic(err)
	}
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
	if !Exists(filepath.Join(svRelease, p.Tag)) {
		if err = Extract(svRelease, srcPath); err != nil {
			return err
		}

		if err = os.Rename(filepath.Join(svRelease, "go"), filepath.Join(svRelease, p.Tag)); err != nil {
			return err
		}
	}

	if err = os.RemoveAll(svRoot); err != nil {
		return err
	}

	if err = os.Symlink(filepath.Join(svRelease, p.Tag), svRoot); err != nil {
		return err
	}

	goBin := filepath.Join(svRoot, "bin", "go")
	cmd := exec.Command(goBin, "env")
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
	return err
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

func (p *Package) uninstall() {

}

func (p *Package) use() {

}
