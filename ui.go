package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

// Minimal ANSI palette — the whole of sv's styling.
func paint(code, s string) string { return "\x1b[" + code + "m" + s + "\x1b[0m" }

func red(s string) string    { return paint("31", s) }
func green(s string) string  { return paint("32", s) }
func yellow(s string) string { return paint("33", s) }
func cyan(s string) string   { return paint("36", s) }
func gray(s string) string   { return paint("90", s) }

func successf(format string, a ...any) { fmt.Println(green(fmt.Sprintf(format, a...))) }
func infof(format string, a ...any)    { fmt.Println(cyan(fmt.Sprintf(format, a...))) }
func warnf(format string, a ...any)    { fmt.Fprintln(os.Stderr, yellow(fmt.Sprintf(format, a...))) }
func errorf(format string, a ...any)   { fmt.Fprintln(os.Stderr, red(fmt.Sprintf(format, a...))) }

// interactive reports whether stdin supports the raw-mode TUI. It is false
// under Git Bash's MinTTY (which pipes stdin, hanging the TUI) and for
// piped input, where the prompts fall back to plain line-based reads.
func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

var stdin = bufio.NewReader(os.Stdin)

func readLine() (string, error) {
	s, err := stdin.ReadString('\n')
	if err != nil && s == "" {
		return "", err
	}
	return strings.TrimSpace(stripEscapes(s)), nil
}

// stripEscapes drops ANSI escape sequences and control bytes from line
// input — under MinTTY, arrow and function keys arrive as raw sequences
// (e.g. "\x1b[B") embedded in the line the user typed.
func stripEscapes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b { // ESC introduces a sequence
			if i+1 < len(s) && (s[i+1] == '[' || s[i+1] == 'O') {
				i += 2
				for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
					i++ // skip parameter bytes up to the final byte
				}
			}
			continue
		}
		if c >= 0x20 && c != 0x7f {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// selectPrompt asks the user to pick one of options; type "/" to filter.
func selectPrompt(title string, options []string) (string, error) {
	if !interactive() {
		return selectPlain(title, options)
	}
	var picked string
	sel := huh.NewSelect[string]().
		Title(title).
		Options(huh.NewOptions(options...)...).
		Value(&picked)
	// Cap the window for long lists; short lists render at natural height
	// (a fixed Height would draw the field border at full height).
	if len(options) > 15 {
		sel = sel.Height(15)
	}
	return picked, sel.Run()
}

// plainListMax caps the options printed by selectPlain; the full list is
// still selectable by typing the value itself.
const plainListMax = 20

func selectPlain(title string, options []string) (string, error) {
	fmt.Println(cyan(title))
	shown := options
	if len(shown) > plainListMax {
		shown = shown[:plainListMax]
	}
	for i, opt := range shown {
		fmt.Printf("%3d) %s\n", i+1, opt)
	}
	if rest := len(options) - len(shown); rest > 0 {
		fmt.Println(gray(fmt.Sprintf("     ... and %d more", rest)))
	}

	// Cap blank reads so a polluted stream (a dirty pty, `yes '' | sv`)
	// cannot flood the prompt forever.
	for blanks := 0; blanks < 20; {
		fmt.Print("Enter a number or version: ")
		s, err := readLine()
		if err != nil {
			return "", err
		}
		if s == "" {
			blanks++
			continue
		}
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(options) {
			return options[n-1], nil
		}
		for _, opt := range options {
			if opt == s || opt == "go"+s {
				return opt, nil
			}
		}
		errorf("invalid selection: %q", s)
	}
	return "", errors.New("no selection")
}

func confirmPrompt(title string) (bool, error) {
	if !interactive() {
		fmt.Print(cyan(title) + " [y/N] ")
		s, err := readLine()
		if err != nil {
			return false, err
		}
		return strings.EqualFold(s, "y") || strings.EqualFold(s, "yes"), nil
	}
	var ok bool
	err := huh.NewConfirm().Title(title).Value(&ok).Run()
	return ok, err
}

func inputPrompt(title string) (string, error) {
	if !interactive() {
		fmt.Print(cyan(title) + " ")
		return readLine()
	}
	var s string
	err := huh.NewInput().Title(title).Value(&s).Run()
	return s, err
}
