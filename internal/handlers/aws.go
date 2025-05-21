package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
	signer "jfrog-credential-provider/internal/sign"
	"log"
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

func GetAwsOidcToken(s *service.Service, ctx context.Context,
	awsRoleName string, secretName string, userPoolName string, resourceServerName string, scope string) (string, error) {
	s.Logger.Println("running aws oidc auth flow")
	// get aws token
	token, err := getToken(s, ctx)
	if err != nil {
		s.Logger.Println("Error getting aws token, ", err)
		return "", err
	}
	//log.Println("token", token)
	// get temp credentials from metadata service
	tempCredentials, err := getTempCredentials(s, ctx, token, awsRoleName)
	if err != nil {
		s.Logger.Println("GetTempCredentials returned err ", err)
		return "", err
	}
	s.Logger.Println("GetTempCredentials returned code ", tempCredentials.Code)
	if tempCredentials.Code != "Success" {
		s.Logger.Println("GetTempCredentials failed with retirned code ", tempCredentials.Code)
		return "", fmt.Errorf("GetTempCredentials failed with retirned code %s", tempCredentials.Code)
	}
	region, err := getAWSRegion(s, ctx, token)
	if err != nil {
		s.Logger.Println("Error getting AWS region:", err, "using default region *")
		region = "*"

	} else {
		s.Logger.Println("Region from SDK:", region)
	}
	config, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		s.Logger.Println("Error loading default config", err)
	}
	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(config)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		// For a list of exceptions thrown, see
		// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
		s.Logger.Printf("Error getting the secret: %s from secret manager", err.Error())
		return "", err
	}

	// Decrypts secret using the associated KMS key.
	var secretString string = *result.SecretString
	var secretBytes = []byte(secretString)
	//s.Logger.Println("secretString", secretString)
	//get secret
	var secretResult SecretResult
	if err := json.Unmarshal(secretBytes, &secretResult); err != nil {
		log.Println("get secret value from secret manager had an error, verify the secret used in the secret manager has the correct format: {\"client-secret\":\"user pool client id\",\"client-id\":\"user pool client secret\"}", err)
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	s.Logger.Println("Secret retrieved from secret manager")
	resourceServerId, userPoolResourceDomain, err := getResourceServerId(s, config, userPoolName, resourceServerName)
	if err != nil {
		return "", err
	}
	s.Logger.Println("resourceServerId", resourceServerId)
	//s.Logger.Println("secretString", secretResult.ClientSecret)
	data := url.Values{}
	data.Set("grant_type", GRANT_TYPE)
	data.Set("client_id", secretResult.ClientId)
	data.Set("client_secret", secretResult.ClientSecret)
	data.Set("scope", (resourceServerId + "/" + scope))
	//s.Logger.Println("data", data)

	oidcUrl := strings.Replace(AWS_OIDC_TOKEN_URL, "$region", region, 1)
	oidcUrl = strings.Replace(oidcUrl, "$user_pool_resource_domain", userPoolResourceDomain, 1)
	s.Logger.Println("oidcUrl", oidcUrl)

	req, err := http.NewRequestWithContext(ctx, "POST", oidcUrl, strings.NewReader(data.Encode()))
	if err != nil {
		log.Println("NewRequestWithContext from aws oidc token failed ", err)
		return "", err
	}
	// Add headers if needed
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		log.Println("Calling aws oidc token failed ", err)
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
		return "", fmt.Errorf("POST oidc token API call failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	var oidcResult AwsOidcResult
	if err := json.Unmarshal(body, &oidcResult); err != nil {
		log.Println("get oidcResult response body ", string(body))
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return oidcResult.Token, nil
}

func GetAWSSignedRequest(s *service.Service, ctx context.Context, awsRoleName string) (*http.Request, error) {
	s.Logger.Println("running aws assume role auth flow")
	// get token from metadata service
	token, err := getToken(s, ctx)
	if err != nil {
		s.Logger.Println("Error getting aws token, ", err)
		return nil, err
	}
	//log.Println("token", token)
	// get temp credentials from metadata service
	tempCredentials, err := getTempCredentials(s, ctx, token, awsRoleName)
	if err != nil {
		s.Logger.Println("GetTempCredentials returned err ", err)
		return nil, err
	}
	s.Logger.Println("GetTempCredentials returned code ", tempCredentials.Code)
	if tempCredentials.Code != "Success" {
		s.Logger.Println("GetTempCredentials failed with retirned code ", tempCredentials.Code)
		return nil, fmt.Errorf("GetTempCredentials failed with retirned code %s", tempCredentials.Code)
	}

	// get aws region, failure in this operation will not affect the request signing and we will try and sign with * region
	region, err := getAWSRegion(s, ctx, token)
	if err != nil {
		s.Logger.Println("Error getting AWS region:", err, "using default region *")
		region = "*"

	} else {
		s.Logger.Println("Region from SDK:", region)
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
		s.Logger.Printf("Error signing the request: %s", err)
		return nil, err
	}
	return req, nil
}

func getToken(s *service.Service, ctx context.Context) (string, error) {
	s.Logger.Println("TOKEN_URL", TOKEN_URL)
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
	// Create a new HTTP client
	//client := &http.Client{
	//		Timeout: time.Second * 10, // Set timeout to 10 seconds
	//	}
	s.Logger.Println("TEMP_SESSION_URL", TOKEN_URL)
	// Create a new request
	url := TEMP_SESSION_URL + awsRoleName
	s.Logger.Println("role temp session url", url)
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
		log.Println("get temp credentials response body ", string(body))
		return TempCredentials{}, fmt.Errorf("Error unmarshaling JSON: %v", err)
	}
	return tempCredentials, nil
}

func getAWSRegion(s *service.Service, ctx context.Context, token string) (string, error) {
	// := &http.Client{
	//		Timeout: time.Second * 10,
	//	}
	// Then get the region
	req, err := http.NewRequestWithContext(ctx, "GET", REGION_URL, nil)
	if err != nil {
		s.Logger.Println("Error creating request to get AWS region:", err)
		return "", err
	}

	req.Header.Add("X-aws-ec2-metadata-token", token)

	resp, err := s.Client.Do(req)
	if err != nil {
		s.Logger.Println("Error getting AWS region:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.Logger.Println("failed to get region from metadata service: ", resp.StatusCode)
		return "", fmt.Errorf("failed to get region from metadata service: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Println("Error reading response body:", err)
		return "", err
	}

	return string(body), nil
}

func getUserPoolId(s *service.Service, cfg aws.Config,
	cognitoSvc *cognitoidentityprovider.Client, userPoolName string) (string, error) {
	s.Logger.Println("getting user pool id")
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
			s.Logger.Println("failed to list user pools:", err)
			return "", err
		}
		//s.Logger.Println("result", result)
		// Check if any user pools are present
		for _, pool := range result.UserPools {
			if strings.EqualFold(*pool.Name, userPoolName) {
				s.Logger.Println("Found User Pool: ID", *pool.Id, "Name", *pool.Name)
				return *pool.Id, nil // Exit once found
			}
		}
		// Set the next token for pagination
		nextToken = result.NextToken
		if nextToken == nil {
			break // Exit the loop if there are no more pages
		}
	}

	s.Logger.Println("User Pool not found")
	return "", fmt.Errorf("user Pool not found")
}

func getResourceServerId(s *service.Service, cfg aws.Config, userPoolName string, resourceServerName string) (string, string, error) {
	s.Logger.Println("getting resource server", resourceServerName, " id for user pool", userPoolName)

	// getting resource domain from cognito
	cognitoSvc := cognitoidentityprovider.NewFromConfig(cfg)
	userPoolId, err := getUserPoolId(s, cfg, cognitoSvc, userPoolName)
	if err != nil {
		s.Logger.Println("failed to get user pool id:", err)
		return "", "", err
	}
	s.Logger.Println("user pool id", userPoolId)
	// Retrieve detailed information about the user pool
	describeInput := &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(userPoolId),
	}
	// Retrieve detailed information about the user pool
	userPoolResult, err := cognitoSvc.DescribeUserPool(context.TODO(), describeInput)
	if err != nil {
		s.Logger.Println("failed to describe user pool:", err)
		return "", "", err
	}
	// Print the User Pool Domain
	if userPoolResult.UserPool.Domain != nil {
		s.Logger.Println("User Pool Domain:", *userPoolResult.UserPool.Domain)
	} else {
		s.Logger.Println("No domain configured for this User Pool.")
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
			s.Logger.Println("failed to list user pool resource servers:", err)
			return "", "", err
		}

		// Check if any user pools are present
		for _, resourceServer := range result.ResourceServers {
			if strings.EqualFold(*resourceServer.Name, resourceServerName) {
				s.Logger.Println("Found resource server: Identifier", *resourceServer.Identifier, " Name", *resourceServer.Name)
				/*
					// Retrieve detailed information about the user pool
					describeInput := &cognitoidentityprovider.DescribeResourceServerInput{
						UserPoolId: pool.Id,
					}

					describeResult, err := cognitoSvc.DescribeUserPool(context.TODO(), describeInput)
					cognitoSvc.ListResourceServers()
					if err != nil {
						s.Logger.Println("failed to describe user pool:", err)
					}

					// Print the User Pool Domain
					if describeResult.UserPool.Domain != nil {
						s.Logger.Println("User Pool Domain:", *describeResult.UserPool.Domain)
						s.Logger.Println("User Pool ResourceName:", *describeResult.UserPool.Domain.ResourceName)
					} else {
						s.Logger.Println("No domain configured for this User Pool.")
					}
				*/
				return *resourceServer.Identifier, *userPoolResult.UserPool.Domain, nil // Exit once found
			}
		}

		// Set the next token for pagination
		nextToken = result.NextToken
		if nextToken == nil {
			break // Exit the loop if there are no more pages
		}
	}

	s.Logger.Println("Resource Server not found")
	return "", "", fmt.Errorf("resource Server not found")
}
