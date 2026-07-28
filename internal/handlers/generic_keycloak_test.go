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

//go:build keycloak_integration

// This file is gated behind the keycloak_integration build tag so a plain
// `go test ./...` (Tier A, D1) never depends on Docker being available. It
// proves the same generic OIDC flow as jfrog_exchange_test.go, but the
// id_token is a real JWT minted by a real Keycloak realm imported from
// test/keycloak/realm-export.json (D3), instead of a hand-typed fake string.
// Artifactory is still a stub httptest server — D1 explicitly allows that.
package handlers_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/provider"
	"jfrog-credential-provider/internal/utils"
)

const (
	keycloakDefaultBaseURL = "http://localhost:8080"
	keycloakRealm          = "jfrog-poc"
	keycloakClientID       = "jfrog-credential-provider"
	keycloakClientSecret   = "poc-test-secret-not-for-prod"
	keycloakUsername       = "poc-user"
	keycloakPassword       = "poc-password"
	keycloakExpectedAud    = "jfrog-artifactory-poc"
)

func keycloakBaseURL() string {
	if v := os.Getenv("KEYCLOAK_BASE_URL"); v != "" {
		return v
	}
	return keycloakDefaultBaseURL
}

// keycloakReadyTimeout bounds how long we wait for Keycloak to boot and import
// the realm. Cold CI runners are markedly slower than a warm local Docker, so
// the default is generous and KEYCLOAK_READY_TIMEOUT_SECONDS can raise it
// further rather than the test failing for being merely slow.
func keycloakReadyTimeout() time.Duration {
	if v := os.Getenv("KEYCLOAK_READY_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 120 * time.Second
}

// waitForKeycloak polls the realm's OIDC discovery document until Keycloak
// has finished starting up and importing the realm, or the deadline passes.
func waitForKeycloak(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	discoveryURL := fmt.Sprintf("%s/realms/%s/.well-known/openid-configuration", baseURL, keycloakRealm)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(discoveryURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("unexpected status %d from %s", resp.StatusCode, discoveryURL)
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("Keycloak at %s did not become ready within %s: %v", baseURL, timeout, lastErr)
}

// fetchKeycloakIDToken performs a Resource Owner Password Credentials grant
// against the imported realm and returns the id_token from the response.
func fetchKeycloakIDToken(t *testing.T, baseURL string) string {
	t.Helper()
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", baseURL, keycloakRealm)

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", keycloakClientID)
	form.Set("client_secret", keycloakClientSecret)
	form.Set("username", keycloakUsername)
	form.Set("password", keycloakPassword)
	form.Set("scope", "openid")

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		t.Fatalf("failed to call Keycloak token endpoint: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		IDToken          string `json:"id_token"`
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to decode Keycloak token response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Keycloak token endpoint returned %d: %s (%s)", resp.StatusCode, tokenResp.Error, tokenResp.ErrorDescription)
	}
	if tokenResp.IDToken == "" {
		t.Fatalf("Keycloak token response did not contain an id_token")
	}
	return tokenResp.IDToken
}

// decodeJWTClaims base64-decodes the payload segment of a JWT without
// verifying its signature - sufficient here since we only want to sanity
// check the claims Keycloak minted, not re-implement token verification.
func decodeJWTClaims(t *testing.T, jwt string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-part JWT, got %d parts", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to base64-decode JWT payload: %v", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("failed to unmarshal JWT claims: %v", err)
	}
	return claims
}

func TestGenericOidcFlow_KeycloakBacked(t *testing.T) {
	baseURL := keycloakBaseURL()
	waitForKeycloak(t, baseURL, keycloakReadyTimeout())

	idToken := fetchKeycloakIDToken(t, baseURL)

	claims := decodeJWTClaims(t, idToken)
	if iss, _ := claims["iss"].(string); !strings.Contains(iss, "/realms/"+keycloakRealm) {
		t.Errorf("expected issuer to reference realm %q, got %q", keycloakRealm, iss)
	}
	switch aud := claims["aud"].(type) {
	case string:
		if aud != keycloakExpectedAud {
			t.Errorf("expected aud %q, got %q", keycloakExpectedAud, aud)
		}
	case []interface{}:
		found := false
		for _, a := range aud {
			if s, _ := a.(string); s == keycloakExpectedAud {
				found = true
			}
		}
		if !found {
			t.Errorf("expected aud to contain %q, got %v", keycloakExpectedAud, aud)
		}
	default:
		t.Errorf("unexpected aud claim type: %T", aud)
	}

	// Same stub-Artifactory exchange as the pure unit test - only the id_token's
	// provenance differs (real Keycloak JWT vs. a fake string). The exchange
	// request is asserted just as strictly here, so this tier proves the real
	// JWT reaches Artifactory intact rather than only that some token did.
	const wantProviderName = "my-keycloak-provider"

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
			AccessToken: "keycloak-backed-stub-rt-token",
			TokenType:   "Bearer",
			ExpiresIn:   600,
			Username:    "stub-user",
		})
	})
	defer ts.Close()

	t.Setenv("jfrog_oidc_provider_name", wantProviderName)
	t.Setenv("jfrog_oidc_audience", keycloakExpectedAud)

	request := utils.CredentialProviderRequest{
		ApiVersion:          "credentialprovider.kubelet.k8s.io/v1",
		Kind:                "CredentialProviderRequest",
		Image:               "myregistry.jfrog.io/repo/image:tag",
		ServiceAccountToken: idToken,
	}
	lg := testLogger()
	svc := service.NewService(ts.Client(), lg)

	username, rtToken, err := provider.GenericAuth(svc, context.Background(), &lg, stubHost(ts), request)
	if err != nil {
		t.Fatalf("GenericAuth returned unexpected error: %v", err)
	}

	if gotRequest.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		t.Errorf("unexpected grant_type: %s", gotRequest.GrantType)
	}
	if gotRequest.SubjectTokenType != "urn:ietf:params:oauth:token-type:id_token" {
		t.Errorf("unexpected subject_token_type: %s", gotRequest.SubjectTokenType)
	}
	if gotRequest.ProviderType != "Generic OpenID Connect" {
		t.Errorf("unexpected provider_type: %s", gotRequest.ProviderType)
	}
	if gotRequest.SubjectToken != idToken {
		t.Error("subject_token did not match the id_token minted by Keycloak")
	}
	if gotRequest.ProviderName != wantProviderName {
		t.Errorf("expected provider_name %q, got %q", wantProviderName, gotRequest.ProviderName)
	}
	if gotRequest.Audience != keycloakExpectedAud {
		t.Errorf("expected audience %q, got %q", keycloakExpectedAud, gotRequest.Audience)
	}

	resp := provider.BuildCredentialProviderResponse(request, username, rtToken)
	cred, ok := resp.Auth.Registry["myregistry.jfrog.io"]
	if !ok {
		t.Fatalf("expected registry entry for myregistry.jfrog.io, got: %+v", resp.Auth.Registry)
	}
	if cred.Username != "stub-user" {
		t.Errorf("expected username %q, got %q", "stub-user", cred.Username)
	}
	if cred.Password != "keycloak-backed-stub-rt-token" {
		t.Errorf("expected password %q, got %q", "keycloak-backed-stub-rt-token", cred.Password)
	}
}
