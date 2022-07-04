package main

import "fmt"

const (
	reset = iota
	bold
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

	white = 97
)

func SetColor(msg string, conf, bg, text int) string {
	return fmt.Sprintf("%c[%d;%d;%dm%s%c[0m", 0x1B, conf, bg, text, msg, 0x1B)
}

func BlackText(s string) {
	println(SetColor(s, 0, 0, black))
}

func BoldText(s string) {
	println(SetColor(s, 0, 0, bold))
}

func RedText(s string) {
	println(SetColor(s, 0, 0, red))
}

func GreenText(s string) {
	println(SetColor(s, 0, 0, green))
}

func YellowText(s string) {
	println(SetColor(s, 0, 0, yellow))
}

func BlueText(s string) {
	println(SetColor(s, 0, 0, blue))
}

func PinkText(s string) {
	println(SetColor(s, 0, 0, pink))
}

func CyanText(s string) {
	println(SetColor(s, 0, 0, cyan))
}

func GrayText(s string) {
	println(SetColor(s, 0, 0, gray))
}

func WhiteText(s string) {
	println(SetColor(s, 0, 0, white))
}
