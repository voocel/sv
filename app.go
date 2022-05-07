package main

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// "https://go.dev/dl"
const baseUrl = "https://studygolang.com"

type app struct {
	opts *startOpts
}

type startOpts struct {
	cmd string
	target string
}

func newApp(opts *startOpts) *app {
	return &app{
		opts: opts,
	}
}

func (a *app) Start() (err error) {
	if a.opts.cmd == "" {
		return a.selectVersion()
	}

	p := &Package{}
	switch a.opts.cmd {
	case "use":
		p.Tag = a.opts.target
		p.use()
	}

	return
}

func (a *app) use() {

}

func (a *app) selectVersion() (err error) {
	var target string
	resp, err := http.Get(baseUrl + "/dl")
	if err != nil {
		return
	}

	parser := NewParser(resp.Body)
	archive := parser.AllVersions()
	versions := make([]string, 0)
	for name := range archive {
		versions = append(versions, name)
	}
	sort.Sort(sortVersion(versions))

	err = survey.AskOne(&survey.Select{
		Message: "Choose a version:",
		Options: versions,
	}, &target, survey.WithValidator(survey.Required), surveyIcons())
	if err != nil {
		return err
	}
	// target = "go1.18"
	targetPkg := a.getPackage(target, archive)
	if targetPkg == nil {
		return fmt.Errorf("not fount package: %s", target)
	}

	if Exists(svDownload + "/" + targetPkg.Name) {
		//if err := targetPkg.CheckSum(); err != nil {
		//	return err
		//}
		if err = targetPkg.install(); err != nil {
			return err
		}
	} else {
		if err = targetPkg.Download(); err != nil {
			return err
		}
		if err = targetPkg.install(); err != nil {
			return err
		}
	}

	return
}

func (a *app) getPackage(target string, m map[string]*Version) *Package {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	for _, v := range m[target].Packages {
		filename := fmt.Sprintf("%s.%s-%s%s", target, runtime.GOOS, runtime.GOARCH, ext)
		if strings.HasPrefix(v.Name, filename) {
			return v
		}
	}
	return nil
}

func surveyIcons() survey.AskOpt {
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = "→"
	})
}
