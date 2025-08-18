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
	signer "jfrog-credential-provider/internal/sign"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const (
	TOKEN_URL          = "http://169.254.169.254/latest/api/token"
	TEMP_SESSION_URL   = "http://169.254.169.254/latest/meta-data/iam/security-credentials/"
	REGION_URL         = "http://169.254.169.254/latest/meta-data/placement/region"
	GRANT_TYPE         = "client_credentials"
	AWS_OIDC_TOKEN_URL = "https://$user_pool_resource_domain.auth.$region.amazoncognito.com/oauth2/token"
	METADATA_URL       = "http://169.254.169.254/latest/meta-data/"
)

type AwsOidcResult struct {
	TokenType  string `json:"token_type"`
	Token      string `json:"access_token"`
	Expiration int    `json:"expires_in"`
}

type TempCredentials struct {
	Code            string `json:"Code"`
	LastUpdated     string `json:"LastUpdated"`
	TokenType       string `json:"Type"`
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
	Expiration      string `json:"Expiration"`
}

// Add custom error types for better error handling
type CredentialError struct {
	Operation string
	Err       error
}

func (e *CredentialError) Error() string {
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

type SecretResult struct {
	ClientSecret string `json:"client-secret"`
	ClientId     string `json:"client-id"`
}

func GetAwsOidcToken(s *service.Service, ctx context.Context, awsRoleName, secretName, userPoolName, resourceServerName, scope string) (string, error) {
	s.Logger.Info("running aws oidc auth flow")

	token, err := getToken(s, ctx)
	if err != nil {
		return "", fmt.Errorf("error getting aws token: %v", err)
	}

	region, err := getRegionOrDefault(s, ctx, token)
	if err != nil {
		return "", err
	}

	secretResult, err := getSecretFromManager(s, secretName, region)
	if err != nil {
		return "", err
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("error loading AWS config: %v", err)
	}

	resourceServerId, userPoolResourceDomain, err := getResourceServerId(s, awsConfig, userPoolName, resourceServerName)
	if err != nil {
		return "", err
	}

	return requestOidcToken(s, ctx, secretResult, resourceServerId, userPoolResourceDomain, scope, region)
}

func getRegionOrDefault(s *service.Service, ctx context.Context, token string) (string, error) {
	region, err := getAWSRegion(s, ctx, token)
	if err != nil {
		s.Logger.Info("error getting AWS region: " + err.Error() + ", using default region *")
		return "*", nil
	}
	s.Logger.Info("Region from SDK: " + region)
	return region, nil
}

func getSecretFromManager(s *service.Service, secretName, region string) (SecretResult, error) {
	config, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return SecretResult{}, fmt.Errorf("error loading default config: %v", err)
	}

	svc := secretsmanager.NewFromConfig(config)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		return SecretResult{}, fmt.Errorf("error getting the secret from secret manager: %v", err)
	}

	var secretResult SecretResult
	if err := json.Unmarshal([]byte(*result.SecretString), &secretResult); err != nil {
		return SecretResult{}, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	s.Logger.Info("Secret retrieved from secret manager")
	return secretResult, nil
}

