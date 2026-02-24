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
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultProviderHome = "/etc/eks/image-credential-provider/"
	jfrogConfigFile     = "jfrog-provider"
	finalConfigFile     = "config"

	jfrogProviderIdentifier = "jfrog"
	backupSuffixOriginal    = ".backup" // pristine pre-JFrog config
	backupSuffixJfrog       = ".jfrog"  // last working config with JFrog
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

// resolveConfigPath builds the full config file path from provider home,
// config file name, and format.
func resolveConfigPath(isYaml bool, providerHome string, providerConfigFileName string) string {
	if isYaml {
		return providerHome + providerConfigFileName + ".yaml"
	}
	return providerHome + providerConfigFileName + ".json"
}

// configContainsJfrogProvider unmarshals the config (without validation) and
// checks if any provider name contains "jfrog". This is safer than a raw
// strings.Contains on the file contents, which could false-positive on
// comments, URLs, or unrelated fields.
func configContainsJfrogProvider(configPath string, isYaml bool) (bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}
	var config utils.CredentialProviderConfig
	if isYaml {
		err = yaml.Unmarshal(data, &config)
	} else {
		err = json.Unmarshal(data, &config)
	}
	if err != nil {
		return false, err
	}
	for _, p := range config.Providers {
		if strings.Contains(p.Name, jfrogProviderIdentifier) {
			return true, nil
		}
	}
	return false, nil
}

// BackupConfig is config-aware: it reads the kubelet credential provider config,
// checks whether the JFrog provider already exists, and decides which backup to create:
//   - JFrog NOT in config (first install) --> saves to <config>.backup
//   - JFrog IS in config (upgrade / post-success) --> saves to <config>.jfrog
func BackupConfig(isYaml bool, providerHome string, providerConfigFileName string, isKubelethWatcher bool, logs *logger.Logger) error {
	configPath := resolveConfigPath(isYaml, providerHome, providerConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Decide suffix by parsing the config struct and checking provider names
	suffix := backupSuffixOriginal
	hasJfrog, err := configContainsJfrogProvider(configPath, isYaml)
	if err != nil {
		logs.Info("Warning: could not parse config to check for JFrog provider: " + err.Error())
		// Default to .backup if we can't determine
	} else if !hasJfrog && !isKubelethWatcher {
		// to backup during merge if JFrog provider is not in the config, and is not triggered by the watcher
		suffix = backupSuffixOriginal
	} else if hasJfrog && isKubelethWatcher {
		suffix = backupSuffixJfrog
	} else {
		logs.Info("jfrog provider is in the config, and is not triggered by the watcher")
		return nil
	}

	backupPath := configPath + suffix
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup to %s: %w", backupPath, err)
	}
	logs.Info("Config backed up to " + backupPath)
	return nil
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
			"JFROG_OIDC_PROVIDER_NAME": os.Getenv("JFROG_OIDC_PROVIDER_NAME"),
			"SECRET_NAME":              os.Getenv("SECRET_NAME"),
			"USER_POOL_NAME":           os.Getenv("USER_POOL_NAME"),
			"RESOURCE_SERVER_NAME":     os.Getenv("RESOURCE_SERVER_NAME"),
			"USER_POOL_RESOURCE_SCOPE": os.Getenv("USER_POOL_RESOURCE_SCOPE"),
			"ARTIFACTORY_USER":         os.Getenv("ARTIFACTORY_USER"),
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

	// Before merge, backup the current config (config-aware: picks .backup or .jfrog)
	if err := BackupConfig(isYaml, providerHome, providerConfigFileName, false, logs); err != nil {
		logs.Info("Warning: could not create pre-merge backup: " + err.Error())
		// Non-fatal: continue with merge even if backup fails
	}

	err = utils.MergeFiles(finalConfigFileName, jfrogConfigFileName, finalConfigFileName, isYaml, dryRun, logs, cloudProvider)
	if err != nil {
		logs.Exit(err, 1)
	}
}

// WatchKubelet monitors kubelet health for the given timeout (in seconds).
// It waits an initial grace period for kubelet restart to begin, then polls
// systemctl is-active kubelet every 5 seconds. If kubelet is not active,
// it triggers a rollback to the most recent backup.
func WatchKubelet(isYaml bool, providerHome string, providerConfigFileName string, timeout int, logs *logger.Logger) {
	configPath := resolveConfigPath(isYaml, providerHome, providerConfigFileName)

	interval := 5
	elapsed := 0
	// Initial grace period to allow kubelet restart to begin
	gracePeriod := 5
	logs.Info(fmt.Sprintf("Watcher: waiting %d seconds grace period before monitoring kubelet", gracePeriod))
	time.Sleep(time.Duration(gracePeriod) * time.Second)
	elapsed += gracePeriod

	for elapsed < timeout {
		out, _ := exec.Command("systemctl", "is-active", "kubelet").Output()
		status := strings.TrimSpace(string(out))
		if status != "active" {
			logs.Error("Kubelet is not active (status: " + status + "), triggering rollback")
			rollbackConfig(configPath, logs)
			return
		}
		logs.Info(fmt.Sprintf("Watcher: kubelet active (%d/%d seconds elapsed)", elapsed, timeout))
		time.Sleep(time.Duration(interval) * time.Second)
		elapsed += interval
	}
	logs.Info("Watcher: kubelet healthy for full timeout period")
	// create a backup of the config
	if err := BackupConfig(isYaml, providerHome, providerConfigFileName, true, logs); err != nil {
		logs.Error("Failed to create post-success backup of kubelet config: " + err.Error())
	}
	logs.Info("Watcher: created post-success backup of kubelet config")
}

// rollbackConfig restores the kubelet credential provider config from the
// best available backup. Priority:
//  1. .jfrog  -- last known working config with JFrog (keeps JFrog working)
//  2. .backup -- pristine pre-JFrog config (removes JFrog entirely)
func rollbackConfig(configPath string, logs *logger.Logger) {
	jfrogBackup := configPath + backupSuffixJfrog
	originalBackup := configPath + backupSuffixOriginal
	logText := ""

	var restoreFrom string
	if _, err := os.Stat(jfrogBackup); err == nil {
		restoreFrom = jfrogBackup
		logText = "Rolled back to your previous working config with JFrog"
	} else if _, err := os.Stat(originalBackup); err == nil {
		restoreFrom = originalBackup
		logText = "Jfrog Credential Provider has been removed from your cluster due to an error, please check the config and retry."
	} else {
		logs.Error("No backup files found, cannot rollback")
		return
	}

	logs.Info("Rolling back kubelet config from " + restoreFrom)
	data, err := os.ReadFile(restoreFrom)
	if err != nil {
		logs.Error("Failed to read backup: " + err.Error())
		return
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logs.Error("Failed to restore config: " + err.Error())
		return
	}
	logs.Info("Restored config from " + restoreFrom)
	logs.Info("Kubelet was restarting continously, so we rolled back to the most recent backup.")
	logs.Error(logText)

	// will wait for kubelet to restart on its own
}
