package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"math"
	"strings"
)

// Bar is a progress bar.
type Bar struct {
	StartDelimiter string  // StartDelimiter for the bar ("|")
	EndDelimiter   string  // EndDelimiter for the bar ("|")
	Filled         string  // Filled section representation ("█")
	Empty          string  // Empty section representation ("░")
	Total          float64 // Total value
	Width          int     // Width of the bar

	value float64
	tmpl  *template.Template
	text  string
	speed string
}

// NewBar return a new bar with the given total
func NewBar(total float64) *Bar {
	b := &Bar{
		StartDelimiter: "|",
		EndDelimiter:   "|",
		Filled:         "█",
		Empty:          "░",
		Total:          total,
		Width:          60,
	}

	b.Template(`{{.Percent | printf "%3.0f"}}% {{.Bar}} {{.Text}} {{.Speed}}`)

	return b
}

// NewInt return a new bar with the given total
func NewInt(total int64) *Bar {
	return NewBar(float64(total))
}

// Text set the text value
func (b *Bar) Text(s string) {
	b.text = s
}

// Speed set the speed value
func (b *Bar) Speed(s string) {
	b.speed = s
}

// Value set the value
func (b *Bar) Value(n float64) {
	if n > b.Total {
		panic("Bar update value cannot be greater than the total")
	}
	b.value = n
}

// ValueInt set the value
func (b *Bar) ValueInt(n int64) {
	b.Value(float64(n))
}

// percent return the percentage
func (b *Bar) percent() float64 {
	return (b.value / b.Total) * 100
}

// Bar return the progress bar string
func (b *Bar) bar() string {
	p := b.value / b.Total
	filled := math.Ceil(float64(b.Width) * p)
	empty := math.Floor(float64(b.Width) - filled)
	s := b.StartDelimiter
	s += strings.Repeat(b.Filled, int(filled))
	s += strings.Repeat(b.Empty, int(empty))
	s += b.EndDelimiter
	return s
}

// String return the progress bar
func (b *Bar) String() string {
	var buf bytes.Buffer
	data := struct {
		Value          float64
		Total          float64
		Percent        float64
		StartDelimiter string
		EndDelimiter   string
		Bar            string
		Text           string
		Speed          string
	}{
		Value:          b.value,
		Percent:        b.percent(),
		StartDelimiter: b.StartDelimiter,
		EndDelimiter:   b.EndDelimiter,
		Bar:            b.bar(),
		Text:           b.text,
		Speed:          b.speed,
	}

	if err := b.tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

// WriteTo write the progress bar to w
func (b *Bar) WriteTo(w io.Writer) (int64, error) {
	s := fmt.Sprintf("\r   %s ", b.String())
	_, err := io.WriteString(w, s)
	return int64(len(s)), err
}

// Template for rendering. This method will panic if the template fails to parse
func (b *Bar) Template(s string) {
	t, err := template.New("").Parse(s)
	if err != nil {
		panic(err)
	}

	b.tmpl = t
}
