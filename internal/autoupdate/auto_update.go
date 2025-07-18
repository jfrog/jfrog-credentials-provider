
// Package autoupdate provides functionality for automatically updating the JFrog credential provider binary.
package autoupdate

import (
	"jfrog-credential-provider/internal/logger"
	"context"
	"runtime"
	"syscall"
	"net/http"
	"os"
	"jfrog-credential-provider/internal/utils"
)

// AutoUpdate checks for a new version, downloads, verifies, validates, and replaces the current binary if an update is available.
func AutoUpdate(logs *logger.Logger, client *http.Client, ctx context.Context, Version string) {
	// check for the environment variable to disable auto-update
	var autoUpdateDisabled bool
	autoUpdateDisabled = utils.GetEnvsBool(logs, "disable_provider_autoupdate", false)
	if autoUpdateDisabled {
		logs.Info("Auto-update functionality is disabled. Skipping auto-update process.")
		runtime.Goexit()
	}

	currentBinaryPath := utils.GetCurrentBinaryPath(logs)
	// check if lock exists
	lockFilePath := currentBinaryPath + ".lock"

	// Open the lock file
	lockFile, err := os.OpenFile(lockFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logs.Error("Failed to open lock file: "+err.Error())
		runtime.Goexit()
	}
	defer lockFile.Close()

	// Acquire lock
	err = utils.GetLock(logs, lockFile, syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		logs.Error("Failed to acquire lock for auto-update: "+err.Error())
		runtime.Goexit()
	}
	defer utils.ReleaseLock(logs, lockFile)

	jfrogPluginReleasesUrl := utils.GetEnvs(logs, "JFROG_CREDENTIAL_PROVIDER_RELEASES_URL", "https://releases.jfrog.io/artifactory/api/storage/run/jfrog-credentials-provider")
	// githubApiToken := os.Getenv("GITHUB_API_TOKEN")


	logs.Info("jfrogPluginReleasesUrl: " + jfrogPluginReleasesUrl)

	// Step 1: Fetch the latest minor version tag from the JFrog plugin releases URL
	latestBinaryVersionAvailable, err := fetchLatestVersionTag(ctx, client, Version, jfrogPluginReleasesUrl, logs)
	if err != nil {
		logs.Error("Failed to fetch latest version tag: " + err.Error())
		runtime.Goexit()
	}
	if latestBinaryVersionAvailable == "" {
		logs.Info("No new version available. Current version is up-to-date: " + Version)
		runtime.Goexit()
	}
	logs.Info("Latest binary version available: " + latestBinaryVersionAvailable)
	newBinaryPath := currentBinaryPath + latestBinaryVersionAvailable
	newBinarySigPath := newBinaryPath + ".asc"

	// Different from releases URL, this is the download URL for the JFrog credential provider binary.
	jfrogPluginDownloadUrl := utils.GetEnvs(logs, "JFROG_CREDENTIAL_PROVIDER_DOWNLOAD_URL", "https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider")
	logs.Info("jfrogPluginDownloadUrl: " + jfrogPluginDownloadUrl)

	// allows us to change the download url incase of release repositories are configured differently
	// does not need to be set in usual cases 
	downloadSuffix := utils.GetEnvs(logs, "JFROG_CREDENTIAL_PROVIDER_DOWNLOAD_SUFFIX", "/")

	// Step 2: Download the latest binary and its signature
	err = downloadLatestBinary(ctx, logs, client, latestBinaryVersionAvailable, newBinaryPath, newBinarySigPath, jfrogPluginDownloadUrl, downloadSuffix)
	if err != nil {
		logs.Error("Failed to download latest binary: " + err.Error())
		runtime.Goexit()
	}

	// Step 3: Verify the downloaded binary with its signature
	err = verifyBinaryWithSignature(logs, newBinaryPath, newBinarySigPath)
	if err != nil {
		logs.Error("Failed to verify binary with signature: " + err.Error())
		runtime.Goexit()
	}

	// Step 4: Validate the new kubelet binary by using the auth to ping targt artifactory
	err = validateKubeletBinary(ctx, client, logs, newBinaryPath)
	if err != nil {
		logs.Error("Failed to validate kubelet binary: " + err.Error())
		runtime.Goexit()
	}

	// Step 5: Replace the current binary with the new binary
	err = replaceBinary(ctx, logs, currentBinaryPath, newBinaryPath)
	if err != nil {
		logs.Error("Failed to replace binary: " + err.Error())
		runtime.Goexit()
	}
	logs.Info("Auto-update to version " + latestBinaryVersionAvailable + " completed successfully. New binary is now in use for the next session.")
}