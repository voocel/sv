//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// Enable ANSI escape sequence processing in the Windows console so colors
// and the progress bar render correctly. Supported on every Windows version
// Go itself supports (Windows 10+).
func init() {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		var mode uint32
		h := windows.Handle(f.Fd())
		if windows.GetConsoleMode(h, &mode) == nil {
			windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
		}
	}
}
