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

package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"jfrog-credential-provider/internal/logger"
	"math/big"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	CloudProviderAWS   = "aws"
	CloudProviderAzure = "azure"
)

// CredentialProviderRequest is the request sent by the kubelet.
type CredentialProviderRequest struct {
	ApiVersion                string            `json:"apiVersion"`
	Kind                      string            `json:"kind"`
	Image                     string            `json:"image"`
	ServiceAccountToken       string            `json:"serviceAccountToken"`
	ServiceAccountAnnotations map[string]string `json:"serviceAccountAnnotations"`
}

type AuthCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type AuthSection struct {
	Registry map[string]AuthCredential `json:"-"`
}

// CredentialProviderResponse is the response expected by the kubelet.
type CredentialProviderResponse struct {
	ApiVersion   string      `json:"apiVersion"`
	Kind         string      `json:"kind"`
	CacheKeyType string      `json:"cacheKeyType"`
	Auth         AuthSection `json:"auth"`
}

type Provider struct {
	Name                 string           `json:"name" yaml:"name"`
	MatchImages          []string         `json:"matchImages" yaml:"matchImages"`
	DefaultCacheDuration string           `json:"defaultCacheDuration" yaml:"defaultCacheDuration"`
	APIVersion           string           `json:"apiVersion" yaml:"apiVersion"`
	Env                  []EnvVar         `json:"env,omitempty" yaml:"env,omitempty"`
	TokenAttributes      *TokenAttributes `json:"tokenAttributes,omitempty" yaml:"tokenAttributes,omitempty"`
}

