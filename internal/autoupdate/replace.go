// Package autoupdate provides functionality for automatically updating the JFrog credential provider binary.
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
	"jfrog-credential-provider/internal/logger"
	"os"
)

// replaceBinary replaces the current running binary with a new binary and makes it executable.
func replaceBinary(ctx context.Context, logs *logger.Logger, currentBinaryPath string, newBinaryPath string) error {
	err := os.Rename(newBinaryPath, currentBinaryPath)
	if err != nil {
		logs.Error("Error replacing binary: " + err.Error())
		return err
	}
	logs.Info("Replaced current binary with the new version at " + currentBinaryPath)
	// Make the binary executable
	err = os.Chmod(currentBinaryPath, 0755)
	if err != nil {
		logs.Error("Error making binary executable: " + err.Error())
		return err
	}
	logs.Info("Binary is now executable")
	return nil
}
