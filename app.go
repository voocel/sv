package main

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"net/http"
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
	resp, err := http.Get("https://go.dev/dl")
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
	fmt.Println(target)

	v := NewVersion(baseUrl, target)
	err = v.download()
	if err != nil {
		return
	}
	return
}

func surveyIcons() survey.AskOpt {
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = "→"
	})
}