type TokenAttributes struct {
	ServiceAccountTokenAudience          string   `json:"serviceAccountTokenAudience,omitempty" yaml:"serviceAccountTokenAudience,omitempty"`
	CacheType                            string   `json:"cacheType,omitempty" yaml:"cacheType,omitempty"`
	RequireServiceAccount                bool     `json:"requireServiceAccount" yaml:"requireServiceAccount"`
	RequiredServiceAccountAnnotationKeys []string `json:"requiredServiceAccountAnnotationKeys,omitempty" yaml:"requiredServiceAccountAnnotationKeys,omitempty"`
	OptionalServiceAccountAnnotationKeys []string `json:"optionalServiceAccountAnnotationKeys,omitempty" yaml:"optionalServiceAccountAnnotationKeys,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type CredentialProviderConfig struct {
	APIVersion string     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string     `json:"kind" yaml:"kind"`
	Providers  []Provider `json:"providers" yaml:"providers"`
}

func (a AuthSection) MarshalJSON() ([]byte, error) {
	// Create a map to hold our custom JSON structure
	m := map[string]interface{}{}
	// Add all registry credentials directly to the map
	for k, v := range a.Registry {
		m[k] = v
	}
	return json.Marshal(m)
}

func (a *AuthSection) UnmarshalJSON(data []byte) error {
	var m map[string]AuthCredential
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	a.Registry = m
	return nil
}

func GetEnvs(logs *logger.Logger, key, fallback string) string {
	logs.Info("Fetching environment variable:" + key)
	if value, ok := os.LookupEnv(key); ok {
		logs.Debug("Found environment variable:" + key + "=" + value)
		return value
	}
	return fallback
}

func GetEnvsBool(logs *logger.Logger, key string, fallback bool) bool {
	valueString := GetEnvs(logs, key, "")
	if value, err := strconv.ParseBool(valueString); err == nil {
		return value
	}
	return fallback
}

func GetCurrentBinaryPath(logs *logger.Logger) string {
	currentBinaryPath, err := os.Executable()
	if err != nil {
		logs.Exit("Error getting current binary path: "+err.Error(), 1)
	}
	logs.Debug("Current binary path:" + currentBinaryPath)
	return currentBinaryPath
}

func ReadFile(filePath string, isYaml bool, v interface{}, cloudProvider string) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse the file based on the format
	if isYaml {
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to parse YAML file %s: %w", filePath, err)
		}
	} else {
		if err := json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to parse JSON file %s: %w", filePath, err)
		}
	}
	// Validate the parsed data
	if config, ok := v.(*CredentialProviderConfig); ok {
		if err := ValidateProviderConfig(config.Providers); err != nil {
			return fmt.Errorf("validation failed for file %s: %w", filePath, err)
		}
	} else if provider, ok := v.(*Provider); ok {
		if err := ValidateJfrogProviderConfig(*provider, cloudProvider); err != nil {
			return fmt.Errorf("validation failed for file %s: %w", filePath, err)
		}
	} else {
		return fmt.Errorf("invalid type for validation, expected CredentialProviderConfig")
	}

	return nil
}

func MergeFiles(file1, file2, outputFile string, isYaml, dryRun bool, logs *logger.Logger, cloudProvider string) error {
	// Read and parse the first file

	var config CredentialProviderConfig

	// cloudProvider is being passed to be used by the validateJfrogProviderConfig function
	if err := ReadFile(file1, isYaml, &config, cloudProvider); err != nil {
		return err
	}

	// Read and parse the second file
	var provider Provider
	if err := ReadFile(file2, isYaml, &provider, cloudProvider); err != nil {
		return err
	}

	providerExist := checkProviderExists(config.Providers, provider)
	// Add the new provider to the config
	if !providerExist {
		config.Providers = append(config.Providers, provider)

		// Write the merged config to the output file
		var mergedData []byte
		var err error
		if isYaml {
			mergedData, err = yaml.Marshal(&config)
		} else {
			mergedData, err = json.MarshalIndent(&config, "", "  ")
		}
		if err != nil {
			return fmt.Errorf("failed to marshal merged config: %w", err)
		}

		if dryRun {
			logs.Info("Dry run: Below config would be written to " + outputFile)
			logs.Info(string(mergedData))
			return nil
		}

		if err := os.WriteFile(outputFile, mergedData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		logs.Info("Merged config written to " + outputFile)
	} else {
		logs.Info("Provider with same artifactory url already exists. Skipping addition.")
	}
	return nil
}

func checkProviderExists(providers []Provider, newProvider Provider) bool {
	for _, provider := range providers {
		if GetEnvVarValue(provider.Env, "artifactory_url") == GetEnvVarValue(newProvider.Env, "artifactory_url") {
			return true
		}
	}
	return false
}

func ValidateProviderConfig(config []Provider) error {

	for _, provider := range config {
		if provider.Name == "" || len(provider.MatchImages) == 0 || provider.DefaultCacheDuration == "" {
			return fmt.Errorf("missing required fields in provider '%s'", provider.Name)
		}
	}

	return nil
}

func ValidateJfrogProviderConfig(config Provider, cloudProvider string) error {

	if config.Name == "" || len(config.MatchImages) == 0 || config.DefaultCacheDuration == "" {
		return fmt.Errorf("missing required fields in provider : name, matchImages, or defaultCacheDuration ]")
	}

	if GetEnvVarValue(config.Env, "artifactory_url") == "" {
		return fmt.Errorf("missing required fields in provider: artifactory_url")
	}

	switch cloudProvider {
	case CloudProviderAWS:
		awsAuthMethod := GetEnvVarValue(config.Env, "aws_auth_method")
		if awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_role" {
			return fmt.Errorf("aws_auth_method can only be set as cognito_oidc or assume_role however the current value is :" + awsAuthMethod)
		}

		if awsAuthMethod == "cognito_oidc" {
			if GetEnvVarValue(config.Env, "jfrog_oidc_provider_name") == "" || GetEnvVarValue(config.Env, "secret_name") == "" || GetEnvVarValue(config.Env, "user_pool_name") == "" || GetEnvVarValue(config.Env, "resource_server_name") == "" || GetEnvVarValue(config.Env, "user_pool_resource_scope") == "" {
				return fmt.Errorf("aws_auth_method as cognito_oidc has one or more missing environment variables: jfrog_oidc_provider_name, secret_name, userPoolResourceDomain, userPoolResourceScope")
			}
		}

	case CloudProviderAzure:
		if GetEnvVarValue(config.Env, "azure_app_client_id") == "" || GetEnvVarValue(config.Env, "azure_tenant_id") == "" || GetEnvVarValue(config.Env, "azure_app_audience") == "" || GetEnvVarValue(config.Env, "azure_nodepool_client_id") == "" || GetEnvVarValue(config.Env, "jfrog_oidc_provider_name") == "" {
			return fmt.Errorf("ERROR in JFrog Credentials provider, environment variables missing: azure_app_client_id, azure_tenant_id, azure_app_audience, azure_nodepool_client_id, jfrog_oidc_provider_name")
		}
	}
	return nil
}

func GetEnvVarValue(envVars []EnvVar, name string) string {
	for _, env := range envVars {
		if env.Name == name {
			return env.Value
		}
	}
	return ""
}

func RandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)

	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		result[i] = letters[num.Int64()]
	}

	return string(result)
}
