package utils

import (
	"jfrog-credential-provider/internal/logger"
	"encoding/json"
	"os"
)

// CredentialProviderRequest is the request sent by the kubelet.
type CredentialProviderRequest struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Image      string `json:"image"`
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

func GetCurrentBinaryPath(logs *logger.Logger) string {
	currentBinaryPath, err := os.Executable()
	if err != nil {
		logs.Exit("Error getting current binary path: "+err.Error(), 1)
	}
	logs.Debug("Current binary path:" + currentBinaryPath)
	return currentBinaryPath
}