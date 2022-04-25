package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	f, err := os.Open(downloadsDir + "/" + p.Name)
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
	r, err := os.Open(downloadsDir + "/" + p.Name)
	if err != nil {
		panic(err)
	}

	err = Untar(versionsDir, r)
	if err != nil {
		return err
	}

	if err = os.Rename(filepath.Join(versionsDir, "go"), filepath.Join(versionsDir, p.Tag)); err!=nil{
		return err
	}

	if err = os.Symlink(filepath.Join(versionsDir, p.Tag), filepath.Join(filepath.Dir(goroot), p.Tag)); err != nil {
		return err
	}

	return err
}

func (p *Package) uninstall()  {

}

func (p *Package) use()  {

}

