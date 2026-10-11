package unittests

import (
	"context"
	"encoding/json"
	"io"
	"jfrog-credential-provider/internal/handlers"
	"net/http"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type fakeSecretsManager struct {
	call func(context.Context, *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
}

func (f fakeSecretsManager) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return f.call(ctx, input)
}

type fakeCognito struct {
	listUserPools       func(*cognitoidentityprovider.ListUserPoolsInput) (*cognitoidentityprovider.ListUserPoolsOutput, error)
	describeUserPool    func(*cognitoidentityprovider.DescribeUserPoolInput) (*cognitoidentityprovider.DescribeUserPoolOutput, error)
	listResourceServers func(*cognitoidentityprovider.ListResourceServersInput) (*cognitoidentityprovider.ListResourceServersOutput, error)
}

func (f fakeCognito) ListUserPools(_ context.Context, input *cognitoidentityprovider.ListUserPoolsInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.ListUserPoolsOutput, error) {
	return f.listUserPools(input)
}

func (f fakeCognito) DescribeUserPool(_ context.Context, input *cognitoidentityprovider.DescribeUserPoolInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.DescribeUserPoolOutput, error) {
	return f.describeUserPool(input)
}

func (f fakeCognito) ListResourceServers(_ context.Context, input *cognitoidentityprovider.ListResourceServersInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.ListResourceServersOutput, error) {
	return f.listResourceServers(input)
}

func TestAWSCognitoOIDCFlowWithInjectedClients(t *testing.T) {
	imds := successfulTokenAndRegionIMDS(t)
	oidc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/token" {
			t.Errorf("unexpected OIDC request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected content type: %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		assertFormValue(t, values, "grant_type", "client_credentials")
		assertFormValue(t, values, "client_id", "test-client")
		assertFormValue(t, values, "client_secret", "test-secret")
		assertFormValue(t, values, "scope", "https://jfrog.example/token.read")
		_ = json.NewEncoder(w).Encode(handlers.AwsOidcResult{
			TokenType:  "Bearer",
			Token:      "signed-cognito-token",
			Expiration: 3600,
		})
	})
	routed := newRoutedService(t, imds, oidc)

	secrets := fakeSecretsManager{call: func(_ context.Context, input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
		if aws.ToString(input.SecretId) != "cognito-client" {
			t.Errorf("unexpected secret id: %q", aws.ToString(input.SecretId))
		}
		return &secretsmanager.GetSecretValueOutput{
			SecretString: aws.String(`{"client-id":"test-client","client-secret":"test-secret"}`),
		}, nil
	}}
	cognito := fakeCognito{
		listUserPools: func(_ *cognitoidentityprovider.ListUserPoolsInput) (*cognitoidentityprovider.ListUserPoolsOutput, error) {
			return &cognitoidentityprovider.ListUserPoolsOutput{
				UserPools: []cognitotypes.UserPoolDescriptionType{
					{Id: aws.String("pool-id"), Name: aws.String("jfrog-pool")},
				},
			}, nil
		},
		describeUserPool: func(input *cognitoidentityprovider.DescribeUserPoolInput) (*cognitoidentityprovider.DescribeUserPoolOutput, error) {
			if aws.ToString(input.UserPoolId) != "pool-id" {
				t.Errorf("unexpected pool id: %q", aws.ToString(input.UserPoolId))
			}
			return &cognitoidentityprovider.DescribeUserPoolOutput{
				UserPool: &cognitotypes.UserPoolType{Domain: aws.String("poc-domain")},
			}, nil
		},
		listResourceServers: func(input *cognitoidentityprovider.ListResourceServersInput) (*cognitoidentityprovider.ListResourceServersOutput, error) {
			if aws.ToString(input.UserPoolId) != "pool-id" {
				t.Errorf("unexpected resource-server pool id: %q", aws.ToString(input.UserPoolId))
			}
			return &cognitoidentityprovider.ListResourceServersOutput{
				ResourceServers: []cognitotypes.ResourceServerType{
					{Identifier: aws.String("https://jfrog.example"), Name: aws.String("jfrog-resource")},
				},
			}, nil
		},
	}

	token, err := handlers.GetAwsOidcTokenWithClients(
		routed.service,
		context.Background(),
		"cognito-client",
		"jfrog-pool",
		"jfrog-resource",
		"token.read",
		handlers.AWSOIDCClients{SecretsManager: secrets, Cognito: cognito},
	)
	if err != nil {
		t.Fatal(err)
	}
	if token != "signed-cognito-token" {
		t.Fatalf("unexpected OIDC token: %q", token)
	}
}

func assertFormValue(t *testing.T, values url.Values, key, expected string) {
	t.Helper()
	if got := values.Get(key); got != expected {
		t.Errorf("unexpected %s: got %q, want %q", key, got, expected)
	}
}
