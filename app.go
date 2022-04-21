package main

import (
	"net/http"
	"runtime"

	"github.com/AlecAivazis/survey/v2"
)

// "https://go.dev/dl"
const baseUrl = "https://studygolang.com/dl"

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
	resp, err := http.Get(baseUrl)
	if err != nil {
		return
	}

	parser := NewParser(resp.Body)
	versions := parser.ArchivedVersions()
	err = survey.AskOne(&survey.Select{
		Message: "Choose a version:",
		Options: versions,
	}, &target, survey.WithValidator(survey.Required), surveyIcons())
	if err != nil {
		return err
	}

	d := NewDownloader(runtime.NumCPU())
	err = d.Download("https://studygolang.com/dl/golang/go1.18.1.darwin-amd64.tar.gz", "go1.18.1.darwin-amd64.tar.gz")
	if err != nil {
		return err
	}

	return
}

func surveyIcons() survey.AskOpt {
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = "→"
	})
}
