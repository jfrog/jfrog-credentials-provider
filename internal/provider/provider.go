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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultSecretTTL   = "18000" // 5 hours
	defaultHTTPTimeout = 10 * time.Second
	logFileLocation    = "/var/log/jfrog-credentials-provider/jfrog-credentials-provider.log"
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

	client := newProviderHTTPClient(defaultHTTPTimeout)
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
		// if cloud_provider is not set, check if the cloud provider is AWS, Azure, or Google
		isAWS, errAWS := handlers.CheckIfAWS(svc, ctx)
		if isAWS {
			cloudProvider = utils.CloudProviderAWS
		}
		isAzure, errAzure := handlers.CheckIfAzure(svc, ctx)
		if isAzure {
			cloudProvider = utils.CloudProviderAzure
		}
		isGoogle, errGoogle := handlers.CheckIfGoogle(svc, ctx)
		if isGoogle {
			cloudProvider = utils.CloudProviderGoogle
		}

		if errAWS != nil && errAzure != nil && errGoogle != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not check if cloud provider is AWS, Azure, or Google", 1)
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
		awsEnvVariables := validateAWSEnvVariables(logs, request)
		rtUsername, rtToken = handleAWSAuth(svc, ctx, logs, awsEnvVariables, artifactoryUrl, secretTTL, request)
		return rtUsername, rtToken
	case utils.CloudProviderAzure:
		logs.Debug("Detected Azure cloud provider")
		rtUsername, rtToken = handleAzureAuth(svc, ctx, logs, artifactoryUrl, request)
		return rtUsername, rtToken
	case utils.CloudProviderGoogle:
		logs.Debug("Detected Google cloud provider")
		rtUsername, rtToken = handleGoogleAuth(svc, ctx, logs, artifactoryUrl, request)
		return rtUsername, rtToken
	default:
		logs.Exit("ERROR in JFrog Credentials provider, cloud_provider value should be either aws, azure, or google", 1)
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

func validateAWSEnvVariables(logs *logger.Logger, request utils.CredentialProviderRequest) utils.AWSEnvVariables {
	awsAuthMethod := os.Getenv("aws_auth_method")
	if awsAuthMethod == "" {
		logs.Info("awsAuthMethod not set, will default to Assume role")
		awsAuthMethod = "assume_role"
	} else if awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_role" && awsAuthMethod != "assume_external_role" {
		logs.Exit("wrong aws_auth_method value :"+awsAuthMethod, 1)
	}

	// aws_role_name is only required for assume_role / web_identity;
	// cognito_oidc and assume_external_role do not use it.
	awsRoleName := os.Getenv("aws_role_name")
	awsExternalRoleARN := os.Getenv("aws_external_role_arn")
	awsExternalRoleSessionDurationVal := os.Getenv("aws_external_role_session_duration_seconds")
	awsExternalRoleSessionDurationSeconds := 3600
	if awsExternalRoleSessionDurationVal != "" {
		duration, err := strconv.Atoi(awsExternalRoleSessionDurationVal)
		if err != nil || duration > 43200 {
			logs.Info("bad value for aws_external_role_session_duration_seconds, defaulting to 3600")
		} else {
			awsExternalRoleSessionDurationSeconds = duration
		}
	}

	if request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"] != "" {
		awsRoleName = request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"]
	} else {
		logs.Info("Service account annotation for eks.amazonaws.com/role-arn not found, using aws_role_name")
	}

	if awsRoleName == "" && awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_external_role" {
		logs.Exit("error in JFrog Credentials provider, environment var: awsRoleName configured in the plugin aws_role_name was empty", 1)
	} else if awsRoleName != "" {
		logs.Info("getting envs - " + "awsRoleName :" + awsRoleName)
	}

	if awsAuthMethod == "assume_external_role" && awsExternalRoleARN == "" {
		logs.Exit("error in JFrog Credentials provider, environment var: aws_external_role_arn must be configured when aws_auth_method is assume_external_role", 1)
	}

	jfrogOIDCProviderName := os.Getenv("jfrog_oidc_provider_name")
	secretName := os.Getenv("secret_name")
	resourceServerName := os.Getenv("resource_server_name")
	userPoolName := os.Getenv("user_pool_name")
	userPoolResourceScope := os.Getenv("user_pool_resource_scope")

	if awsAuthMethod == "cognito_oidc" {
		if jfrogOIDCProviderName == "" || secretName == "" || userPoolName == "" || resourceServerName == "" || userPoolResourceScope == "" {
			logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: jfrog_oidc_provider_name, secret_name, user_pool_name, resource_server_name, user_pool_resource_scope", 1)
		}
		logs.Info(fmt.Sprintf("getting envs - jfrogOidcProviderName: %s, secretName: %s, userPoolName: %s, resourceServerName: %s, scope: %s",
			jfrogOIDCProviderName, secretName, userPoolName, resourceServerName, userPoolResourceScope))
	}

	return utils.AWSEnvVariables{
		AWSAuthMethod:                  awsAuthMethod,
		AWSRoleName:                    awsRoleName,
		AWSExternalRoleARN:             awsExternalRoleARN,
		AWSExternalRoleDurationSeconds: awsExternalRoleSessionDurationSeconds,
		JFrogOIDCProviderName:          jfrogOIDCProviderName,
		SecretName:                     secretName,
		ResourceServerName:             resourceServerName,
		UserPoolName:                   userPoolName,
		UserPoolResourceScope:          userPoolResourceScope,
	}
}

func handleAWSAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, awsEnvVariables utils.AWSEnvVariables, artifactoryUrl, secretTTL string, request utils.CredentialProviderRequest) (string, string) {
	var rtUsername, rtToken string
	var useServiceAccount = false

	if request.ServiceAccountAnnotations["JFrogExchange"] == "true" && request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"] != "" {
		useServiceAccount = true
		awsEnvVariables.AWSRoleName = request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"]
	}

	if useServiceAccount {
		awsEnvVariables.AWSAuthMethod = "web_identity"
		logs.Info("Using web_identity aws auth method based on service account annotation")
	}

	if awsEnvVariables.AWSAuthMethod == "assume_role" || awsEnvVariables.AWSAuthMethod == "web_identity" || awsEnvVariables.AWSAuthMethod == "assume_external_role" {
		req, err := handlers.GetAWSSignedRequest(svc, ctx, request.ServiceAccountToken, awsEnvVariables)
		if err != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not get aws signed request :"+err.Error(), 1)
		}
		rtUsername, rtToken, err = handlers.ExchangeAssumedRoleArtifactoryToken(svc, ctx, req, artifactoryUrl, secretTTL)
		if err != nil {
			logs.Exit("Error in createArtifactoryToken: "+err.Error(), 1)
		}
	} else {
		token, err := handlers.GetAwsOidcToken(svc, ctx, awsEnvVariables.AWSRoleName, awsEnvVariables.SecretName, awsEnvVariables.UserPoolName, awsEnvVariables.ResourceServerName, awsEnvVariables.UserPoolResourceScope)
		if err != nil {
			logs.Exit("ERROR in JFrog Credentials provider, could not get aws oidc token :"+err.Error(), 1)
		}
		rtUsername, rtToken, err = handlers.ExchangeOidcArtifactoryToken(svc, ctx, token, artifactoryUrl, awsEnvVariables.JFrogOIDCProviderName, "")
		if err != nil {
			logs.Exit("Error in createArtifactoryToken: "+err.Error(), 1)
		}
	}
	return rtUsername, rtToken
}

func handleAzureAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, artifactoryUrl string, request utils.CredentialProviderRequest) (string, string) {

	var token string
	var err error

	// get required env variables
	azureAppClientId := utils.GetEnvs(logs, "azure_app_client_id", "")
	azureAppCloudName := utils.GetEnvs(logs, "azure_cloud_name", "AzureCloud")
	azureAppTenantId := utils.GetEnvs(logs, "azure_tenant_id", "")
	azureAppAudience := utils.GetEnvs(logs, "azure_app_audience", "")
	azureNodepoolClientId := utils.GetEnvs(logs, "azure_nodepool_client_id", "")
	jfrogOidcProviderName := utils.GetEnvs(logs, "jfrog_oidc_provider_name", "")

	if request.ServiceAccountAnnotations["JFrogExchange"] != "true" {
		if azureAppClientId == "" || azureAppTenantId == "" || azureAppAudience == "" || azureNodepoolClientId == "" || jfrogOidcProviderName == "" {
			logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: azure_app_client_id, azure_tenant_id, azure_app_audience, azure_nodepool_client_id, jfrog_oidc_provider_name", 1)
		} else {
			logs.Info(fmt.Sprintf("getting envs - azureAppClientId: %s, azureAppCloudName: %s, azureNodepoolClientId: %s, azureAppTenantId: %s, azureAppAudience: %s, jfrogOidcProviderName: %s",
				azureAppClientId, azureAppCloudName, azureNodepoolClientId, azureAppTenantId, azureAppAudience, jfrogOidcProviderName))
		}
		logs.Info("Service Account Token obtained using Node Identity (VM Service Account)")
		// Get Azure OIDC token
		token, err = handlers.GetAzureOIDCToken(svc, ctx, azureAppTenantId, azureAppClientId, azureNodepoolClientId, azureAppAudience, azureAppCloudName)
	} else {
		if azureAppClientId == "" || azureAppAudience == "" || jfrogOidcProviderName == "" {
			logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: azure_app_client_id, azure_app_audience, jfrog_oidc_provider_name", 1)
		} else {
			logs.Info(fmt.Sprintf("getting envs - azureAppClientId: %s, azureAppAudience: %s, jfrogOidcProviderName: %s",
				azureAppClientId, azureAppAudience, jfrogOidcProviderName))
		}
		logs.Info("Service Account Token obtained using Pod Identity (Kubernetes Workload Identity)")
		token = request.ServiceAccountToken
	}
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

func handleGoogleAuth(svc *service.Service, ctx context.Context, logs *logger.Logger, artifactoryUrl string, request utils.CredentialProviderRequest) (string, string) {
	// get required env variables
	googleServiceAccountEmail := utils.GetEnvs(logs, "google_service_account_email", "")
	jfrogOidcProviderAudience := utils.GetEnvs(logs, "jfrog_oidc_audience", "")
	jfrogOidcProviderName := utils.GetEnvs(logs, "jfrog_oidc_provider_name", "")
	var token string
	var err error
	if googleServiceAccountEmail == "" || jfrogOidcProviderAudience == "" || jfrogOidcProviderName == "" {
		logs.Exit("ERROR in JFrog Credentials provider, environment variables missing: google_service_account_email, jfrog_oidc_audience, jfrog_oidc_provider_name", 1)
	} else {
		logs.Info(fmt.Sprintf("getting envs - googleServiceAccountEmail: %s, jfrogOidcProviderAudience: %s, jfrogOidcProviderName: %s",
			googleServiceAccountEmail, jfrogOidcProviderAudience, jfrogOidcProviderName))
	}

	if request.ServiceAccountAnnotations["JFrogExchange"] == "true" {
		logs.Info("Service Account Token obtained using Pod Identity (Kubernetes Workload Identity)")
		token = request.ServiceAccountToken
	} else {
		// Get Google OIDC token
		logs.Info("Service Account Token obtained using Node Identity (VM Service Account)")
		token, err = handlers.GetGoogleOIDCToken(svc, ctx, googleServiceAccountEmail, jfrogOidcProviderAudience)
		if err != nil {
			logs.Exit("ERROR in GetGoogleOIDCToken :"+err.Error(), 1)
		}
	}

	// Exchange Google OIDC token with JFrog Artifactory token
	rtUsername, rtToken, err := handlers.ExchangeOidcArtifactoryToken(svc, ctx, token, artifactoryUrl, jfrogOidcProviderName, jfrogOidcProviderAudience)
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
