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

// Package handlers_test proves the full D1 flow — a CredentialProviderRequest
// carrying an id_token, exchanged against a stub Artifactory OIDC endpoint,
// producing a CredentialProviderResponse with the exchanged token — without
// any network access outside an in-process httptest server. It lives in the
// _test package (not package handlers) because it needs both internal/handlers
// and internal/provider, which would otherwise be an import cycle.
package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/logger"
	"jfrog-credential-provider/internal/provider"
	"jfrog-credential-provider/internal/utils"
)

func testLogger() logger.Logger {
	return logger.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// newStubArtifactory starts an in-process TLS server standing in for
// Artifactory's /access/api/v1/oidc/token endpoint. Its own *http.Client
// (via ts.Client()) already trusts the server's self-signed certificate, so
// no changes to the production TLS/URL-building code are needed.
func newStubArtifactory(t *testing.T, respond func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(respond))
}

func stubHost(ts *httptest.Server) string {
	return strings.TrimPrefix(ts.URL, "https://")
}

func TestGenericOidcFlow_EndToEnd(t *testing.T) {
	const wantIdToken = "fake.header.payload"
	const wantProviderName = "my-keycloak-provider"
	const wantAudience = "my-audience"

	var gotRequest handlers.OidcTokenRequest
	ts := newStubArtifactory(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != handlers.OIDC_ENDPOINT {
			t.Errorf("expected path %s, got %s", handlers.OIDC_ENDPOINT, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(handlers.OidcAccessResponse{
			AccessToken: "stub-short-lived-rt-token",
			TokenType:   "Bearer",
			ExpiresIn:   600,
			Username:    "stub-user",
		})
	})
	defer ts.Close()

	request := utils.CredentialProviderRequest{
		ApiVersion:          "credentialprovider.kubelet.k8s.io/v1",
		Kind:                "CredentialProviderRequest",
		Image:               "myregistry.jfrog.io/repo/image:tag",
		ServiceAccountToken: wantIdToken,
	}
	svc := service.NewService(ts.Client(), testLogger())

	idToken, err := handlers.GetGenericIdentityToken(request)
	if err != nil {
		t.Fatalf("GetGenericIdentityToken returned unexpected error: %v", err)
	}

	username, token, err := handlers.ExchangeOidcArtifactoryToken(
		svc, context.Background(), idToken, stubHost(ts), wantProviderName, wantAudience)
	if err != nil {
		t.Fatalf("ExchangeOidcArtifactoryToken returned unexpected error: %v", err)
	}

	// The stub saw the right token-exchange request shape.
	if gotRequest.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		t.Errorf("unexpected grant_type: %s", gotRequest.GrantType)
	}
	if gotRequest.SubjectTokenType != "urn:ietf:params:oauth:token-type:id_token" {
		t.Errorf("unexpected subject_token_type: %s", gotRequest.SubjectTokenType)
	}
	if gotRequest.ProviderType != "Generic OpenID Connect" {
		t.Errorf("unexpected provider_type: %s", gotRequest.ProviderType)
	}
	if gotRequest.SubjectToken != wantIdToken {
		t.Errorf("expected subject_token %q, got %q", wantIdToken, gotRequest.SubjectToken)
	}
	if gotRequest.ProviderName != wantProviderName {
		t.Errorf("expected provider_name %q, got %q", wantProviderName, gotRequest.ProviderName)
	}
	if gotRequest.Audience != wantAudience {
		t.Errorf("expected audience %q, got %q", wantAudience, gotRequest.Audience)
	}

	// The provider's CredentialProviderResponse carries the exchanged token.
	resp := provider.BuildCredentialProviderResponse(request, username, token)
	if resp.ApiVersion != "credentialprovider.kubelet.k8s.io/v1" {
		t.Errorf("unexpected apiVersion: %s", resp.ApiVersion)
	}
	if resp.Kind != "CredentialProviderResponse" {
		t.Errorf("unexpected kind: %s", resp.Kind)
	}
	cred, ok := resp.Auth.Registry["myregistry.jfrog.io"]
	if !ok {
		t.Fatalf("expected registry entry for myregistry.jfrog.io, got: %+v", resp.Auth.Registry)
	}
	if cred.Username != "stub-user" {
		t.Errorf("expected username %q, got %q", "stub-user", cred.Username)
	}
	if cred.Password != "stub-short-lived-rt-token" {
		t.Errorf("expected password %q, got %q", "stub-short-lived-rt-token", cred.Password)
	}

	// The response marshals into the kubelet-facing JSON shape.
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal CredentialProviderResponse: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("failed to unmarshal produced JSON: %v", err)
	}
	auth, ok := raw["auth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected auth object in response JSON, got: %s", jsonBytes)
	}
	if _, ok := auth["myregistry.jfrog.io"]; !ok {
		t.Errorf("expected registry key myregistry.jfrog.io in response JSON, got: %s", jsonBytes)
	}
}

