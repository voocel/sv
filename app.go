package main

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

// "https://go.dev/dl"
const baseUrl = "https://studygolang.com"

type app struct {
	opts   *startOpts
	client *http.Client
}

type startOpts struct {
	cmd    string
	target string
	latest string
	remote bool
	force  bool
}

func newApp(opts *startOpts) *app {
	return &app{
		opts: opts,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (a *app) Start() (err error) {
	p := &Package{}
	switch a.opts.cmd {
	case "list":
		return a.list()
	case "use":
		p.Tag = a.opts.target
		p.Name = a.tagToName(p.Tag)
		if err := p.useLocal(); err != nil {
			var ok bool
			err = survey.AskOne(&survey.Confirm{
				Message: "Do you like to download and install from remote?",
			}, &ok)
			if ok {
				p.URL = a.tagToURL(p.Tag)
				return p.useRemote()
			}
		}
		return
	case "install":
		if a.opts.target == "" {
			return errors.New("tag is empty")
		}
		p.Tag = a.opts.target
		return p.install()
	case "uninstall":
		if a.opts.target == "" {
			return errors.New("tag is empty")
		}
		p.Tag = a.opts.target
		p.Name = a.tagToName(p.Tag)
		return p.remove()
	case "upgrade":
		u := NewUpgrade(a.opts.force)
		return u.checkUpgrade()
	}
	return
}

func (a *app) list() error {
	if a.opts.remote {
		resp, err := a.client.Get(baseUrl + "/dl")
		if err != nil {
			return err
		}
		respDate, err := a.client.Get(releaseUrl)
		if err != nil {
			return err
		}
		parser := NewParser(resp.Body)
		dateParser := NewDateParser(respDate.Body)

		releases := dateParser.findReleaseDate()
		parser.setReleases(releases)

		archive := parser.AllVersions()
		versions := make([]string, 0)
		for name := range archive {
			release := releases[name]
			versions = append(versions, fmt.Sprintf("%v (%v)", name, release))
		}

		target, err := a.selectVersions(versions)
		if err != nil {
			return err
		}
		targetPkg := a.getPackage(target, archive)
		if targetPkg == nil {
			return fmt.Errorf("not fount package on this version: %s", target)
		}

		return targetPkg.use()
	} else {
		pkg := &Package{}
		versions, err := pkg.getLocalVersion()
		if err != nil {
			return err
		}
		target, err := a.selectVersions(versions)
		if err != nil {
			return err
		}
		pkg.Tag = target
		return pkg.useLocal()
	}
}

func (a *app) selectVersions(versions []string) (target string, err error) {
	if len(versions) == 0 {
		return "", errors.New(Red("No available versions locally to select, you can use -r for get remote versions"))
	}

	surveyIcon := func() survey.AskOpt {
		return survey.WithIcons(func(icons *survey.IconSet) {
			icons.SelectFocus.Text = "→"
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versionCompare(versions[i]) > versionCompare(versions[j])
	})
	err = survey.AskOne(&survey.Select{
		Message: "Choose a version:",
		Help:    "Enter to install the selected version",
		Options: versions,
	}, &target, survey.WithValidator(survey.Required), surveyIcon())
	if i := strings.Index(target, "("); i != -1 {
		target = strings.TrimSpace(target[:i])
	}
	return
}

func (a *app) getPackage(target string, m map[string]*Version) *Package {
	archive, ok := m[target]
	if !ok {
		return nil
	}
	for _, v := range archive.Packages {
		filename := a.tagToName(target)
		if strings.HasPrefix(v.Name, filename) {
			return v
		}
	}
	return nil
}

func (a *app) tagToName(tag string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s.%s-%s%s", tag, runtime.GOOS, runtime.GOARCH, ext)
}

func (a *app) tagToURL(tag string) string {
	name := a.tagToName(tag)
	if strings.HasPrefix(tag, "v") {
		name = strings.Replace(name, "v", "go", 1)
	}
	return fmt.Sprintf("/dl/golang/%s", name)
}
