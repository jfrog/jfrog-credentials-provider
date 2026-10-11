package unittests

import (
	"context"
	"encoding/json"
	"io"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/provider"
	"jfrog-credential-provider/internal/utils"
	"net/http"
	"strings"
	"testing"
)

const (
	testAccessKey = "AKIDEXAMPLE"
	testSecretKey = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testSession   = "test-session-token"
)

func TestAWSAssumeRoleFlow(t *testing.T) {
	var exchangeCalled bool
	imds := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/latest/api/token":
			if got := r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds"); got != "600" {
				t.Errorf("unexpected IMDS token TTL: %q", got)
			}
			_, _ = io.WriteString(w, "imds-token")
		case r.Method == http.MethodGet && r.URL.Path == "/latest/meta-data/placement/region":
			assertIMDSToken(t, r)
			_, _ = io.WriteString(w, "us-east-1")
		case r.Method == http.MethodGet && r.URL.Path == "/latest/meta-data/iam/security-credentials/node-role":
			assertIMDSToken(t, r)
			_ = json.NewEncoder(w).Encode(handlers.TempCredentials{
				Code:            handlers.CREDENTIALS_SUCCESS_CODE,
				AccessKeyId:     testAccessKey,
				SecretAccessKey: testSecretKey,
				Token:           testSession,
			})
		default:
			http.NotFound(w, r)
		}
	})
	jfrog := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exchangeCalled = true
		if r.Method != http.MethodPost || r.URL.Path != handlers.AWS_TOKEN_ENDPOINT {
			t.Errorf("unexpected exchange request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("region"); got != "us-east-1" {
			t.Errorf("unexpected region query: %q", got)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "AWS4-ECDSA-P256-SHA256 ") {
			t.Errorf("missing SigV4a Authorization header: %q", got)
		}
		if got := r.Header.Get("X-Amz-Region-Set"); got != "us-east-1" {
			t.Errorf("unexpected X-Amz-Region-Set: %q", got)
		}
		if got := r.Header.Get("X-Amz-Security-Token"); got != testSession {
			t.Errorf("unexpected security token: %q", got)
		}
		if r.Header.Get("X-Amz-Date") == "" {
			t.Error("missing X-Amz-Date")
		}
		body, _ := io.ReadAll(r.Body)
		if got := string(body); got != `{"expires_in": 600}` {
			t.Errorf("unexpected exchange body: %s", got)
		}
		_ = json.NewEncoder(w).Encode(handlers.AwsRoleAccessResponse{
			Username:    "aws-user",
			AccessToken: "jfrog-token",
		})
	})

	routed := newRoutedService(t, imds, jfrog)
	t.Setenv("aws_region", "")
	signed, err := handlers.GetAWSSignedRequest(routed.service, context.Background(), "", utils.AWSEnvVariables{
		AWSAuthMethod: "assume_role",
		AWSRoleName:   "node-role",
	})
	if err != nil {
		t.Fatal(err)
	}
	username, token, err := handlers.ExchangeAssumedRoleArtifactoryToken(
		routed.service, context.Background(), signed, "artifactory.test", "600",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !exchangeCalled {
		t.Fatal("JFrog exchange endpoint was not called")
	}

	request := utils.CredentialProviderRequest{Image: "artifactory.test/docker/app:1.0"}
	response := provider.BuildCredentialProviderResponse(request, username, token)
	credential := response.Auth.Registry["artifactory.test"]
	if credential.Username != "aws-user" || credential.Password != "jfrog-token" {
		t.Fatalf("unexpected credential response: %+v", credential)
	}
}

func TestAWSAssumeRoleFailures(t *testing.T) {
	t.Run("IMDS token failure", func(t *testing.T) {
		routed := newRoutedService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}), http.NotFoundHandler())
		_, err := handlers.GetAWSSignedRequest(routed.service, context.Background(), "", utils.AWSEnvVariables{
			AWSAuthMethod: "assume_role",
			AWSRoleName:   "node-role",
		})
		if err == nil || !strings.Contains(err.Error(), "status code: 503") {
			t.Fatalf("expected IMDS status error, got %v", err)
		}
	})

	t.Run("credential response is not successful", func(t *testing.T) {
		imds := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/latest/api/token":
				_, _ = io.WriteString(w, "imds-token")
			case "/latest/meta-data/placement/region":
				_, _ = io.WriteString(w, "us-east-1")
			default:
				_ = json.NewEncoder(w).Encode(handlers.TempCredentials{Code: "Failed"})
			}
		})
		routed := newRoutedService(t, imds, http.NotFoundHandler())
		_, err := handlers.GetAWSSignedRequest(routed.service, context.Background(), "", utils.AWSEnvVariables{
			AWSAuthMethod: "assume_role",
			AWSRoleName:   "node-role",
		})
		if err == nil || !strings.Contains(err.Error(), "return code Failed") {
			t.Fatalf("expected credentials code error, got %v", err)
		}
	})

	t.Run("JFrog rejects exchange", func(t *testing.T) {
		routed := newRoutedService(t, http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		signed, err := http.NewRequest(http.MethodGet, "https://sts.amazonaws.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = handlers.ExchangeAssumedRoleArtifactoryToken(
			routed.service, context.Background(), signed, "artifactory.test", "600",
		)
		if err == nil || !strings.Contains(err.Error(), "returned 401") {
			t.Fatalf("expected JFrog 401 error, got %v", err)
		}
	})
}

func assertIMDSToken(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("X-aws-ec2-metadata-token"); got != "imds-token" {
		t.Errorf("unexpected IMDS token: %q", got)
	}
}
