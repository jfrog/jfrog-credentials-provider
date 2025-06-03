package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/utils"
	"net/http"
)

const (
	AWS_TOKEN_ENDPOINT = "/access/api/v1/aws/token"
	OIDC_ENDPOINT      = "/access/api/v1/oidc/token"
)

// AccessResponse JFrog token response
type AwsRoleAccessResponse struct {
	TokenId     string `json:"token_id"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Username    string `json:"username"`
}

// AccessResponse JFrog token response
type OidcAccessResponse struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	IssuedTokenType string `json:"issued_token_type"`
	Username        string `json:"username"`
}
type OidcTokenRequest struct {
	GrantType        string `json:"grant_type"`
	ProviderName     string `json:"provider_name"`
	SubjectTokenType string `json:"subject_token_type"`
	SubjectToken     string `json:"subject_token"`
	ProviderType     string `json:"provider_type"`
	Audience         string `json:"audience"`
}

func ExchangeOidcArtifactoryToken(s *service.Service, ctx context.Context,
	token string, artifactoryUrl string, providerName string, audience string) (string, string, error) {
	url := fmt.Sprintf("%s%s%s", "https://", artifactoryUrl, OIDC_ENDPOINT)
	s.Logger.Info("RT oidc token url :" + url)

	requestData := OidcTokenRequest{
		GrantType:        "urn:ietf:params:oauth:grant-type:token-exchange",
		ProviderName:     providerName,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:id_token",
		SubjectToken:     token,
		ProviderType:     "Generic OpenID Connect",
		Audience:         audience,
	}
	body, err := json.Marshal(requestData)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := utils.HttpReq(s, ctx, url, body, nil)
	myResponse := &OidcAccessResponse{}
	err = json.NewDecoder(resp.Body).Decode(myResponse)
	if err != nil {
		return "", "", fmt.Errorf("error reading artifactory response")
	}
	resp.Body.Close()
	return myResponse.Username, myResponse.AccessToken, nil
}

func ExchangeAssumedRoleArtifactoryToken(s *service.Service, ctx context.Context, request *http.Request, artifactoryUrl string, secretTTL string) (string, string, error) {
	url := fmt.Sprintf("%s%s%s", "https://", artifactoryUrl, AWS_TOKEN_ENDPOINT)
	s.Logger.Info("RT token url :" + url)
	requestBody := fmt.Sprintf("%s%s%s", "{\"expires_in\": ", secretTTL, "}")
	s.Logger.Info("RT requestBody :" + requestBody)
	body := []byte(requestBody)

	resp, err := utils.HttpReq(s, ctx, url, body, request)
	if err != nil {
		return "", "", err
	}
	myResponse := &AwsRoleAccessResponse{}
	err = json.NewDecoder(resp.Body).Decode(myResponse)
	if err != nil {
		return "", "", fmt.Errorf("Error reading artifactory response")
	}
	resp.Body.Close() // Close the response body to prevent resource leaks
	return myResponse.Username, myResponse.AccessToken, nil
}
