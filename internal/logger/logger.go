package logger

import (
	"log"
	"os"
)

const logFileLocation = "/var/log/jfrog-credential-provider.log"
const logPrefix = "[JFROG CREDENTIALS PROVIDER] "

type Logger struct {
	Logger *log.Logger
}

func NewLogger() (*Logger, error) {
	logFile, err := os.OpenFile(logFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		Logger: log.New(logFile, logPrefix, log.Ldate|log.Ltime|log.Lshortfile),
	}, nil
}

func (l *Logger) Info(message interface{}) {
	l.Logger.Println("[INFO] " + formatMessage(message))
}

func (l *Logger) Debug(message interface{}) {
	l.Logger.Println("[DEBUG] " + formatMessage(message))
}

func (l *Logger) Error(message interface{}) {
	l.Logger.Println("[ERROR] " + formatMessage(message))
}

func (l *Logger) Exit(message interface{}, code int) {
	l.Logger.Println("[EXIT] " + formatMessage(message))
	os.Exit(code)
}

func formatMessage(message interface{}) string {
	switch v := message.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return "unknown message type"
	}
}
