
// Package autoupdate provides functionality for automatically updating the JFrog credential provider binary.
package autoupdate

import (
	"os"
	"context"
	"jfrog-credential-provider/internal/logger"
)

// replaceBinary replaces the current running binary with a new binary and makes it executable.
func replaceBinary(ctx context.Context, logs *logger.Logger, currentBinaryPath string, newBinaryPath string) error {
	err := os.Rename(newBinaryPath, currentBinaryPath)
	if err != nil {
		logs.Error("Error replacing binary: "+err.Error())
		return err
	}
	logs.Info("Replaced current binary with the new version at " + currentBinaryPath)
	// Make the binary executable
	err = os.Chmod(currentBinaryPath, 0755)
	if err != nil {
		logs.Error("Error making binary executable: "+err.Error())
		return err
	}
	logs.Info("Binary is now executable")
	return nil
}