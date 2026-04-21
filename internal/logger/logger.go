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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const logFileLocation = "/var/log/jfrog-credentials-provider/jfrog-credentials-provider.log"

type Logger struct {
	Logger *slog.Logger
}

func NewLogger() (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logFileLocation), 0755); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("log_level"), "debug") {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})

	return &Logger{
		Logger: slog.New(handler),
	}, nil
}

func (l *Logger) Info(message interface{}) {
	l.Logger.Info(toStr(message))
}

func (l *Logger) Debug(message interface{}) {
	l.Logger.Debug(toStr(message))
}

func (l *Logger) Error(message interface{}) {
	l.Logger.Error(toStr(message))
}

func (l *Logger) Exit(message interface{}, code int) {
	l.Logger.Error(toStr(message))
	os.Exit(code)
}

func toStr(message interface{}) string {
	switch v := message.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}