func TestGenericOidcFlow_ArtifactoryError(t *testing.T) {
	ts := newStubArtifactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid subject_token"}`))
	})
	defer ts.Close()

	svc := service.NewService(ts.Client(), testLogger())
	_, _, err := handlers.ExchangeOidcArtifactoryToken(
		svc, context.Background(), "fake.header.payload", stubHost(ts), "my-provider", "my-audience")
	if err == nil {
		t.Fatal("expected an error when Artifactory stub returns 401, got nil")
	}
}

func TestGetGenericIdentityToken_MissingServiceAccountToken(t *testing.T) {
	request := utils.CredentialProviderRequest{
		Image: "myregistry.jfrog.io/repo/image:tag",
	}
	_, err := handlers.GetGenericIdentityToken(request)
	if err == nil {
		t.Fatal("expected an error when ServiceAccountToken is empty, got nil")
	}
}

// TestGenericAuth_EndToEnd drives provider.GenericAuth, the function the
// kubelet path actually calls for cloud_provider=generic. TestGenericOidcFlow_EndToEnd
// above chains the three handler calls from the test body; this one leaves the
// chaining to production code, so a regression in the wiring (wrong env var,
// wrong argument order, provider name and audience swapped) fails here.
func TestGenericAuth_EndToEnd(t *testing.T) {
	const wantIdToken = "fake.header.payload"
	const wantProviderName = "my-keycloak-provider"
	const wantAudience = "my-audience"

	var gotRequest handlers.OidcTokenRequest
	ts := newStubArtifactory(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != handlers.OIDC_ENDPOINT {
			t.Errorf("expected path %s, got %s", handlers.OIDC_ENDPOINT, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(handlers.OidcAccessResponse{
			AccessToken: "stub-short-lived-rt-token",
			TokenType:   "Bearer",
			ExpiresIn:   600,
			Username:    "stub-user",
		})
	})
	defer ts.Close()

	t.Setenv("jfrog_oidc_provider_name", wantProviderName)
	t.Setenv("jfrog_oidc_audience", wantAudience)

	request := utils.CredentialProviderRequest{
		ApiVersion:          "credentialprovider.kubelet.k8s.io/v1",
		Kind:                "CredentialProviderRequest",
		Image:               "myregistry.jfrog.io/repo/image:tag",
		ServiceAccountToken: wantIdToken,
	}
	lg := testLogger()
	svc := service.NewService(ts.Client(), lg)

	username, token, err := provider.GenericAuth(svc, context.Background(), &lg, stubHost(ts), request)
	if err != nil {
		t.Fatalf("GenericAuth returned unexpected error: %v", err)
	}

	if gotRequest.SubjectToken != wantIdToken {
		t.Errorf("expected subject_token %q, got %q", wantIdToken, gotRequest.SubjectToken)
	}
	if gotRequest.ProviderName != wantProviderName {
		t.Errorf("expected provider_name %q, got %q", wantProviderName, gotRequest.ProviderName)
	}
	if gotRequest.Audience != wantAudience {
		t.Errorf("expected audience %q, got %q", wantAudience, gotRequest.Audience)
	}

	resp := provider.BuildCredentialProviderResponse(request, username, token)
	cred, ok := resp.Auth.Registry["myregistry.jfrog.io"]
	if !ok {
		t.Fatalf("expected registry entry for myregistry.jfrog.io, got: %+v", resp.Auth.Registry)
	}
	if cred.Username != "stub-user" {
		t.Errorf("expected username %q, got %q", "stub-user", cred.Username)
	}
	if cred.Password != "stub-short-lived-rt-token" {
		t.Errorf("expected password %q, got %q", "stub-short-lived-rt-token", cred.Password)
	}
}

func TestGenericAuth_MissingEnvVars(t *testing.T) {
	cases := []struct {
		name         string
		providerName string
		audience     string
	}{
		{name: "both_missing"},
		{name: "audience_missing", providerName: "my-keycloak-provider"},
		{name: "provider_name_missing", audience: "my-audience"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newStubArtifactory(t, func(w http.ResponseWriter, r *http.Request) {
				t.Error("Artifactory must not be called when configuration is incomplete")
			})
			defer ts.Close()

			t.Setenv("jfrog_oidc_provider_name", tc.providerName)
			t.Setenv("jfrog_oidc_audience", tc.audience)

			lg := testLogger()
			svc := service.NewService(ts.Client(), lg)
			request := utils.CredentialProviderRequest{
				Image:               "myregistry.jfrog.io/repo/image:tag",
				ServiceAccountToken: "fake.header.payload",
			}

			if _, _, err := provider.GenericAuth(svc, context.Background(), &lg, stubHost(ts), request); err == nil {
				t.Fatal("expected an error when required env vars are missing, got nil")
			}
		})
	}
}
