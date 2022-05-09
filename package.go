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

type sortVersion []string

func (sv sortVersion) Len() int {
	return len(sv)
}

func (sv sortVersion) Less(i, j int) bool {
	arr1, arr2 := strings.Split(sv[i], "."), strings.Split(sv[j], ".")
	if len(arr1) != len(arr2) {
		if len(arr1) > len(arr2) {
			arr2 = append(arr2, "0")
		} else {
			arr1 = append(arr1, "0")
		}
	}

	for i := range arr1 {
		bytes1, bytes2 := []byte(arr1[i]), []byte(arr2[i])
		if len(bytes1) > len(bytes2) {
			return true
		}
		if len(bytes1) < len(bytes2) {
			return false
		}

		for i2 := range bytes1 {
			if bytes1[i2] > bytes2[i2] {
				return true
			}
			if bytes1[i2] < bytes2[i2] {
				return false
			}
		}
	}

	return false
}

func (sv sortVersion) Swap(i, j int) {
	sv[i], sv[j] = sv[j], sv[i]
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

	if err = os.RemoveAll(svRoot); err != nil {
		return err
	}
	if err = os.Symlink(filepath.Join(svCache, p.Tag), svRoot); err != nil {
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
	return err
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