func requestOidcToken(s *service.Service, ctx context.Context, secretResult SecretResult, resourceServerId, userPoolResourceDomain, scope, region string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", GRANT_TYPE)
	data.Set("client_id", secretResult.ClientId)
	data.Set("client_secret", secretResult.ClientSecret)
	data.Set("scope", resourceServerId+"/"+scope)

	oidcUrl := strings.Replace(AWS_OIDC_TOKEN_URL, "$region", region, 1)
	oidcUrl = strings.Replace(oidcUrl, "$user_pool_resource_domain", userPoolResourceDomain, 1)
	s.Logger.Info("oidcUrl: " + oidcUrl)

	req, err := http.NewRequestWithContext(ctx, "POST", oidcUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating OIDC token request: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling OIDC token API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading OIDC token response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC token API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var oidcResult AwsOidcResult
	if err := json.Unmarshal(body, &oidcResult); err != nil {
		return "", fmt.Errorf("error unmarshaling OIDC token response: %v", err)
	}

	return oidcResult.Token, nil
}

func GetAWSSignedRequest(s *service.Service, ctx context.Context, awsRoleName string) (*http.Request, error) {
	s.Logger.Info("running aws assume role auth flow")
	// get token from metadata service
	token, err := getToken(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting aws token, %v", err)
	}
	// get temp credentials from metadata service
	tempCredentials, err := getTempCredentials(s, ctx, token, awsRoleName)
	if err != nil {
		return nil, fmt.Errorf("GetTempCredentials returned err %v", err)
	}
	s.Logger.Info("GetTempCredentials returned code :" + tempCredentials.Code)
	if tempCredentials.Code != "Success" {
		return nil, fmt.Errorf("GetTempCredentials failed with retirned code %s", tempCredentials.Code)
	}

	// get aws region, failure in this operation will not affect the request signing and we will try and sign with * region
	region, err := getAWSRegion(s, ctx, token)
	if err != nil {
		s.Logger.Info("error getting AWS region :" + err.Error() + "using default region *")
		region = "*"

	} else {
		s.Logger.Info("Region from SDK :" + region)
	}
	// getting signed request headers for AWS STS GetCallerIdentity call
	creds := &signer.AwsCredentials{
		AccessKey:    tempCredentials.AccessKeyId,
		SecretKey:    tempCredentials.SecretAccessKey,
		RegionName:   region,
		SessionToken: tempCredentials.Token,
	}
	req, err := signer.SignV4a("GET",
		"https://sts.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15", "sts", *creds)
	if err != nil {
		return nil, fmt.Errorf("Error signing the request: %s", err)
	}
	return req, nil
}

func getToken(s *service.Service, ctx context.Context) (string, error) {
	s.Logger.Info("TOKEN_URL :" + TOKEN_URL)
	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "PUT", TOKEN_URL, nil)
	if err != nil {
		return "", &CredentialError{"create token request", err}
	}
	// Add headers if needed
	req.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", "600")
	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Check if the status code is successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET TOKEN API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func getTempCredentials(s *service.Service, ctx context.Context, token string, awsRoleName string) (TempCredentials, error) {
	s.Logger.Info("TEMP_SESSION_URL :" + TOKEN_URL)
	// Create a new request
	url := TEMP_SESSION_URL + awsRoleName
	s.Logger.Info("role temp session url :" + url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return TempCredentials{}, err
	}
	// Add headers if needed
	req.Header.Add("X-aws-ec2-metadata-token", token)
	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		return TempCredentials{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TempCredentials{}, err
	}
	// Check if the status code is successful
	if resp.StatusCode != http.StatusOK {
		return TempCredentials{}, fmt.Errorf("PUT temp session API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	var tempCredentials TempCredentials
	if err := json.Unmarshal(body, &tempCredentials); err != nil {
		return TempCredentials{}, fmt.Errorf("Error unmarshaling JSON: %v", err)
	}
	return tempCredentials, nil
}

func getAWSRegion(s *service.Service, ctx context.Context, token string) (string, error) {
	// Then get the region
	req, err := http.NewRequestWithContext(ctx, "GET", REGION_URL, nil)
	if err != nil {
		return "", fmt.Errorf("Error creating request to get AWS region: %v", err)
	}

	req.Header.Add("X-aws-ec2-metadata-token", token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error getting AWS region: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get region from metadata service: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(body), nil
}

func getUserPoolId(s *service.Service, cfg aws.Config,
	cognitoSvc *cognitoidentityprovider.Client, userPoolName string) (string, error) {
	s.Logger.Info("getting user pool id")
	// Initialize the pagination variables
	var nextToken *string
	// Loop through the user pools
	for {
		// Call ListUserPools API
		input := &cognitoidentityprovider.ListUserPoolsInput{
			MaxResults: aws.Int32(60), // Maximum number of results to return per call
			NextToken:  nextToken,
		}

		result, err := cognitoSvc.ListUserPools(context.TODO(), input)
		if err != nil {
			return "", fmt.Errorf("failed to list user pools: %v", err)
		}
		// Check if any user pools are present
		for _, pool := range result.UserPools {
			if strings.EqualFold(*pool.Name, userPoolName) {
				s.Logger.Info("Found User Pool: ID " + *pool.Id + "Name" + *pool.Name)
				return *pool.Id, nil // Exit once found
			}
		}
		// Set the next token for pagination
		nextToken = result.NextToken
		if nextToken == nil {
			break // Exit the loop if there are no more pages
		}
	}

	return "", fmt.Errorf("user Pool not found")
}

func getResourceServerId(s *service.Service, cfg aws.Config, userPoolName string, resourceServerName string) (string, string, error) {
	s.Logger.Info("getting resource server :" + resourceServerName + " id for user pool" + userPoolName)

	// getting resource domain from cognito
	cognitoSvc := cognitoidentityprovider.NewFromConfig(cfg)
	userPoolId, err := getUserPoolId(s, cfg, cognitoSvc, userPoolName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user pool id: %v", err)
	}
	s.Logger.Info("user pool id :" + userPoolId)
	// Retrieve detailed information about the user pool
	describeInput := &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(userPoolId),
	}
	// Retrieve detailed information about the user pool
	userPoolResult, err := cognitoSvc.DescribeUserPool(context.TODO(), describeInput)
	if err != nil {
		return "", "", fmt.Errorf("failed to describe user pool: %v", err)
	}
	// Print the User Pool Domain
	if userPoolResult.UserPool.Domain != nil {
		s.Logger.Info("User Pool Domain :" + *userPoolResult.UserPool.Domain)
	} else {
		return "", "", fmt.Errorf("domain was not found for user pool %s", userPoolId)
	}
	// Initialize the pagination variables
	var nextToken *string
	// Loop through the user pools
	for {
		// Call ListUserPools API
		input := &cognitoidentityprovider.ListResourceServersInput{
			UserPoolId: aws.String(userPoolId),
			MaxResults: aws.Int32(10), // Maximum number of results to return per call
			NextToken:  nextToken,
		}

		result, err := cognitoSvc.ListResourceServers(context.TODO(), input)
		if err != nil {
			return "", "", fmt.Errorf("failed to list user pool resource servers: %v", err)
		}

		// Check if any user pools are present
		for _, resourceServer := range result.ResourceServers {
			if strings.EqualFold(*resourceServer.Name, resourceServerName) {
				s.Logger.Info("Found resource server : Identifier" + *resourceServer.Identifier + " Name" + *resourceServer.Name)
				return *resourceServer.Identifier, *userPoolResult.UserPool.Domain, nil // Exit once found
			}
		}

		// Set the next token for pagination
		nextToken = result.NextToken
		if nextToken == nil {
			break // Exit the loop if there are no more pages
		}
	}
	return "", "", fmt.Errorf("resource Server not found")
}

func CheckIfAWS(s *service.Service, ctx context.Context) (bool, error) {
	s.Logger.Info("Checking if cloud provider is AWS")
	req, err := http.NewRequestWithContext(ctx, "GET", METADATA_URL, nil)
	if err != nil {
		return false, fmt.Errorf("Error creating request to check if cloud provider is AWS: %v", err)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Error checking if cloud provider is AWS: %v", err)
	}
	defer resp.Body.Close()
	s.Logger.Info(fmt.Sprintf("AWS metadata server response status code: %d", resp.StatusCode))
	return (resp.StatusCode == http.StatusOK), nil
}
