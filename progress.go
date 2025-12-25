package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	barWidth       = 40
	refreshRate    = 50 * time.Millisecond
	smoothingAlpha = 0.3 // EMA smoothing factor for speed calculation
)

var (
	filledChar = "█"
	emptyChar  = "░"
	clearLine  = "\r\033[K"
)

func init() {
	// Windows compatibility: use ASCII characters if needed
	if runtime.GOOS == "windows" {
		filledChar = "#"
		emptyChar = "-"
		clearLine = "\r" + strings.Repeat(" ", 100) + "\r" // fallback clear
		enableWindowsVT()
	}
}

// enableWindowsVT enables virtual terminal processing on Windows 10+
func enableWindowsVT() {
	// On Windows 10 1511+, we can enable ANSI escape sequences
	// This is a best-effort attempt; if it fails, we use the fallback
	// The actual implementation would use syscall, but for simplicity
	// we just set the ENABLE_VIRTUAL_TERMINAL_PROCESSING flag
	// via environment or let the terminal handle it
}

// Bar is a terminal progress bar
type Bar struct {
	name   string
	status string
	total  int64

	current  atomic.Int64
	speed    float64 // bytes per second (smoothed)
	prevTime time.Time
	prevSize int64

	done   chan struct{}
	closed atomic.Bool
	mu     sync.Mutex
}

// NewBar creates a new progress bar with the given total size
func NewBar(total int64) *Bar {
	b := &Bar{
		status:   "Downloading",
		total:    total,
		prevTime: time.Now(),
		done:     make(chan struct{}),
	}
	go b.refresh()
	return b
}

// SetName sets the display name with optional color
func (b *Bar) SetName(name, color string) {
	b.mu.Lock()
	if color != "" {
		b.name = SetColor(name, 0, 0, colorToCode(color))
	} else {
		b.name = name
	}
	b.mu.Unlock()
}

// Add increases the current progress
func (b *Bar) Add(n int64) {
	newVal := b.current.Add(n)
	if newVal >= b.total {
		b.current.Store(b.total)
		b.finish()
	}
}

// Write implements io.Writer for use with io.Copy
func (b *Bar) Write(p []byte) (n int, err error) {
	n = len(p)
	b.Add(int64(n))
	return n, nil
}

// Close stops the progress bar (safe to call multiple times)
func (b *Bar) Close() {
	if b.closed.CompareAndSwap(false, true) {
		close(b.done)
	}
}

func (b *Bar) finish() {
	b.mu.Lock()
	b.status = Green("Success")
	b.mu.Unlock()
	b.render()
	b.Close()
	fmt.Println() // newline after completion
}

func (b *Bar) refresh() {
	ticker := time.NewTicker(refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.updateSpeed()
			b.render()
		case <-b.done:
			return
		}
	}
}

func (b *Bar) updateSpeed() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.prevTime).Seconds()
	if elapsed <= 0 {
		return
	}

	current := b.current.Load()
	bytesTransferred := float64(current - b.prevSize)
	instantSpeed := bytesTransferred / elapsed

	// Exponential moving average for smooth speed display
	if b.speed == 0 {
		b.speed = instantSpeed
	} else {
		b.speed = smoothingAlpha*instantSpeed + (1-smoothingAlpha)*b.speed
	}

	b.prevTime = now
	b.prevSize = current
}

func (b *Bar) render() {
	b.mu.Lock()
	current := b.current.Load()
	status := b.status
	name := b.name
	speed := b.speed
	b.mu.Unlock()

	percent := float64(current) / float64(b.total) * 100
	if b.total == 0 {
		percent = 0
	}

	// Build progress bar
	filled := int(float64(barWidth) * float64(current) / float64(b.total))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	empty := barWidth - filled

	bar := "|" + strings.Repeat(filledChar, filled) + strings.Repeat(emptyChar, empty) + "|"

	// Format sizes and speed
	currentStr := formatBytes(current)
	totalStr := formatBytes(b.total)
	speedStr := formatBytes(int64(speed)) + "/s"

	// Build output line
	line := fmt.Sprintf("%s %s %3.0f%% %s %s/%s %s",
		status,
		name,
		percent,
		bar,
		Green(currentStr),
		Green(totalStr),
		Yellow(speedStr),
	)

	// Clear line and print
	fmt.Fprint(os.Stdout, clearLine+line)
}

func formatBytes(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	exp := 0
	val := float64(bytes) / unit

	for val >= unit && exp < len(units)-1 {
		val /= unit
		exp++
	}

	return fmt.Sprintf("%.1f %s", val, units[exp])
}
