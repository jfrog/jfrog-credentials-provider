package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	service "jfrog-credential-provider/internal"
	handlers "jfrog-credential-provider/internal/handlers"
)

const (
	defaultSecretTTL   = "3600" // 1 hour
	defaultHTTPTimeout = 10 * time.Second
	logFileLocation    = "/var/log/jfrog-credential-provider.log" // "/var/log/jfrog-credential-provider.log" // used for debug: "jfrog-credential-provider.log"
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

func extractRepository(imagePath string) string {
	if imagePath == "" {
		return ""
	}
	parts := strings.Split(imagePath, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
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

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create or open log file
	logFile, err := os.OpenFile(logFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "[JFROG CREDENTIALS PROVIDER] ", log.Ldate|log.Ltime|log.Lshortfile)
	//logger.SetOutput(logFile)
	//logger.SetFlags(logger.Ldate | logger.Ltime | logger.Lshortfile)
	logger.Println("Running JFrog Credentials provider...")

	var request CredentialProviderRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		logger.Println("Error reading stdin:", err)
		os.Exit(1)
	}
	logger.Println("request.Image", request.Image)
	awsAuthMethod := os.Getenv("aws_auth_method")
	if awsAuthMethod == "" {
		logger.Println("awsAuthMethod not set, willdefault to Assume role")
		awsAuthMethod = "assume_role"
	} else if awsAuthMethod != "cognito_oidc" && awsAuthMethod != "assume_role" {
		logger.Println("wrong aws_auth_method value", awsAuthMethod)
		os.Exit(1)
	}

	//fmt.Fprintf(os.Stdout, "request.Image %s", request.Image)
	// the plugin requires a valid artifactory_url and aws_role_name for completing the switch between an aws token to an artifactory token
	// get rt url and aws role for performing the switch between an aws token to an artifactory token from environment
	artifactoryUrl := os.Getenv("artifactory_url")

	if artifactoryUrl == "" {
		logger.Println("ERROR in JFrog Credentials provider, environment vars configured in the plugin: artifactory_url was empty")
		os.Exit(1)
	} else {
		logger.Println("getting envs", "artifactoryUrl", artifactoryUrl)
	}

	secretTTL := os.Getenv("secret_ttl_seconds")
	if secretTTL == "" {
		secretTTL = defaultSecretTTL
	}

	// Add a reusable HTTP client
	var client = &http.Client{
		Timeout: defaultHTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    10 * time.Second,
			DisableCompression: true,
		},
	}
	svc := service.NewService(client, logger)
	// Declare variables before the if statement
	var rtUsername string
	var rtToken string

	awsRoleName := os.Getenv("aws_role_name")
	if awsRoleName == "" {
		logger.Println("ERROR in JFrog Credentials provider, environment var: awsRoleName configured in the plugin aws_role_name was empty")
		os.Exit(1)
	} else {
		logger.Println("getting envs", "awsRoleName", awsRoleName)
	}
	if awsAuthMethod == "assume_role" {
		//get aws signed request for arn role
		req, err := handlers.GetAWSSignedRequest(svc, ctx, awsRoleName)
		if err != nil {
			logger.Println("ERROR in JFrog Credentials provider, could not get aws signed request ", err)
			os.Exit(1)
		}
		//exchnage aws signed temp credentials with jfrog artifactory token
		rtUsername, rtToken, err = handlers.ExchangeAssumedRoleArtifactoryToken(svc, ctx, req, artifactoryUrl, secretTTL)
		if err != nil {
			logger.Printf("Error in createArtifactoryToken: %s", err)
			os.Exit(1)
		}

	} else {
		jfrogOidcProviderName := os.Getenv("jfrog_oidc_provider_name")
		secretName := os.Getenv("secret_name")
		resourceServerName := os.Getenv("resource_server_name")
		userPoolName := os.Getenv("user_pool_name")
		scope := os.Getenv("user_pool_resource_scope")
		//userPoolName string, resourceServerName string, scope string

		//jfrogOidcProviderAudience := os.Getenv("jfrog_oidc_audience")
		if jfrogOidcProviderName == "" || secretName == "" || userPoolName == "" || resourceServerName == "" || scope == "" {
			logger.Println("ERROR in JFrog Credentials provider, environment variables missing: jfrog_oidc_provider_name, secret_name, userPoolResourceDomain, userPoolResourceScope")
			os.Exit(1)
		} else {
			logger.Println("getting envs",
				"jfrogOidcProviderName", jfrogOidcProviderName,
				"secretName", secretName,
				"userPoolName", userPoolName,
				"resourceServerName", resourceServerName,
				"scope", scope)
		}

		// assuming this is cognito_oidc flow
		token, err := handlers.GetAwsOidcToken(svc, ctx, awsRoleName, secretName, userPoolName, resourceServerName, scope)
		if err != nil {
			logger.Println("ERROR in JFrog Credentials provider, could not get aws oidc token ", err)
			os.Exit(1)
		}
		rtUsername, rtToken, err = handlers.ExchangeOidcArtifactoryToken(svc, ctx, token, artifactoryUrl, jfrogOidcProviderName, "")
		if err != nil {
			logger.Printf("Error in createArtifactoryToken: %s", err)
			os.Exit(1)
		}

	}

	logger.Println("JFrog Username used for pull", rtUsername)
	//logger.Println("rtToken", rtToken)
	//logger.Println("request.Image", request.Image)

	response := CredentialProviderResponse{
		ApiVersion:   "credentialprovider.kubelet.k8s.io/v1",
		Kind:         "CredentialProviderResponse",
		CacheKeyType: "Registry",
		Auth: AuthSection{
			Registry: map[string]AuthCredential{
				extractRepository(request.Image): {
					Username: rtUsername,
					Password: rtToken,
				},
			},
		},
	}
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		logger.Println("Error marshaling JSON", err)
		os.Exit(1)
	}
	//logger.Println("returning response", response)

	// return response to kubelet
	os.Stdout.Write(jsonBytes)
}
