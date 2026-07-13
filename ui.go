package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

// Minimal ANSI palette — the whole of sv's styling.
func paint(code, s string) string { return "\x1b[" + code + "m" + s + "\x1b[0m" }

func red(s string) string    { return paint("31", s) }
func green(s string) string  { return paint("32", s) }
func yellow(s string) string { return paint("33", s) }
func cyan(s string) string   { return paint("36", s) }

func successf(format string, a ...any) { fmt.Println(green(fmt.Sprintf(format, a...))) }
func infof(format string, a ...any)    { fmt.Println(cyan(fmt.Sprintf(format, a...))) }
func warnf(format string, a ...any)    { fmt.Fprintln(os.Stderr, yellow(fmt.Sprintf(format, a...))) }
func errorf(format string, a ...any)   { fmt.Fprintln(os.Stderr, red(fmt.Sprintf(format, a...))) }

// selectPrompt asks the user to pick one of options; type "/" to filter.
func selectPrompt(title string, options []string) (string, error) {
	var picked string
	err := huh.NewSelect[string]().
		Title(title).
		Options(huh.NewOptions(options...)...).
		Height(15).
		Value(&picked).
		Run()
	return picked, err
}

func confirmPrompt(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().Title(title).Value(&ok).Run()
	return ok, err
}

func inputPrompt(title string) (string, error) {
	var s string
	err := huh.NewInput().Title(title).Value(&s).Run()
	return s, err
}
