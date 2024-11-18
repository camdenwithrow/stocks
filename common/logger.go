package common

import (
	"fmt"
	go_log "log"
	"os"
)

const (
	INFO  = "INFO"
	DEBUG = "DEBUG"
	WARN  = "WARN"
	ERROR = "ERROR"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) logMessage(logLevel, color, msg string) {
	go_log.SetOutput(os.Stdout)
	// log.Printf("%s[%s] %s: %s%s\n", color, logLevel, time.Now().Format(time.RFC3339), msg, colorReset)
	// fmt.Printf("%s[%s] %s: %s%s\n", color, logLevel, time.Now().Format("2006-01-02 15:04"), msg, colorReset)
	fmt.Printf("%s[%s] %s%s\n", color, logLevel, msg, colorReset)
}

func (l *Logger) Debug(msg string, a ...any) {
	l.logMessage(DEBUG, colorCyan, fmt.Sprintf(msg, a...))
}

func (l *Logger) Info(msg string, a ...any) {
	l.logMessage(INFO, colorGreen, fmt.Sprintf(msg, a...))
}

func (l *Logger) Warn(msg string, a ...any) {
	l.logMessage(WARN, colorYellow, fmt.Sprintf(msg, a...))
}

func (l *Logger) Error(msg string, a ...any) {
	l.logMessage(ERROR, colorRed, fmt.Sprintf(msg, a...))
}
