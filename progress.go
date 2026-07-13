package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

const (
	barMaxWidth = 30
	barMinWidth = 8             // below this the bar is dropped, keeping numbers only
	byteWidth   = 9             // len("1023.9 KB") — widest formatBytes output
	speedWidth  = byteWidth + 2 // widest speed readout, e.g. "1023.9 KB/s"
	sepWidth    = 4             // " · " — the dot may render double-wide in CJK fonts
	refreshRate = 100 * time.Millisecond
	speedAlpha  = 0.3 // EMA smoothing factor for the speed readout

	lineClear = "\r\x1b[K"
	barFill   = "\x1b[46m"  // cyan while downloading
	barDone   = "\x1b[42m"  // green once finished
	barTrack  = "\x1b[100m" // dark gray remainder
)

// sep visually separates the stat groups on the progress line.
var sep = " " + gray("·") + " "

// renderBar draws the bar as background-colored spaces: block glyphs like
// █/░ are East-Asian ambiguous-width and render double-wide in CJK fonts,
// which wraps the line and breaks the \r-based in-place refresh.
func renderBar(width, filled int, fill string) string {
	return fill + strings.Repeat(" ", filled) +
		barTrack + strings.Repeat(" ", width-filled) + "\x1b[0m"
}

// termWidth reports the terminal width, defaulting to 80 when unknown.
func termWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

// Bar is a terminal progress bar, safe for concurrent writers.
type Bar struct {
	label string
	total int64

	current atomic.Int64
	done    atomic.Bool

	mu     sync.Mutex
	speed  float64
	prevAt time.Time
	prevN  int64

	stop chan struct{}
	once sync.Once
}

func newBar(label string, total int64) *Bar {
	b := &Bar{
		label:  label,
		total:  total,
		prevAt: time.Now(),
		stop:   make(chan struct{}),
	}
	go b.loop()
	return b
}

// Add advances the bar; reaching the total finishes it automatically.
func (b *Bar) Add(n int64) {
	if b.current.Add(n) >= b.total && b.total > 0 {
		b.Finish()
	}
}

// Write implements io.Writer for use with io.Copy / io.MultiWriter.
func (b *Bar) Write(p []byte) (int, error) {
	b.Add(int64(len(p)))
	return len(p), nil
}

// Finish renders the final success state. Safe to call multiple times.
func (b *Bar) Finish() {
	if !b.done.CompareAndSwap(false, true) {
		return
	}
	b.render()
	fmt.Println()
	b.once.Do(func() { close(b.stop) })
}

// Close stops rendering without marking success (e.g. on error).
func (b *Bar) Close() {
	b.once.Do(func() {
		close(b.stop)
		if !b.done.Load() {
			fmt.Println()
		}
	})
}

func (b *Bar) loop() {
	ticker := time.NewTicker(refreshRate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.updateSpeed()
			b.render()
		case <-b.stop:
			return
		}
	}
}

func (b *Bar) updateSpeed() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.prevAt).Seconds()
	if elapsed <= 0 {
		return
	}

	current := b.current.Load()
	instant := float64(current-b.prevN) / elapsed
	if b.speed == 0 {
		b.speed = instant
	} else {
		b.speed = speedAlpha*instant + (1-speedAlpha)*b.speed
	}
	b.prevAt = now
	b.prevN = current
}

// render draws `label  ▓▓▓▓░░░  45% · 32.1/71.4 MB · 10.8 MB/s`. The fixed
// width elements sit at the front and the variable-width numbers at the end
// of the line, so nothing jumps as they grow — only the right edge moves.
// The bar is sized against the widest possible tail so the line never
// wraps; wrapping would break the \r in-place refresh. Label and bar are
// cyan while downloading and turn green when done.
func (b *Bar) render() {
	b.mu.Lock()
	speed := b.speed
	b.mu.Unlock()

	current := b.current.Load()

	label, fill := cyan(b.label), barFill
	if b.done.Load() {
		label, fill = green(b.label), barDone
	}
	speedStr := yellow(formatBytes(int64(speed)) + "/s")

	if b.total <= 0 {
		fmt.Fprint(os.Stdout, lineClear+label+"  "+formatBytes(current)+sep+speedStr)
		return
	}

	percent := fmt.Sprintf("%3.0f%%", float64(current)/float64(b.total)*100)

	line := label + "  "
	tailMax := len(percent) + sepWidth + byteWidth + 1 + len(formatBytes(b.total)) + sepWidth + speedWidth
	// -6: the label's and the bar's trailing spaces plus a 2-column margin.
	if barW := min(termWidth()-len(b.label)-tailMax-6, barMaxWidth); barW >= barMinWidth {
		filled := min(int(float64(barW)*float64(current)/float64(b.total)), barW)
		line += renderBar(barW, filled, fill) + "  "
	}
	line += percent + sep + formatPair(current, b.total) + sep + speedStr

	fmt.Fprint(os.Stdout, lineClear+line)
}

// formatPair renders "current/total", dropping the unit from the current
// side when both share it: "32.1/71.4 MB" rather than "32.1 MB/71.4 MB".
func formatPair(current, total int64) string {
	c, t := formatBytes(current), formatBytes(total)
	if i := strings.LastIndexByte(c, ' '); i >= 0 && strings.HasSuffix(t, c[i:]) {
		c = c[:i]
	}
	return c + "/" + t
}

func formatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	val := float64(n) / unit
	exp := 0
	for val >= unit && exp < len(units)-1 {
		val /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", val, units[exp])
}
