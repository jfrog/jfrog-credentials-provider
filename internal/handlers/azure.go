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

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
	"net/http"
	"net/url"
	"strings"
)

const (
	AZURE_IDENTITY_ENDPOINT = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2023-11-01&resource=$audience&client_id=$nodepool_client_id"
	AZURE_OIDC_TOKEN_URL    = "https://login.microsoftonline.com/$tenant/oauth2/v2.0/token"
	AZURE_GRANT_TYPE        = "client_credentials"
	AZURE_SCOPE             = "$client_id/.default"
	AZURE_METADATA_URL      = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
)

type OidcResult struct {
	TokenType    string `json:"token_type"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	Token        string `json:"access_token"`
	Expiration   int    `json:"expires_in"`
}

type IdentityTokenResult struct {
	TokenType    string `json:"token_type"`
	ClientId     string `json:"client_id"`
	ExtExpiresIn string `json:"ext_expires_in"`
	Token        string `json:"access_token"`
	Expiration   string `json:"expires_in"`
	ExpirationOn string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
}

// Add JWT header structure
type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid,omitempty"`
	X5t       string `json:"x5t,omitempty"`
}

// GetAzureClusterIdentity retrieves the identity token from the kubelet managed identity
func GetAzureClusterIdentity(s *service.Service, ctx context.Context, azureAppAudience, azureNodepoolClientId string) (string, error) {
	tokenEndpoint := strings.Replace(AZURE_IDENTITY_ENDPOINT, "$audience", azureAppAudience, 1)
	tokenEndpoint = strings.Replace(tokenEndpoint, "$nodepool_client_id", azureNodepoolClientId, 1)

	tokenReq, err := http.NewRequestWithContext(ctx, "GET", tokenEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("NewRequestWithContext from azure identity token fetching failed: %v", err)
	}
	tokenReq.Header.Add("Metadata", "true")
	tokenResp, err := s.Client.Do(tokenReq)
	if err != nil {
		return "", fmt.Errorf("Calling azure identity token failed: %v" + err.Error())
	}
	defer tokenResp.Body.Close()

	// Read the response body
	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return "", err
	}
	// Check if the status code is successful
	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET identity token API call failed with status code: %d, body: %s", tokenResp.StatusCode, string(tokenBody))
	}

	s.Logger.Info("constructing identityTokenResult")
	var identityTokenResult IdentityTokenResult
	if err := json.Unmarshal(tokenBody, &identityTokenResult); err != nil {
		s.Logger.Error("get identity token response body: " + string(tokenBody))
		return "", fmt.Errorf("error unmarshaling identity token JSON: %v", err)
	}

	identityTokenAssertion := identityTokenResult.Token

	return identityTokenAssertion, nil
}

// GetAzureOIDCToken retrieves an OIDC token from Azure using managed identity
func GetAzureOIDCToken(s *service.Service, ctx context.Context,
	tenantId, clientId, azureNodepoolClientId, azureAppAudience string) (string, error) {

	identityTokenAssertion, err := GetAzureClusterIdentity(s, ctx, azureAppAudience, azureNodepoolClientId)
	if err != nil {
		s.Logger.Error("GetAzureClusterIdentity failed: " + err.Error())
		return "", err
	}

	oidc_url := strings.Replace(AZURE_OIDC_TOKEN_URL, "$tenant", tenantId, 1)

	// Create url.Values for form data
	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	data.Set("client_assertion", identityTokenAssertion)
	data.Set("grant_type", AZURE_GRANT_TYPE)
	data.Set("scope", strings.Replace(AZURE_SCOPE, "$client_id", clientId, 1))
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt")

	// Get oidc token
	req, err := http.NewRequestWithContext(ctx, "POST", oidc_url, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("NewRequestWithContext from azure oidc token failed: %v" + err.Error())
	}
	// Add headers if needed
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Calling azure oidc token failed: %v" + err.Error())
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Check if the status code is successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST oidc token API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	var oidcResult OidcResult
	if err := json.Unmarshal(body, &oidcResult); err != nil {
		s.Logger.Error("get oidcResult response body: " + string(body))
		return "", fmt.Errorf("error unmarshaling oidc token JSON: %v", err)
	}

	return oidcResult.Token, nil
}

func CheckIfAzure(s *service.Service, ctx context.Context) (bool, error) {
	s.Logger.Info("Checking if cloud provider is Azure")
	req, err := http.NewRequestWithContext(ctx, "GET", AZURE_METADATA_URL, nil)
	if err != nil {
		return false, fmt.Errorf("Error creating request to check if cloud provider is Azure: %v", err)
	}
	req.Header.Add("Metadata", "true")
	resp, err := s.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Error checking if cloud provider is Azure: %v", err)
	}
	defer resp.Body.Close()
	s.Logger.Info(fmt.Sprintf("Azure metadata server response status code: %d", resp.StatusCode))
	return (resp.StatusCode == http.StatusOK), nil
}
