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

type app struct{}

func newApp() *app {
	return &app{}
}

func (a *app) Start() (err error) {
	err = a.selectVersion()
	if err != nil {
		return err
	}

	return
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
		targetPkg.Download()
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
