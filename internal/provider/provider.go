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
	"context"
	"encoding/json"
	"fmt"
	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/autoupdate"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/logger"
	"jfrog-credential-provider/internal/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultSecretTTL   = "18000" // 5 hours
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
	artifactoryUrl := validateRTRequiredEnvVariables(logs)

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
		autoupdate.AutoUpdate(request, logs, client, ctx, Version)
	}()

	rtUsername, rtToken := cloudProviderAuth(svc, ctx, logs, artifactoryUrl, secretTTL, request)
	// rtUsername, rtToken := handleAWSAuth(svc, ctx, logs, awsAuthMethod, awsRoleName, artifactoryUrl, secretTTL)
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

func getCloudProvider(svc *service.Service, ctx context.Context, logs *logger.Logger) string {
	cloudProvider := utils.GetEnvs(logs, "cloud_provider", "")
	logs.Info("cloud_provider from env:" + cloudProvider)
	if cloudProvider == "" {
		// if cloud_provider is not set, check if the cloud provider is AWS or Azure
		isAWS, errAWS := handlers.CheckIfAWS(svc, ctx)
		if isAWS {
			cloudProvider = utils.CloudProviderAWS
		}
		isAzure, errAzure := handlers.CheckIfAzure(svc, ctx)
		if isAzure {
			cloudProvider = utils.CloudProviderAzure
		}

		if errAWS != nil && errAzure != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not check if cloud provider is AWS or Azure", 1)
		}
	}
	return cloudProvider
}

func cloudProviderAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, artifactoryUrl, secretTTL string, request utils.CredentialProviderRequest) (string, string) {
	var rtUsername, rtToken string

	cloudProvider := getCloudProvider(svc, ctx, logs)

	switch cloudProvider {
	case utils.CloudProviderAWS:
		logs.Debug("Detected AWS cloud provider")
		awsAuthMethod, awsRoleName := validateAWSEnvVariables(logs, request)
		rtUsername, rtToken = handleAWSAuth(svc, ctx, logs, awsAuthMethod, awsRoleName, artifactoryUrl, secretTTL, request)
		return rtUsername, rtToken
	case utils.CloudProviderAzure:
		logs.Debug("Detected Azure cloud provider")
		rtUsername, rtToken = handleAzureAuth(svc, ctx, logs, artifactoryUrl)
		return rtUsername, rtToken
	default:
		logs.Exit("ERROR in JFrog Credentials provider, cloud_provider value should be either aws or azure", 1)
	}
	return rtUsername, rtToken
}

func validateRTRequiredEnvVariables(logs *logger.Logger) string {
	artifactoryUrl := os.Getenv("artifactory_url")
	if artifactoryUrl == "" {
		logs.Exit("ERROR in JFrog Credentials provider, environment vars configured in the plugin: artifactory_url was empty", 1)
	} else {
		logs.Info("getting envs - " + "artifactoryUrl :" + artifactoryUrl)
	}
	return artifactoryUrl
}

func validateAWSEnvVariables(logs *logger.Logger, request utils.CredentialProviderRequest) (string, string) {
	awsAuthMethod := os.Getenv("aws_auth_method")
	if awsAuthMethod == "" {
		logs.Info("awsAuthMethod not set, will default to Assume role")
		awsAuthMethod = "assume_role"
	} else if awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_role" {
		logs.Exit("wrong aws_auth_method value :"+awsAuthMethod, 1)
	}

	awsRoleName := os.Getenv("aws_role_name")

	if request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"] != "" {
		awsRoleName = request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"]
	} else {
		logs.Info("Service account annotation for eks.amazonaws.com/role-arn not found, using aws_role_name")
	}

	if awsRoleName == "" {
		logs.Exit("error in JFrog Credentials provider, environment var: awsRoleName configured in the plugin aws_role_name was empty", 1)
	} else {
		logs.Info("getting envs - " + "awsRoleName :" + awsRoleName)
	}

	return awsAuthMethod, awsRoleName
}

func handleAWSAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, awsAuthMethod, awsRoleName, artifactoryUrl, secretTTL string, request utils.CredentialProviderRequest) (string, string) {
	var rtUsername, rtToken string
	var useServiceAccount = false

	if request.ServiceAccountAnnotations["JFrogExchange"] == "true" && request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"] != "" {
		useServiceAccount = true
		awsRoleName = request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"]
	}

	if useServiceAccount {
		awsAuthMethod = "web_identity"
		logs.Info("Using web_identity aws auth method based on service account annotation")
	}

	if awsAuthMethod == "assume_role" || awsAuthMethod == "web_identity" {
		req, err := handlers.GetAWSSignedRequest(svc, ctx, request.ServiceAccountToken, awsRoleName, awsAuthMethod)
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

func handleAzureAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, artifactoryUrl string) (string, string) {
	// get required env variables
	azureAppClientId := utils.GetEnvs(logs, "azure_app_client_id", "")
	azureAppTenantId := utils.GetEnvs(logs, "azure_tenant_id", "")
	azureAppAudience := utils.GetEnvs(logs, "azure_app_audience", "")
	azureNodepoolClientId := utils.GetEnvs(logs, "azure_nodepool_client_id", "")
	jfrogOidcProviderName := utils.GetEnvs(logs, "jfrog_oidc_provider_name", "")

	if azureAppClientId == "" || azureAppTenantId == "" || azureAppAudience == "" || azureNodepoolClientId == "" || jfrogOidcProviderName == "" {
		logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: azure_app_client_id, azure_tenant_id, azure_app_audience, azure_nodepool_client_id, jfrog_oidc_provider_name", 1)
	} else {
		logs.Info(fmt.Sprintf("getting envs - azureAppClientId: %s, azureNodepoolClientId: %s, azureAppTenantId: %s, azureAppAudience: %s, jfrogOidcProviderName: %s",
			azureAppClientId, azureNodepoolClientId, azureAppTenantId, azureAppAudience, jfrogOidcProviderName))
	}

	// Get Azure OIDC token
	token, err := handlers.GetAzureOIDCToken(svc, ctx, azureAppTenantId, azureAppClientId, azureNodepoolClientId, azureAppAudience)
	if err != nil {
		logs.Exit("ERROR in GetAzureOIDCToken :"+err.Error(), 1)
	}

	// Exchange Azure OIDC token with JFrog Artifactory token
	rtUsername, rtToken, err := handlers.ExchangeOidcArtifactoryToken(svc, ctx, token, artifactoryUrl, jfrogOidcProviderName, azureAppClientId)
	if err != nil {
		logs.Exit("ERROR in JFrog Credentials provider, error in createArtifactoryToken :"+err.Error(), 1)
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
