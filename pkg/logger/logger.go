package logger

import (
	"log"
	"os"
)

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
}

type simpleLogger struct {
	logger *log.Logger
}

func New() Logger {
	return &simpleLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

func (l *simpleLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("[INFO] %s %v", msg, keysAndValues)
}

func (l *simpleLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("[ERROR] %s %v", msg, keysAndValues)
}

func (l *simpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("[DEBUG] %s %v", msg, keysAndValues)
}

func (l *simpleLogger) Fatal(msg string, keysAndValues ...interface{}) {
	l.logger.Fatalf("[FATAL] %s %v", msg, keysAndValues)
}
