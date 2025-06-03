package autoupdate

import (
	"context"
	"net/http"
	"strings"
	"os"
	"encoding/json"
	"io"
	"fmt"
	"golang.org/x/mod/semver"
	"runtime"
	"jfrog-credential-provider/internal/logger"
)

type BinaryReleasesList struct {
	Tag      string `json:"tag_name"`
	AssetURL string `json:"assets_url"`
}

// removeVPrefix ensures the version tag starts with 'v'. If not, it prepends 'v'.
func removeVPrefix(logs *logger.Logger, versionTag string) string {
	if !strings.HasPrefix(versionTag, "v") {
		logs.Info("Release tag '%s' is missing 'v' prefix. Prepending 'v'." + versionTag)
		versionTag = "v" + versionTag
		return versionTag
	}
	return versionTag
}

// fetchLatestVersionTag fetches the latest available release tag from the JFrog plugin releases URL and compares it to the current version. Outputs a new version tag if a newer version is available.
func fetchLatestVersionTag(ctx context.Context, client *http.Client, currentVersion string, jfrogPluginReleasesUrl string, logs *logger.Logger) (string, error) {
	jfrogPluginReleasesUrl = jfrogPluginReleasesUrl + "/releases"
	request, err := http.NewRequestWithContext(ctx,"GET", jfrogPluginReleasesUrl, nil)
	// request.Header.Set("Authorization", "Bearer "+githubApiToken)
	request.Header.Set("Authorization", "Bearer cmVmdGtuOjAxOjE3ODA1NjkxOTY6cndoSkFDZ0IwWTdJZWhaMUw0d09zcE41T29J")
	logs.Info("Fetching latest version from: " + jfrogPluginReleasesUrl)
	if err != nil {
		logs.Error("Error creating request: "+err.Error())
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		logs.Error("Error sending request: "+err.Error())
		return "", err
	}

	logs.Debug("Response status code: " + fmt.Sprint(response.StatusCode))

	if response.StatusCode != http.StatusOK {
		logs.Error("Error: received non-200 response code: " + fmt.Sprint(response.StatusCode))
	}

	var releases []BinaryReleasesList
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logs.Error("Error reading response body: "+err.Error())
		return "", err
	}


	err = json.Unmarshal(body, &releases)
	if err != nil {
		logs.Error("Error decoding response: "+err.Error())
		return "", err
	}

	var latestVersionTag = removeVPrefix(logs, currentVersion)
	if !semver.IsValid(currentVersion) {
		logs.Error("Current Version" + currentVersion +  "isn't valid! Exiting")
		return "", fmt.Errorf("invalid current version: %s", currentVersion)
	}

	logs.Info("Current version: " + latestVersionTag)
	for _, release := range releases {
		releaseTag := removeVPrefix(logs, release.Tag)
		if !semver.IsValid(releaseTag) {
			logs.Debug("Version tag %s is not valid! Skipping" + releaseTag)
			continue
		}

		if semver.Major(latestVersionTag) == semver.Major(releaseTag) && semver.Compare(releaseTag, latestVersionTag) > 0 {
			latestVersionTag = releaseTag
			logs.Info("Newer version available: " + releaseTag)
		}
	}

	if latestVersionTag == removeVPrefix(logs, currentVersion) {
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
	if err != nil  {
		logs.Error("Failed to create new file: " + err.Error())
	}
	defer out.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		logs.Error("Received some error while forming get request: "+err.Error())
		return err
	}
	// req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer cmVmdGtuOjAxOjE3ODA1NjkxOTY6cndoSkFDZ0IwWTdJZWhaMUw0d09zcE41T29J")
	logs.Info("Downloading release artifact : " + downloadUrl)

	response, err := client.Do(req)
	if err != nil {
		logs.Error("Received error while downloading: "+err.Error())
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		logs.Error("Error: received non-200 response code: "+fmt.Sprint(response.StatusCode))
	}

	_, err = io.Copy(out, response.Body)
	if err != nil  {
		logs.Error("Error copying response body: "+err.Error())
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
func downloadLatestBinary(ctx context.Context, logs *logger.Logger, client *http.Client, newVersion string, newBinaryPath string, newBinarySigPath string, jfrogPluginReleasesUrl string, downloadSuffix string) error {
	// check if new version has v prefix, if not, add it
	if !strings.HasPrefix(newVersion, "v") {
		logs.Info("Release tag '%s' is missing 'v' prefix. Prepending 'v'." + newVersion)
		newVersion = "v" + newVersion
	}
	downloadUrl := jfrogPluginReleasesUrl + downloadSuffix + newVersion + "/jfrog-credential-provider-aws-linux-" + getArchSuffix(logs)
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
