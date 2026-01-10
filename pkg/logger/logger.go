package logger

import (
	"log"
	"os"
)

type Logger struct {
	level string
}

func New(level string) *Logger {
	return &Logger{level: level}
}

func (l *Logger) Info(msg string) {
	log.Printf("[INFO] %s", msg)
}

func (l *Logger) Error(msg string) {
	log.Printf("[ERROR] %s", msg)
}

func (l *Logger) Debug(msg string) {
	if l.level == "debug" {
		log.Printf("[DEBUG] %s", msg)
	}
}

func (l *Logger) Warn(msg string) {
	log.Printf("[WARN] %s", msg)
}

func (l *Logger) Fatal(msg string) {
	log.Printf("[FATAL] %s", msg)
	os.Exit(1)
}
