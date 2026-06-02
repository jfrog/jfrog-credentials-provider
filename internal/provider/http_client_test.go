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

package provider

import (
	"net/http"
	"testing"
	"time"
)

func TestNewProviderHTTPClientUsesProxyFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example:8080")
	t.Setenv("NO_PROXY", "")

	client := newProviderHTTPClient(defaultHTTPTimeout)
	if client.Timeout != defaultHTTPTimeout {
		t.Fatalf("expected timeout %s, got %s", defaultHTTPTimeout, client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}

	req, err := http.NewRequest(http.MethodGet, "http://registry.example/v2/", nil)
	if err != nil {
		t.Fatal(err)
	}

	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if proxyURL == nil || proxyURL.String() != "http://proxy.example:8080" {
		t.Fatalf("expected proxy from HTTP_PROXY, got %v", proxyURL)
	}
}

func TestNewProviderHTTPClientPreservesTransportSettings(t *testing.T) {
	client := newProviderHTTPClient(60 * time.Second)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}

	if transport.MaxIdleConns != 100 {
		t.Fatalf("expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != 10*time.Second {
		t.Fatalf("expected IdleConnTimeout 10s, got %s", transport.IdleConnTimeout)
	}
	if !transport.DisableCompression {
		t.Fatal("expected DisableCompression to be true")
	}
}
