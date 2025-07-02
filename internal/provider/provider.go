package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/logger"
	"jfrog-credential-provider/internal/autoupdate"
	"jfrog-credential-provider/internal/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultSecretTTL   = "3600" // 1 hour
	defaultHTTPTimeout = 10 * time.Second
	logFileLocation    = "/var/log/jfrog-credential-provider.log" // "/var/log/jfrog-credential-provider.log" // used for debug: "jfrog-credential-provider.log"
	logPrefix          = "[JFROG CREDENTIALS PROVIDER] "
)

func extractRepository(imagePath string) string {
	if imagePath == "" {
		return ""
	}
	parts := strings.Split(imagePath, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func StartProvider(ctx context.Context, Version string) {
	logs, request := initializeLoggerAndParseRequest()
	awsAuthMethod, artifactoryUrl, awsRoleName := validateEnvVariables(logs)

	secretTTL := os.Getenv("secret_ttl_seconds")
	if secretTTL == "" {
		secretTTL = defaultSecretTTL
	}

	client := &http.Client{
		Timeout: defaultHTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    10 * time.Second,
			DisableCompression: true,
		},
	}
	svc := service.NewService(client, *logs)
	// wait group for autoupdate goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		autoupdate.AutoUpdate(logs, client, ctx, Version)
	}()


	rtUsername, rtToken := handleAWSAuth(svc, ctx, logs, awsAuthMethod, awsRoleName, artifactoryUrl, secretTTL)
	logs.Info("JFrog Username used for pull :" + rtUsername)

	generateAndOutputResponse(logs, request, rtUsername, rtToken)
	// wait until autoupdate is finished before terminating main process
	wg.Wait()
}

func initializeLoggerAndParseRequest() (*logger.Logger, utils.CredentialProviderRequest) {
	logs, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logs.Info("Running JFrog Credentials provider...")

	var request utils.CredentialProviderRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		logs.Exit("Error reading stdin :"+err.Error(), 1)
	}
	logs.Info("request.Image :" + request.Image)

	return logs, request
}

func validateEnvVariables(logs *logger.Logger) (string, string, string) {
	awsAuthMethod := os.Getenv("aws_auth_method")
	if awsAuthMethod == "" {
		logs.Info("awsAuthMethod not set, will default to Assume role")
		awsAuthMethod = "assume_role"
	} else if awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_role" {
		logs.Exit("wrong aws_auth_method value :"+awsAuthMethod, 1)
	}

	artifactoryUrl := os.Getenv("artifactory_url")
	if artifactoryUrl == "" {
		logs.Exit("ERROR in JFrog Credentials provider, environment vars configured in the plugin: artifactory_url was empty", 1)
	} else {
		logs.Info(fmt.Sprintf("getting envs", "artifactoryUrl", artifactoryUrl))
	}

	awsRoleName := os.Getenv("aws_role_name")
	if awsRoleName == "" {
		logs.Exit("error in JFrog Credentials provider, environment var: awsRoleName configured in the plugin aws_role_name was empty", 1)
	} else {
		logs.Info("getting envs - " + "awsRoleName :" + awsRoleName)
	}

	return awsAuthMethod, artifactoryUrl, awsRoleName
}

func handleAWSAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, awsAuthMethod, awsRoleName, artifactoryUrl, secretTTL string) (string, string) {
	var rtUsername, rtToken string
	if awsAuthMethod == "assume_role" {
		req, err := handlers.GetAWSSignedRequest(svc, ctx, awsRoleName)
		if err != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not get aws signed request :"+err.Error(), 1)
		}
		rtUsername, rtToken, err = handlers.ExchangeAssumedRoleArtifactoryToken(svc, ctx, req, artifactoryUrl, secretTTL)
		if err != nil {
			logs.Exit("Error in createArtifactoryToken: "+err.Error(), 1)
		}
	} else {
		jfrogOidcProviderName := os.Getenv("jfrog_oidc_provider_name")
		secretName := os.Getenv("secret_name")
		resourceServerName := os.Getenv("resource_server_name")
		userPoolName := os.Getenv("user_pool_name")
		scope := os.Getenv("user_pool_resource_scope")

		if jfrogOidcProviderName == "" || secretName == "" || userPoolName == "" || resourceServerName == "" || scope == "" {
			logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: jfrog_oidc_provider_name, secret_name, userPoolResourceDomain, userPoolResourceScope", 1)
		} else {
			logs.Info(fmt.Sprintf("getting envs",
				"jfrogOidcProviderName", jfrogOidcProviderName,
				"secretName", secretName,
				"userPoolName", userPoolName,
				"resourceServerName", resourceServerName,
				"scope", scope))
		}

		token, err := handlers.GetAwsOidcToken(svc, ctx, awsRoleName, secretName, userPoolName, resourceServerName, scope)
		if err != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not get aws oidc token :"+err.Error(), 1)
		}
		rtUsername, rtToken, err = handlers.ExchangeOidcArtifactoryToken(svc, ctx, token, artifactoryUrl, jfrogOidcProviderName, "")
		if err != nil {
			logs.Exit("Error in createArtifactoryToken: "+err.Error(), 1)
		}
	}
	return rtUsername, rtToken
}

func generateAndOutputResponse(logs *logger.Logger, request utils.CredentialProviderRequest, rtUsername, rtToken string) {
	response := utils.CredentialProviderResponse{
		ApiVersion:   "credentialprovider.kubelet.k8s.io/v1",
		Kind:         "CredentialProviderResponse",
		CacheKeyType: "Registry",
		Auth: utils.AuthSection{
			Registry: map[string]utils.AuthCredential{
				extractRepository(request.Image): {
					Username: rtUsername,
					Password: rtToken,
				},
			},
		},
	}
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		logs.Exit("Error marshaling JSON :"+err.Error(), 1)
	}

	os.Stdout.Write(jsonBytes)
}
