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

package main

import (
	"context"
	"flag"
	"jfrog-credential-provider/internal/provider"
	"os"
	"time"
)

var Version string

func main() {
	if Version == "" {
		Version = "unknown"
	}

	// Create a subcommand for add-provider-config
	addProviderConfigCmd := flag.NewFlagSet("add-provider-config", flag.ExitOnError)
	dryRun := addProviderConfigCmd.Bool("dry-run", false, "Perform a dry run without making changes")
	generateConfig := addProviderConfigCmd.Bool("generateConfig", false, "Generate jfrog provider config from environment variables")
	isYaml := addProviderConfigCmd.Bool("yaml", false, "Generate config in YAML format")
	providerHome := addProviderConfigCmd.String("provider-home", "", "Provider home directory")
	providerConfig := addProviderConfigCmd.String("provider-config", "", "Provider config file name")
	switch {
	case len(os.Args) > 1 && os.Args[1] == "add-provider-config":
		// Parse flags for the subcommand
		addProviderConfigCmd.Parse(os.Args[2:])

		resolvedProviderHome, resolvedProviderConfig := provider.ProcessProviderConfigEnvs(*providerHome, *providerConfig)

		if *generateConfig {
			provider.CreateProviderConfigFromEnv(*isYaml, resolvedProviderHome, resolvedProviderConfig)
		} else {
			provider.MergeConfig(*dryRun, *isYaml, resolvedProviderHome, resolvedProviderConfig)
		}
		return
	default:
		// Default behavior
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		provider.StartProvider(ctx, Version)
	}
}
