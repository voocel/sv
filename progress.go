package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultBarWidth = 50
	updateInterval  = time.Second / 10
	bytesPerKB      = 1024
)

// Bar is a progress bar
type Bar struct {
	StartDelimiter string // StartDelimiter for the bar ("|")
	EndDelimiter   string // EndDelimiter for the bar ("|")
	Filled         string // Filled section representation ("█ ■")
	Empty          string // Empty section representation ("░ □")
	Width          int    // Width of the bar
	Name           string // Name of the bar
	Status         string // Status of the bar

	text    string
	rate    string
	prev    int64
	current int64
	total   int64
	tmpl    *template.Template
	done    chan struct{}
	closed  atomic.Bool
	mu      sync.Mutex
}

// NewBar return a new bar with the given total
func NewBar(total int64) *Bar {
	b := &Bar{
		StartDelimiter: "|",
		EndDelimiter:   "|",
		Filled:         "█",
		Empty:          "░",
		Width:          defaultBarWidth,
		Status:         "Downloading",
		total:          total,
		done:           make(chan struct{}),
	}
	go b.listenRate()
	fmt.Print("\r\n")
	b.template(`{{.Status}} {{.Name}} {{.Percent | printf "%3.0f"}}% {{.Bar}} {{.Total}} {{.Rate}} {{.Text}}`)

	return b
}

// listenRate start listen the speed
func (b *Bar) listenRate() {
	tick := time.NewTicker(updateInterval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			r := b.current - b.prev
			b.rate = "[" + b.bytesToSize(r*10) + "/s]"
			b.rate = SetColor(b.rate, 0, 0, yellow)
			b.prev = b.current
		case <-b.done:
			fmt.Print("\r\n")
			return
		}
	}
}

// template for rendering
func (b *Bar) template(s string) {
	t, err := template.New("").Parse(s)
	if err != nil {
		Errorf("Failed to parse progress bar template: %v", err)
		t, _ = template.New("").Parse("{{.Status}} {{.Percent | printf \"%.0f\"}}%")
	}
	b.tmpl = t
}

// SetText set the text value
func (b *Bar) SetText(s string, color ...string) {
	b.text = s
	if len(color) > 0 {
		b.text = SetColor(b.text, 0, 0, colorToCode(color[0]))
	}
}

// SetStatus set the status value
func (b *Bar) SetStatus(s string, color ...string) {
	b.Status = s
	if len(color) > 0 {
		b.Status = SetColor(b.Status, 0, 0, colorToCode(color[0]))
	}
}

// SetName set the name value
func (b *Bar) SetName(s string, color ...string) {
	b.Name = s
	if len(color) > 0 {
		b.Name = SetColor(b.Name, 0, 0, colorToCode(color[0]))
	}
}

// SetFilled set the filled value
func (b *Bar) SetFilled(s string, color ...string) {
	b.Filled = s
	if len(color) > 0 {
		b.Filled = SetColor(b.Filled, 0, 0, colorToCode(color[0]))
	}
}

// SetEmpty set the empty value
func (b *Bar) SetEmpty(s string, color ...string) {
	b.Empty = s
	if len(color) > 0 {
		b.Empty = SetColor(b.Empty, 0, 0, colorToCode(color[0]))
	}
}

// Add the specified amount to the progressbar
func (b *Bar) Add(n int64) {
	b.mu.Lock()
	b.current += n
	if b.current > b.total {
		b.current = b.total
	}
	shouldClose := b.current == b.total
	if shouldClose {
		b.Status = "Success"
	}
	b.mu.Unlock()

	if shouldClose {
		b.Close()
	}
}

// string return the progress bar
func (b *Bar) string() string {
	if b.tmpl == nil {
		return fmt.Sprintf("%s %.0f%%", b.Status, b.percent())
	}

	var buf bytes.Buffer
	if b.rate == "" {
		b.rate = "[" + b.bytesToSize(0) + "/s]"
	}
	data := struct {
		Status  string
		Name    string
		Percent float64
		Bar     string
		Text    string
		Rate    string
		Total   string
	}{
		Status:  b.Status,
		Name:    b.Name,
		Percent: b.percent(),
		Bar:     b.bar(),
		Text:    b.text,
		Rate:    b.rate,
		Total:   b.formatTotal(),
	}

	data.Total = SetColor(b.formatTotal(), 0, 0, green)
	if err := b.tmpl.Execute(&buf, data); err != nil {
		Errorf("Failed to execute progress bar template: %v", err)
		return fmt.Sprintf("%s %.0f%%", b.Status, b.percent())
	}

	return buf.String()
}

// percent return the percentage
func (b *Bar) percent() float64 {
	if b.total == 0 {
		return 0
	}
	return (float64(b.current) / float64(b.total)) * 100
}

// formatTotal return the format total
func (b *Bar) formatTotal() string {
	return b.bytesToSize(b.current) + "/" + b.bytesToSize(b.total)
}

// Bar return the progress bar string
func (b *Bar) bar() string {
	var p float64
	if b.total > 0 {
		p = float64(b.current) / float64(b.total)
	}

	// Ensure progress ratio is between 0-1
	if p < 0 {
		p = 0
	} else if p > 1 {
		p = 1
	}

	// Ensure width is reasonable
	width := b.Width
	if width <= 0 {
		width = defaultBarWidth
	}

	filled := math.Ceil(float64(width) * p)
	empty := math.Floor(float64(width) - filled)

	// Ensure repeat count is not negative
	filledCount := int(filled)
	emptyCount := int(empty)
	if filledCount < 0 {
		filledCount = 0
	}
	if emptyCount < 0 {
		emptyCount = 0
	}

	s := b.StartDelimiter
	s += strings.Repeat(b.Filled, filledCount)
	s += strings.Repeat(b.Empty, emptyCount)
	s += b.EndDelimiter
	return s
}

// Render write the progress bar to io.Writer
func (b *Bar) Render(w io.Writer) int64 {
	s := fmt.Sprintf("\x1bM\r %s", b.string())
	fmt.Print("\x1B7")     // save the cursor position
	fmt.Print("\x1B[0J")   // erase from cursor to end of screen
	fmt.Print("\x1B[?47l") // restore screen
	io.WriteString(w, s)
	return int64(len(s))
}

// Write implement io.Writer
func (b *Bar) Write(bytes []byte) (n int, err error) {
	n = len(bytes)
	b.Add(int64(n))
	b.Render(os.Stdout)
	return
}

// bytesToSize format bytes to string
func (b *Bar) bytesToSize(bytes int64) string {
	sizes := []string{"Bytes", "KB", "MB", "GB", "TB"}
	if bytes == 0 {
		return "0 Bytes"
	}
	if bytes < 0 {
		return "0 Bytes"
	}

	i := math.Floor(math.Log(float64(bytes)) / math.Log(float64(bytesPerKB)))
	if int(i) >= len(sizes) {
		i = float64(len(sizes) - 1)
	}

	r := float64(bytes) / math.Pow(float64(bytesPerKB), i)
	return strconv.FormatFloat(r, 'f', 2, 64) + " " + sizes[int(i)]
}

// Close the rate listen safely (can be called multiple times)
func (b *Bar) Close() {
	if b.closed.CompareAndSwap(false, true) {
		close(b.done)
	}
}
