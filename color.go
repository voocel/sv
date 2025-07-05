package main

import "fmt"

const (
	reset = iota
	bold
	fuzzy
	italic
	underscore
	blink
	fastBlink
	reverse
	concealed
	strikethrough
)

const (
	black = iota + 30
	red
	green
	yellow
	blue
	pink
	cyan
	gray

	white   = 97
	unknown = 999
)

var colorMap = map[int]string{
	bold:    "bold",
	black:   "black",
	red:     "red",
	green:   "green",
	yellow:  "yellow",
	blue:    "blue",
	pink:    "pink",
	cyan:    "cyan",
	gray:    "gray",
	white:   "white",
	unknown: "unknown",
}

func SetColor(text string, conf, bg, color int) string {
	return fmt.Sprintf("%c[%d;%d;%dm%s%c[0m", 0x1B, conf, bg, color, text, 0x1B)
}

func Bold(s string) string {
	return SetColor(s, 0, 0, bold)
}

func Black(s string) string {
	return SetColor(s, 0, 0, black)
}

func Red(s string) string {
	return SetColor(s, 0, 0, red)
}

func Green(s string) string {
	return SetColor(s, 0, 0, green)
}

func Yellow(s string) string {
	return SetColor(s, 0, 0, yellow)
}

func Blue(s string) string {
	return SetColor(s, 0, 0, blue)
}

func Pink(s string) string {
	return SetColor(s, 0, 0, pink)
}

func Cyan(s string) string {
	return SetColor(s, 0, 0, cyan)
}

func Gray(s string) string {
	return SetColor(s, 0, 0, gray)
}

func White(s string) string {
	return SetColor(s, 0, 0, white)
}

func PrintBold(s string) {
	fmt.Println(Bold(s))
}

func PrintBlack(s string) {
	fmt.Println(Black(s))
}

func PrintRed(s string) {
	fmt.Println(Red(s))
}

func PrintGreen(s string) {
	fmt.Println(Green(s))
}

func PrintYellow(s string) {
	fmt.Println(Yellow(s))
}

func PrintBlue(s string) {
	fmt.Println(Blue(s))
}

func PrintPink(s string) {
	fmt.Println(Pink(s))
}

func PrintCyan(s string) {
	fmt.Println(Cyan(s))
}

func PrintGray(s string) {
	fmt.Println(Gray(s))
}

func PrintWhite(s string) {
	fmt.Println(White(s))
}

func codeReason(code int) string {
	v, ok := colorMap[code]
	if !ok {
		v = colorMap[unknown]
	}
	return v
}

func colorToCode(s string) int {
	for k := range colorMap {
		if colorMap[k] == s {
			return k
		}
	}
	return unknown
}
