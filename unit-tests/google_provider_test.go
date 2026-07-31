package unittests

import (
	"context"
	"encoding/json"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/provider"
	"jfrog-credential-provider/internal/utils"
	"net/http"
	"strings"
	"testing"
)

func TestGoogleServiceAccountProviderFlow(t *testing.T) {
	const serviceAccount = "puller@example.iam.gserviceaccount.com"
	metadata := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/computeMetadata/v1/instance/service-accounts/" + serviceAccount + "/token"
		if r.Method != http.MethodGet || r.URL.Path != expectedPath {
			t.Errorf("unexpected Google metadata request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Metadata-Flavor") != "Google" {
			t.Error("Google metadata request is missing Metadata-Flavor: Google")
		}
		_ = json.NewEncoder(w).Encode(handlers.GoogleTokenResult{
			TokenType:  "Bearer",
			Token:      "google-access-token",
			Expiration: 3600,
		})
	})
	iamCredentials := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/projects/-/serviceAccounts/" + serviceAccount + ":generateIdToken"
		if r.Method != http.MethodPost || r.URL.Path != expectedPath {
			t.Errorf("unexpected IAM Credentials request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer google-access-token" {
			t.Errorf("unexpected Google authorization header: %q", got)
		}
		var request handlers.GoogleOidcRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Audience != "jfrog-audience" || !request.IncludeEmail {
			t.Errorf("unexpected Google OIDC request: %+v", request)
		}
		_ = json.NewEncoder(w).Encode(handlers.GoogleOidcResult{Token: "google-oidc-token"})
	})
	jfrog := oidcExchangeHandler(t, "google-oidc-token", "google-provider", "jfrog-audience")
	svc := newMultiRoutedService(t, map[string]http.Handler{
		"169.254.169.254":               metadata,
		"iamcredentials.googleapis.com": iamCredentials,
		"artifactory.test":              jfrog,
	})

	token, err := handlers.GetGoogleOIDCToken(svc, context.Background(), serviceAccount, "jfrog-audience")
	if err != nil {
		t.Fatal(err)
	}
	username, password, err := handlers.ExchangeOidcArtifactoryToken(
		svc, context.Background(), token, "artifactory.test", "google-provider", "jfrog-audience",
	)
	if err != nil {
		t.Fatal(err)
	}
	response := provider.BuildCredentialProviderResponse(
		utils.CredentialProviderRequest{Image: "artifactory.test/docker/app:1.0"}, username, password,
	)
	credential := response.Auth.Registry["artifactory.test"]
	if credential.Username != "cloud-user" || credential.Password != "jfrog-cloud-token" {
		t.Fatalf("unexpected Google credential response: %+v", credential)
	}
}

func TestGoogleProviderFailures(t *testing.T) {
	t.Run("metadata rejects access token request", func(t *testing.T) {
		svc := newMultiRoutedService(t, map[string]http.Handler{
			"169.254.169.254": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			}),
		})
		_, err := handlers.GetGoogleOIDCToken(
			svc, context.Background(), "puller@example.iam.gserviceaccount.com", "jfrog-audience",
		)
		if err == nil || !strings.Contains(err.Error(), "status code: 503") {
			t.Fatalf("expected Google metadata error, got %v", err)
		}
	})

	t.Run("IAM Credentials rejects ID token request", func(t *testing.T) {
		svc := newMultiRoutedService(t, map[string]http.Handler{
			"169.254.169.254": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(handlers.GoogleTokenResult{Token: "google-access-token"})
			}),
			"iamcredentials.googleapis.com": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "forbidden", http.StatusForbidden)
			}),
		})
		_, err := handlers.GetGoogleOIDCToken(
			svc, context.Background(), "puller@example.iam.gserviceaccount.com", "jfrog-audience",
		)
		if err == nil || !strings.Contains(err.Error(), "returned 403") {
			t.Fatalf("expected IAM Credentials status error, got %v", err)
		}
	})
}
