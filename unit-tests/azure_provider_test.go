package unittests

import (
	"context"
	"encoding/json"
	"io"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/provider"
	"jfrog-credential-provider/internal/utils"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAzureManagedIdentityProviderFlow(t *testing.T) {
	imds := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/metadata/identity/oauth2/token" {
			t.Errorf("unexpected Azure IMDS request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Metadata") != "true" {
			t.Error("Azure IMDS request is missing Metadata: true")
		}
		if got := r.URL.Query().Get("resource"); got != "api://jfrog-app" {
			t.Errorf("unexpected Azure resource: %q", got)
		}
		if got := r.URL.Query().Get("client_id"); got != "nodepool-client" {
			t.Errorf("unexpected nodepool client id: %q", got)
		}
		_ = json.NewEncoder(w).Encode(handlers.IdentityTokenResult{Token: "managed-identity-assertion"})
	})
	azureAD := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tenant-id/oauth2/v2.0/token" {
			t.Errorf("unexpected Azure AD request: %s %s", r.Method, r.URL.Path)
		}
		values := readForm(t, r)
		assertFormValue(t, values, "client_id", "app-client")
		assertFormValue(t, values, "client_assertion", "managed-identity-assertion")
		assertFormValue(t, values, "grant_type", handlers.AZURE_GRANT_TYPE)
		assertFormValue(t, values, "scope", "app-client/.default")
		assertFormValue(t, values, "subject_token_type", "urn:ietf:params:oauth:token-type:jwt")
		_ = json.NewEncoder(w).Encode(handlers.OidcResult{Token: "azure-oidc-token"})
	})
	jfrog := oidcExchangeHandler(t, "azure-oidc-token", "azure-provider", "*@*")
	svc := newMultiRoutedService(t, map[string]http.Handler{
		"169.254.169.254":           imds,
		"login.microsoftonline.com": azureAD,
		"artifactory.test":          jfrog,
	})

	token, err := handlers.GetAzureOIDCToken(
		svc, context.Background(), "tenant-id", "app-client", "nodepool-client", "api://jfrog-app", "AzureCloud",
	)
	if err != nil {
		t.Fatal(err)
	}
	username, password, err := handlers.ExchangeOidcArtifactoryToken(
		svc, context.Background(), token, "artifactory.test", "azure-provider", "*@*",
	)
	if err != nil {
		t.Fatal(err)
	}
	response := provider.BuildCredentialProviderResponse(
		utils.CredentialProviderRequest{Image: "artifactory.test/docker/app:1.0"}, username, password,
	)
	credential := response.Auth.Registry["artifactory.test"]
	if credential.Username != "cloud-user" || credential.Password != "jfrog-cloud-token" {
		t.Fatalf("unexpected Azure credential response: %+v", credential)
	}
}

func TestAzureProviderFailures(t *testing.T) {
	t.Run("IMDS rejects identity request", func(t *testing.T) {
		svc := newMultiRoutedService(t, map[string]http.Handler{
			"169.254.169.254": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			}),
		})
		_, err := handlers.GetAzureClusterIdentity(svc, context.Background(), "api://audience", "nodepool-client")
		if err == nil || !strings.Contains(err.Error(), "status code: 503") {
			t.Fatalf("expected Azure IMDS error, got %v", err)
		}
	})

	t.Run("Azure AD rejects token request", func(t *testing.T) {
		svc := newMultiRoutedService(t, map[string]http.Handler{
			"169.254.169.254": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(handlers.IdentityTokenResult{Token: "identity-token"})
			}),
			"login.microsoftonline.com": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}),
		})
		_, err := handlers.GetAzureOIDCToken(
			svc, context.Background(), "tenant-id", "app-client", "nodepool-client", "api://audience", "AzureCloud",
		)
		if err == nil || !strings.Contains(err.Error(), "status code: 401") {
			t.Fatalf("expected Azure AD status error, got %v", err)
		}
	})
}

func readForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return values
}
