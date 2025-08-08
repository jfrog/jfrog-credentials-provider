// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
