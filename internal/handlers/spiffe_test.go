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

package handlers

import (
	"context"
	"io"
	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/logger"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// testService builds a Service with a discarding logger so handler helpers can
// be exercised without touching the on-disk log file.
func testService() *service.Service {
	lg := logger.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	return service.NewService(&http.Client{}, lg)
}

func TestCheckIfSpiffe(t *testing.T) {
	// Create a stand-in socket file. CheckIfSpiffe only stats the path for
	// existence, so a regular file is sufficient (and avoids the unix-socket
	// path-length limits that a real net.Listen would hit under t.TempDir).
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "agent.sock")
	if err := os.WriteFile(socketPath, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		envKey   string
		envValue string
		want     bool
	}{
		{name: "unset", envKey: "spiffe_endpoint_socket", envValue: "", want: false},
		{name: "missing_socket", envKey: "spiffe_endpoint_socket", envValue: "unix:///nonexistent/agent.sock", want: false},
		{name: "present_lowercase", envKey: "spiffe_endpoint_socket", envValue: "unix://" + socketPath, want: true},
		{name: "present_uppercase_fallback", envKey: "SPIFFE_ENDPOINT_SOCKET", envValue: "unix://" + socketPath, want: true},
	}

	svc := testService()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear both possible sources, then set the one under test.
			t.Setenv("spiffe_endpoint_socket", "")
			t.Setenv("SPIFFE_ENDPOINT_SOCKET", "")
			t.Setenv(tc.envKey, tc.envValue)

			got, err := CheckIfSpiffe(svc, context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CheckIfSpiffe = %v, want %v", got, tc.want)
			}
		})
	}
}
