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

package autoupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"jfrog-credential-provider/internal/logger"
	"jfrog-credential-provider/internal/utils"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// createRequestJson creates a JSON request for the credential provider using the given Artifactory URL.
func createRequestJson(logs *logger.Logger, artifactoryUrl string, request utils.CredentialProviderRequest) ([]byte, error) {

	jsonReq := utils.CredentialProviderRequest{
		ApiVersion: "credentialutils.kubelet.k8s.io/v1",
		Kind:       "CredentialProviderRequest",
		Image:      artifactoryUrl,
	}

	if request.ServiceAccountAnnotations["JFrogExchange"] == "true" &&
		(request.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"] != "" ||
			request.ServiceAccountAnnotations["azure.workload.identity/client-id"] != "") {
		jsonReq.ServiceAccountToken = request.ServiceAccountToken
		jsonReq.ServiceAccountAnnotations = request.ServiceAccountAnnotations
	}

	jsonBytes, err := json.Marshal(jsonReq)
	if err != nil {
		logs.Error("Error marshalling JSON: " + err.Error())
		return nil, err
	}

	return jsonBytes, nil
}

// fetchArtifactoryAuth runs the new binary with the given request and environment, returning the parsed response.
func fetchArtifactoryAuth(ctx context.Context, client *http.Client, logs *logger.Logger, newBinaryPath string, kubeletPluginRequest []byte) (utils.CredentialProviderResponse, error) {
	logs.Info("Validating new binary with request JSON: " + string(kubeletPluginRequest))

	os.Chmod(newBinaryPath, 0755)
	cmd := exec.Command(newBinaryPath)
	cmd.Stdin = strings.NewReader(string(kubeletPluginRequest))

	cmd.Env = os.Environ()
	logs.Info("Validating new binary with following environment variables: " + strings.Join(cmd.Env, " "))

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	logs.Debug("Running: " + cmd.String())
	err := cmd.Run()
	if err != nil {
		if stderrBuf.Len() > 0 {
			logs.Error("Error running the new binary: " + stderrBuf.String())
			return utils.CredentialProviderResponse{}, err
		}
		logs.Error("Error running the new binary: " + err.Error())
		return utils.CredentialProviderResponse{}, err
	}

	//convert stdoutBuf to CredentialProviderResponse
	var providerResponse utils.CredentialProviderResponse
	err = json.Unmarshal(stdoutBuf.Bytes(), &providerResponse)
	if err != nil {
		logs.Error("Error decoding response from new binary: " + err.Error())
		return utils.CredentialProviderResponse{}, err
	}
	// logs.Debug("Response from new binary: " + string(stdoutBuf.Bytes()))

	if providerResponse.Auth.Registry == nil {
		logs.Error("Error: Invalid response structure, Auth.Registry is nil")
		// To provide more context on what was actually received:
		var rawResponse map[string]interface{}
		if err := json.Unmarshal(stdoutBuf.Bytes(), &rawResponse); err == nil {
			logs.Debug("Raw response auth section: " + fmt.Sprint(rawResponse["auth"]))
		}
		logs.Error("Exiting due to invalid response structure")
		return utils.CredentialProviderResponse{}, err
	}

	return providerResponse, nil
}

// validateAuthWithArtifactory validates the provider's credentials with the Artifactory server.
func validateAuthWithArtifactory(ctx context.Context, client *http.Client, logs *logger.Logger, providerResponse utils.CredentialProviderResponse, artifactoryUrl string) error {

	artifactoryUrl = fmt.Sprintf("%s%s", "https://", artifactoryUrl)
	artRequest, err := http.NewRequestWithContext(ctx, "GET", artifactoryUrl, nil)
	if err != nil {
		logs.Error("Error creating request to Artifactory: " + err.Error())
		return err
	}
	artRequest.SetBasicAuth(providerResponse.Auth.Registry[artifactoryUrl].Username, providerResponse.Auth.Registry[artifactoryUrl].Password)

	response, err := client.Do(artRequest)
	if err != nil {
		logs.Error("Error validating new binary with Artifactory: " + err.Error())
		return err
	}
	defer response.Body.Close() // Ensure response body is closed

	if response.StatusCode != http.StatusOK {
		logs.Error("Error: received non-200 response code from Artifactory: " + fmt.Sprint(response.StatusCode))
		return err
	}
	logs.Info("New binary validated successfully with Artifactory")
	return nil

}

// validateKubeletBinary validates the new binary by running it and checking its response and authenticating with target artifactory.
func validateKubeletBinary(ctx context.Context, request utils.CredentialProviderRequest, client *http.Client, logs *logger.Logger, newBinaryPath string) error {
	if _, err := os.Stat(newBinaryPath); os.IsNotExist(err) {
		logs.Error("Error: New binary does not exist at path: " + newBinaryPath)
		return err
	}

	artifactoryUrl := utils.GetEnvs(logs, "artifactory_url", "")
	if artifactoryUrl == "" {
		logs.Error("Error: artifactoryUrl environment variable is not set.")
		return fmt.Errorf("artifactoryUrl environment variable is not set")
	}

	providerRequest, err := createRequestJson(logs, artifactoryUrl, request)
	if err != nil {
		return err
	}
	providerResponse, err := fetchArtifactoryAuth(ctx, client, logs, newBinaryPath, providerRequest)
	if err != nil {
		return err
	}
	err = validateAuthWithArtifactory(ctx, client, logs, providerResponse, artifactoryUrl)
	if err != nil {
		return err
	}
	return nil
}
