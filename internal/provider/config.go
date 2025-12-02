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
	"jfrog-credential-provider/internal/logger"
	"jfrog-credential-provider/internal/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultProviderHome = "/etc/eks/image-credential-provider/"
	jfrogConfigFile     = "jfrog-provider"
	finalConfigFile     = "config"
)

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProviderConfig struct {
	Name                 string   `json:"name"`
	MatchImages          []string `json:"matchImages"`
	DefaultCacheDuration string   `json:"defaultCacheDuration"`
	APIVersion           string   `json:"apiVersion"`
	Env                  []EnvVar `json:"env"`
}

func ProcessProviderConfigEnvs(providerHome string, providerConfigFileName string) (string, string) {
	if providerConfigFileName == "" {
		providerConfigFileName = finalConfigFile
	}

	if providerHome == "" {
		providerHome = defaultProviderHome
	}

	// if providerConfigFileName contains extensions (.yaml, .yml, .json), remove them
	providerConfigFileName = strings.TrimSuffix(providerConfigFileName, ".yaml")
	providerConfigFileName = strings.TrimSuffix(providerConfigFileName, ".yml")
	providerConfigFileName = strings.TrimSuffix(providerConfigFileName, ".json")

	// if trailing slash is not present, add it
	if !strings.HasSuffix(providerHome, "/") {
		providerHome = providerHome + "/"
	}

	return providerHome, providerConfigFileName
}

func CreateProviderConfigFromEnv(isYaml bool, providerHome string, providerConfigFileName string) {
	logs, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	envVars := []EnvVar{}
	addEnvVar := func(name, value string) {
		if value != "" {
			envVars = append(envVars, EnvVar{Name: name, Value: value})
		}
	}

	addEnvVar("artifactory_url", os.Getenv("ARTIFACTORY_URL"))
	addEnvVar("artifactory_user", os.Getenv("ARTIFACTORY_USER"))
	addEnvVar("aws_auth_method", os.Getenv("AWS_AUTH_METHOD"))
	addEnvVar("aws_role_name", os.Getenv("AWS_ROLE_NAME"))
	addEnvVar("secret_name", os.Getenv("SECRET_NAME"))
	addEnvVar("secret_ttl_seconds", os.Getenv("SECRET_TTL_SECONDS"))
	addEnvVar("jfrog_oidc_provider_name", os.Getenv("JFROG_OIDC_PROVIDER_NAME"))
	addEnvVar("user_pool_name", os.Getenv("USER_POOL_NAME"))
	addEnvVar("user_pool_resource_scope", os.Getenv("USER_POOL_RESOURCE_SCOPE"))
	addEnvVar("resource_server_name", os.Getenv("RESOURCE_SERVER_NAME"))
	addEnvVar("google_service_account_email", os.Getenv("GOOGLE_SERVICE_ACCOUNT_EMAIL"))
	addEnvVar("jfrog_oidc_audience", os.Getenv("JFROG_OIDC_AUDIENCE"))

	// Read MatchImages and DefaultCacheDuration from environment variables
	matchImages := os.Getenv("MATCH_IMAGES")
	if matchImages == "" {
		matchImages = "*.jfrog.io" // Default value
	}
	defaultCacheDuration := os.Getenv("DEFAULT_CACHE_DURATION")
	if defaultCacheDuration == "" {
		defaultCacheDuration = "4h" // Default value
	}

	// Validate conditions
	authMethod := os.Getenv("AWS_AUTH_METHOD")
	if authMethod == "assume_role" || authMethod == "" {
		iamRoleArn := os.Getenv("IAM_ROLE_ARN")
		if iamRoleArn == "" {
			logs.Exit("if authentication_method is 'assume_role', then 'IAM_ROLE_ARN' must be provided and be a non-empty string", 1)
		}
	}
	if authMethod == "cognito_oidc" {
		requiredVars := map[string]string{
			"JFROG_OIDC_PROVIDER_NAME":                   os.Getenv("JFROG_OIDC_PROVIDER_NAME"),
			"AWS_COGNITO_USER_POOL_SECRET_NAME":          os.Getenv("AWS_COGNITO_USER_POOL_SECRET_NAME"),
			"AWS_COGNITO_USER_POOL_NAME":                 os.Getenv("AWS_COGNITO_USER_POOL_NAME"),
			"AWS_COGNITO_USER_POOL_DOMAIN_NAME":          os.Getenv("AWS_COGNITO_USER_POOL_DOMAIN_NAME"),
			"AWS_COGNITO_RESOURCE_SERVER_NAME":           os.Getenv("AWS_COGNITO_RESOURCE_SERVER_NAME"),
			"ARTIFACTORY_OIDC_IDENTITY_MAPPING_USERNAME": os.Getenv("ARTIFACTORY_OIDC_IDENTITY_MAPPING_USERNAME"),
			"ARTIFACTORY_USER":                           os.Getenv("ARTIFACTORY_USER"),
		}
		for key, value := range requiredVars {
			if value == "" {
				logs.Exit(fmt.Sprintf("if authentication_method is 'cognito_oidc', then '%s' must be provided and be a non-empty string", key), 1)
			}
		}
	}

	// Create the provider config
	providerConfig := ProviderConfig{
		Name:                 "jfrog-credential-provider",
		MatchImages:          []string{matchImages},
		DefaultCacheDuration: defaultCacheDuration,
		APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
		Env:                  envVars,
	}

	// Marshal the config to JSON
	data, err := json.MarshalIndent(providerConfig, "", "  ")
	if err != nil {
		logs.Exit(fmt.Sprintf("failed to marshal provider config: %v", err), 1)
	}

	var jfrogConfigFileName string
	if isYaml {
		jfrogConfigFileName = providerHome + providerConfigFileName + ".yaml"
	} else {
		jfrogConfigFileName = providerHome + providerConfigFileName + ".json"
	}
	// Write the JSON to the output file
	if err := os.WriteFile(jfrogConfigFileName, data, 0644); err != nil {
		logs.Exit(fmt.Sprintf("failed to write provider config to file: %v", err), 1)
	}

	logs.Info(fmt.Sprintf("Provider config written to %s\n", jfrogConfigFileName))

}

func MergeConfig(dryRun, isYaml bool, providerHome string, providerConfigFileName string) {
	logs, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	var jfrogConfigFileName, finalConfigFileName string
	if isYaml {
		jfrogConfigFileName = providerHome + jfrogConfigFile + ".yaml"
		finalConfigFileName = providerHome + providerConfigFileName + ".yaml"
	} else {
		jfrogConfigFileName = providerHome + jfrogConfigFile + ".json"
		finalConfigFileName = providerHome + providerConfigFileName + ".json"
	}
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    10 * time.Second,
			DisableCompression: true,
		},
	}
	svc := service.NewService(client, *logs)
	ctx := context.Background()
	cloudProvider := getCloudProvider(svc, ctx, logs)

	err = utils.MergeFiles(finalConfigFileName, jfrogConfigFileName, finalConfigFileName, isYaml, dryRun, logs, cloudProvider)
	if err != nil {
		logs.Exit(err, 1)
	}
}
