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
	//github.com/fatih/color
)

// Bar is a progress bar.
type Bar struct {
	StartDelimiter string // StartDelimiter for the bar ("|")
	EndDelimiter   string // EndDelimiter for the bar ("|")
	Filled         string // Filled section representation ("█")
	Empty          string // Empty section representation ("░")
	Width          int    // width of the bar

	total   float64
	text    string
	rate    string
	current float64
	tmpl    *template.Template
}

// NewBar return a new bar with the given total
func NewBar(total float64) *Bar {
	b := &Bar{
		StartDelimiter: "|",
		EndDelimiter:   "|",
		Filled:         "█",
		Empty:          "░",
		Width:          50,
		total:          total,
	}
	b.template(`{{.Percent | printf "%3.0f"}}% {{.Bar}} {{.Text}} {{.Speed}} {{.Total}}`)

	return b
}

// NewBarInt return a new bar with the given total
func NewBarInt(total int64) *Bar {
	return NewBar(float64(total))
}

// Text set the text value
func (b *Bar) Text(s string) {
	b.text = s
}

// Rate set the speed value
func (b *Bar) Rate(s string) {
	b.rate = s
}

// Add the specified amount to the progressbar
func (b *Bar) Add(n float64) {
	b.current += n
	if b.current > b.total {
		panic("cannot be greater than the total")
	}
}

// Bar return the progress bar string
func (b *Bar) bar() string {
	p := b.current / b.total
	filled := math.Ceil(float64(b.Width) * p)
	empty := math.Floor(float64(b.Width) - filled)
	s := b.StartDelimiter
	s += strings.Repeat(b.Filled, int(filled))
	s += strings.Repeat(b.Empty, int(empty))
	s += b.EndDelimiter
	return s
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
		Total:   b.bytesToSize(int64(b.current)) + "/" + b.bytesToSize(int64(b.total)),
	}

	if err := b.tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

// percent return the percentage
func (b *Bar) percent() float64 {
	return (b.current / b.total) * 100
}

// Template for rendering. This method will panic if the template fails to parse
func (b *Bar) template(s string) {
	t, err := template.New("").Parse(s)
	if err != nil {
		panic(err)
	}
	b.tmpl = t
}

// WriteTo write the progress bar to io.Writer
func (b *Bar) WriteTo(w io.Writer) (int64, error) {
	s := fmt.Sprintf("\r   %s ", b.string())
	_, err := io.WriteString(w, s)
	return int64(len(s)), err
}

// Write implement io.Writer
func (b *Bar) Write(bytes []byte) (n int, err error) {
	n = len(bytes)
	b.Add(float64(n))
	b.WriteTo(os.Stdout)
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
