package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
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
	s.Logger.Println("RT oidc token url", url)

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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	// set headers
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error sending artifactory oidc token request, Cause %s", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.Logger.Println("Could not close response body", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.Logger.Println("cold not parse artifactory body ", err)
		}
		errMessage := fmt.Sprintf("%s%s%s%d%s%s%s", "Error getting artifactory oidc token request, token creation to ", url, " returned ", resp.StatusCode, " response", "body ", body)
		return "", "", fmt.Errorf("message: %s", errMessage)
	}

	myResponse := &OidcAccessResponse{}
	err = json.NewDecoder(resp.Body).Decode(myResponse)
	if err != nil {
		return "", "", fmt.Errorf("error reading artifactory response")
	}
	return myResponse.Username, myResponse.AccessToken, nil
}

func ExchangeAssumedRoleArtifactoryToken(s *service.Service, ctx context.Context, request *http.Request, artifactoryUrl string, secretTTL string) (string, string, error) {
	url := fmt.Sprintf("%s%s%s", "https://", artifactoryUrl, AWS_TOKEN_ENDPOINT)
	s.Logger.Println("RT token url", url)
	requestBody := fmt.Sprintf("%s%s%s", "{\"expires_in\": ", secretTTL, "}")
	s.Logger.Println("RT requestBody", requestBody)

	body := []byte(requestBody)
	//client := &http.Client{
	//		Timeout: time.Second * 10,
	//	}
	// Then get the region
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	// set headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range request.Header {
		//s.Logger.Println("signed headers key=", k, " value", v[0])
		req.Header.Add(k, v[0])
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Error sending artifactory create token request, Cause %s", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.Logger.Println("Could not close response body", err)
		}
	}(resp.Body)

	//s.logger.Println("resp.Body", resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.Logger.Println("cold not parse artifactory body ", err)
		}
		errMessage := fmt.Sprintf("%s%s%s%d%s%s%s", "Error getting artifactory token request, token creation to ", url, " returned ", resp.StatusCode, " response", "body ", body)
		return "", "", fmt.Errorf("message: %s", errMessage)
	}

	myResponse := &AwsRoleAccessResponse{}
	err = json.NewDecoder(resp.Body).Decode(myResponse)
	if err != nil {
		return "", "", fmt.Errorf("Error reading artifactory response")
	}
	return myResponse.Username, myResponse.AccessToken, nil
}
