package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	barWidth    = 40
	refreshRate = 100 * time.Millisecond
	speedAlpha  = 0.3 // EMA smoothing factor for the speed readout
)

const (
	barFilled = "█"
	barEmpty  = "░"
	lineClear = "\r\x1b[K"
)

// Bar is a terminal progress bar, safe for concurrent writers.
type Bar struct {
	label string
	total int64

	current atomic.Int64
	done    atomic.Bool

	mu     sync.Mutex
	status string
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
		status: "Downloading",
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
	b.mu.Lock()
	b.status = green("Done")
	b.mu.Unlock()
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

func (b *Bar) render() {
	b.mu.Lock()
	status := b.status
	speed := b.speed
	b.mu.Unlock()

	current := b.current.Load()

	var line string
	if b.total > 0 {
		percent := float64(current) / float64(b.total) * 100
		filled := min(int(float64(barWidth)*float64(current)/float64(b.total)), barWidth)
		bar := "|" + strings.Repeat(barFilled, filled) + strings.Repeat(barEmpty, barWidth-filled) + "|"
		line = fmt.Sprintf("%s %s %3.0f%% %s %s/%s %s",
			status, b.label, percent, bar,
			green(formatBytes(current)), green(formatBytes(b.total)),
			yellow(formatBytes(int64(speed))+"/s"))
	} else {
		line = fmt.Sprintf("%s %s %s %s",
			status, b.label,
			green(formatBytes(current)),
			yellow(formatBytes(int64(speed))+"/s"))
	}

	fmt.Fprint(os.Stdout, lineClear+line)
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
