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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jfrog-credential-provider/internal/logger"
	"net/http"
	"os"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

// addVPrefix ensures the version tag starts with 'v'. If not, it prepends 'v'.
func addVPrefix(logs *logger.Logger, versionTag string) string {
	if !strings.HasPrefix(versionTag, "v") {
		logs.Info("Release tag " + versionTag + " is missing 'v' prefix. Prepending 'v'." + versionTag)
		versionTag = "v" + versionTag
		return versionTag
	}
	return versionTag
}

// fetchLatestVersionTag fetches the latest available release tag from the JFrog plugin releases URL and compares it to the current version. Outputs a new version tag if a newer version is available.
func fetchLatestVersionTag(ctx context.Context, client *http.Client, currentVersion string, jfrogPluginReleasesUrl string, logs *logger.Logger) (string, error) {
	// jfrogPluginReleasesUrl = jfrogPluginReleasesUrl + "/releases"
	request, err := http.NewRequestWithContext(ctx, "GET", jfrogPluginReleasesUrl, nil)
	logs.Info("Fetching latest version from: " + jfrogPluginReleasesUrl)
	if err != nil {
		logs.Error("Error creating request: " + err.Error())
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		logs.Error("Error sending request: " + err.Error())
		return "", err
	}

	logs.Debug("Response status code: " + fmt.Sprint(response.StatusCode))

	if response.StatusCode != http.StatusOK {
		logs.Error("Error: received non-200 response code: " + fmt.Sprint(response.StatusCode))
	}

	// var releases []BinaryReleasesList
	var releaseData map[string]interface{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logs.Error("Error reading response body: " + err.Error())
		return "", err
	}

	// err = json.Unmarshal(body, &releases)
	err = json.Unmarshal(body, &releaseData)
	if err != nil {
		logs.Error("Error decoding response: " + err.Error())
		return "", err
	}

	var latestVersionTag = addVPrefix(logs, currentVersion)
	if !semver.IsValid(latestVersionTag) {
		logs.Error("Current Version " + latestVersionTag + " isn't valid! Exiting")
		return "", fmt.Errorf("invalid current version: %s", latestVersionTag)
	}

	logs.Info("Current version: " + latestVersionTag)

	releases, ok := releaseData["children"].([]interface{})
	if !ok {
		logs.Error("Error: children is not a slice")
		return "", fmt.Errorf("invalid response structure, expected 'children' to be a slice")
	}
	for _, release := range releases {
		releaseMap, ok := release.(map[string]interface{})
		if !ok {
			logs.Error("Error: release is not a map")
			continue
		}
		releaseName, ok := releaseMap["uri"].(string)
		if !ok {
			logs.Error("Error: uri is not a string")
			continue
		}
		releaseName = strings.TrimPrefix(releaseName, "/")
		releaseName = addVPrefix(logs, releaseName)
		logs.Debug("Checking if " + releaseName + " version is latest")
		if semver.IsValid(releaseName) && semver.Compare(latestVersionTag, releaseName) < 0 {
			logs.Debug("Found newer version: " + releaseName)
			latestVersionTag = releaseName
		}
	}

	if latestVersionTag == addVPrefix(logs, currentVersion) {
		logs.Info("No newer version available")
		return "", nil
	}
	return latestVersionTag, nil
}

// downloadReleaseArtifacts downloads a release artifact from the given URL to the specified file path.
func downloadReleaseArtifacts(ctx context.Context, logs *logger.Logger, client *http.Client, filepath string, downloadUrl string) error {
	logs.Info("Downloading release artifacts from: " + downloadUrl + " to " + filepath)
	if _, err := os.Stat(filepath); err == nil {
		logs.Info("File " + filepath + " exists. Deleting...")
		err := os.Remove(filepath)
		if err != nil {
			logs.Error("Failed to delete existing file: " + err.Error())
		}
		logs.Info("File " + filepath + " deleted successfully.")
	}

	out, err := os.Create(filepath)
	if err != nil {
		logs.Error("Failed to create new file: " + err.Error())
	}
	defer out.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		logs.Error("Received some error while forming get request: " + err.Error())
		return err
	}
	// req.Header.Set("Accept", "application/octet-stream")
	logs.Info("Downloading release artifact: " + downloadUrl)

	response, err := client.Do(req)
	if err != nil {
		logs.Error("Received error while downloading: " + err.Error())
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		logs.Error("Error: received non-200 response code: " + fmt.Sprint(response.StatusCode))
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		logs.Error("Error copying response body: " + err.Error())
		return err
	}
	return nil
}

// getArchSuffix returns the architecture suffix based on the current runtime architecture.
func getArchSuffix(logs *logger.Logger) string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		logs.Info("Warning: Unrecognized architecture" + arch + "Defaulting to amd64 binary")
		return "amd64"
	}
}

// downloadLatestBinary downloads the latest binary and its signature for the specified version and architecture.
func downloadLatestBinary(ctx context.Context, logs *logger.Logger, client *http.Client, newVersion string, newBinaryPath string, newBinarySigPath string, jfrogPluginDownloadUrl string, downloadSuffix string) error {
	// check if new version has v prefix, if yes, remove it
	if strings.HasPrefix(newVersion, "v") {
		logs.Info("Release tag '%s' is missing 'v' prefix. Prepending 'v'." + newVersion)
		newVersion = strings.TrimPrefix(newVersion, "v")
	}
	downloadUrl := jfrogPluginDownloadUrl + downloadSuffix + newVersion + "/jfrog-credential-provider-linux-" + getArchSuffix(logs)
	logs.Info("Downloading new binary from: " + downloadUrl)
	downloadSignUrl := downloadUrl + ".asc"
	err := downloadReleaseArtifacts(ctx, logs, client, newBinaryPath, downloadUrl)
	if err != nil {
		logs.Error("Failed to download new binary: " + err.Error())
		return err
	}

	err = downloadReleaseArtifacts(ctx, logs, client, newBinarySigPath, downloadSignUrl)
	if err != nil {
		logs.Error("Failed to download new binary signature: " + err.Error())
		return err
	}
	logs.Info("Attempting to download from: " + downloadUrl)
	return nil
}
