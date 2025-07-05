package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = DEBUG
	useColor     = false
)

func init() {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		useColor = true
	}
}

func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		currentLevel = DEBUG
	case "info":
		currentLevel = INFO
	case "warn":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	}
}

// 日志函数
func logf(level LogLevel, format string, v ...interface{}) {
	if level < currentLevel {
		return
	}

	var prefix string
	if useColor {
		switch level {
		case DEBUG:
			prefix = Cyan("[DEBUG]")
		case INFO:
			prefix = Green("[INFO]")
		case WARN:
			prefix = Yellow("[WARN]")
		case ERROR:
			prefix = Red("[ERROR]")
		}
	} else {
		switch level {
		case DEBUG:
			prefix = "[DEBUG]"
		case INFO:
			prefix = "[INFO]"
		case WARN:
			prefix = "[WARN]"
		case ERROR:
			prefix = "[ERROR]"
		}
	}

	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(format, v...))
}

func Debugf(format string, v ...interface{}) { logf(DEBUG, format, v...) }
func Infof(format string, v ...interface{})  { logf(INFO, format, v...) }
func Warnf(format string, v ...interface{})  { logf(WARN, format, v...) }
func Errorf(format string, v ...interface{}) { logf(ERROR, format, v...) }

func Debug(v ...interface{}) { logf(DEBUG, fmt.Sprint(v...)) }
func Info(v ...interface{})  { logf(INFO, fmt.Sprint(v...)) }
func Warn(v ...interface{})  { logf(WARN, fmt.Sprint(v...)) }
func Error(v ...interface{}) { logf(ERROR, fmt.Sprint(v...)) }
