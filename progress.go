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
	"time"
	//github.com/fatih/color
)

// Bar is a progress bar
type Bar struct {
	StartDelimiter string // StartDelimiter for the bar ("|")
	EndDelimiter   string // EndDelimiter for the bar ("|")
	Filled         string // Filled section representation ("█ ■")
	Empty          string // Empty section representation ("░ □")
	Width          int    // Width of the bar

	text    string
	rate    string
	prev    int64
	current int64
	total   int64
	tmpl    *template.Template
	done    chan struct{}
}

// NewBar return a new bar with the given total
func NewBar(total int64) *Bar {
	b := &Bar{
		StartDelimiter: "|",
		EndDelimiter:   "|",
		Filled:         "█",
		Empty:          "░",
		Width:          40,
		total:          total,
		done:           make(chan struct{}),
	}
	go b.listenRate()
	b.template(`{{.Percent | printf "%3.0f"}}% {{.Bar}} {{.Total}} {{.Rate}} {{.Text}}`)

	return b
}

// listenRate start listen the speed
func (b *Bar) listenRate() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			r := b.current - b.prev
			b.rate = "[" + b.bytesToSize(r) + "/s]"
			b.prev = b.current
		case <-b.done:
			close(b.done)
			return
		}
	}
}

// template for rendering. This method will panic if the template fails to parse
func (b *Bar) template(s string) {
	t, err := template.New("").Parse(s)
	if err != nil {
		panic(err)
	}
	b.tmpl = t
}

// Text set the text value
func (b *Bar) Text(s string) {
	b.text = s
}

// Add the specified amount to the progressbar
func (b *Bar) Add(n int64) {
	b.current += n
	if b.current > b.total {
		panic("cannot be greater than the total")
	}
}

// string return the progress bar
func (b *Bar) string() string {
	var buf bytes.Buffer
	data := struct {
		Percent float64
		Bar     string
		Text    string
		Rate    string
		Total   string
	}{
		Percent: b.percent(),
		Bar:     b.bar(),
		Text:    b.text,
		Rate:    b.rate,
		Total:   b.formatTotal(),
	}

	if err := b.tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

// percent return the percentage
func (b *Bar) percent() float64 {
	return (float64(b.current) / float64(b.total)) * 100
}

// formatTotal return the format total
func (b *Bar) formatTotal() string {
	return b.bytesToSize(b.current) + "/" + b.bytesToSize(b.total)
}

// Bar return the progress bar string
func (b *Bar) bar() string {
	p := float64(b.current) / float64(b.total)
	filled := math.Ceil(float64(b.Width) * p)
	empty := math.Floor(float64(b.Width) - filled)
	s := b.StartDelimiter
	s += strings.Repeat(b.Filled, int(filled))
	s += strings.Repeat(b.Empty, int(empty))
	s += b.EndDelimiter
	return s
}

// Render write the progress bar to io.Writer
func (b *Bar) Render(w io.Writer) int64 {
	s := fmt.Sprintf("\r   %s ", b.string())
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
	var k = 1024
	var sizes = []string{"Bytes", "KB", "MB", "GB", "TB"}
	if bytes == 0 {
		return "0 Bytes"
	}
	i := math.Floor(math.Log(float64(bytes)) / math.Log(float64(k)))
	r := float64(bytes) / math.Pow(float64(k), i)
	return strconv.FormatFloat(r, 'f', 2, 64) + sizes[int(i)]
}

// Close the rate listen
func (b *Bar) Close() {
	b.done <- struct{}{}
}
