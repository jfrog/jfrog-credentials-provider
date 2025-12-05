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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
	"net/http"
	"strings"
)

const (
	GOOGLE_OIDC_TOKEN_URL                    = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/$serviceaccount:generateIdToken"
	GOOGLE_SERVICE_ACCOUNT_IMPERSONATION_URL = "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/$serviceaccount/token"
	GOOGLE_METADATA_URL                      = "http://169.254.169.254/computeMetadata/v1/instance/id"
)

type GoogleOidcResult struct {
	Token string `json:"token"`
}

type GoogleOidcRequest struct {
	Audience     string `json:"audience"`
	IncludeEmail bool   `json:"includeEmail"`
}

type GoogleTokenResult struct {
	TokenType  string `json:"token_type"`
	Token      string `json:"access_token"`
	Expiration int    `json:"expires_in"`
}

func GetGoogleOIDCToken(s *service.Service, ctx context.Context,
	google_service_account_email string, audience string) (string, error) {
	token, err := getGoogleServiceAccountToken(s, ctx, google_service_account_email)
	if err != nil {
		return "", err
	}

	oidcToken, err := getServiceAccountOidcToken(s, ctx, token, google_service_account_email, audience)
	if err != nil {
		return "", err
	}
	return oidcToken, nil
}

func getGoogleServiceAccountToken(s *service.Service, ctx context.Context,
	google_service_account_email string) (string, error) {

	token_url := strings.Replace(GOOGLE_SERVICE_ACCOUNT_IMPERSONATION_URL, "$serviceaccount", google_service_account_email, 1)
	s.Logger.Info("token_url :" + token_url)

	// Get service account token
	req, err := http.NewRequestWithContext(ctx, "GET", token_url, nil)
	if err != nil {
		return "", fmt.Errorf("NewRequestWithContext from google metadata token failed: %v", err)
	}
	// Add header "Metadata-Flavor: Google"
	req.Header.Add("Metadata-Flavor", "Google")
	// Make the token impersonation request to get service account token
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Calling google get impersonation token failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	// Check if the status code is successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET token API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	var tokenResult GoogleTokenResult
	if err := json.Unmarshal(body, &tokenResult); err != nil {
		s.Logger.Error("get token response body: " + string(body))
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return tokenResult.Token, nil
}

func getServiceAccountOidcToken(s *service.Service, ctx context.Context,
	token string, google_service_account_email string, audience string) (string, error) {
	// get oidc token
	oidcUrl := strings.Replace(GOOGLE_OIDC_TOKEN_URL, "$serviceaccount", google_service_account_email, 1)
	s.Logger.Info("oidcUrl :" + oidcUrl)

	requestData := GoogleOidcRequest{
		Audience:     audience,
		IncludeEmail: true,
	}
	tokenRequestBody, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", oidcUrl, bytes.NewBuffer(tokenRequestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	oidcRequestResponse, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending google oidc token request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.Logger.Error("Could not close response body: " + err.Error())
		}
	}(oidcRequestResponse.Body)

	if oidcRequestResponse.StatusCode != http.StatusOK {
		oidcResponseBody, err := io.ReadAll(oidcRequestResponse.Body)
		if err != nil {
			s.Logger.Error("could not parse oidc response body: " + err.Error())
		}
		errMessage := fmt.Sprintf("Error getting google oidc token, token creation to %s returned %d response body %s, this might be related to missing binding of role roles/iam.serviceAccountOpenIdTokenCreator to Service Account %s",
			oidcUrl, oidcRequestResponse.StatusCode, string(oidcResponseBody), google_service_account_email)
		return "", fmt.Errorf("%s", errMessage)
	}

	myResponse := &GoogleOidcResult{}
	err = json.NewDecoder(oidcRequestResponse.Body).Decode(myResponse)
	if err != nil {
		return "", fmt.Errorf("error reading google oidc response")
	}
	return myResponse.Token, nil
}

func CheckIfGoogle(s *service.Service, ctx context.Context) (bool, error) {
	s.Logger.Info("Checking if cloud provider is Google")
	req, err := http.NewRequestWithContext(ctx, "GET", GOOGLE_METADATA_URL, nil)
	if err != nil {
		return false, fmt.Errorf("Error creating request to check if cloud provider is Google: %v", err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := s.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Error checking if cloud provider is Google: %v", err)
	}
	defer resp.Body.Close()
	s.Logger.Info(fmt.Sprintf("Google metadata server response status code: %d", resp.StatusCode))
	return (resp.StatusCode == http.StatusOK), nil
}
