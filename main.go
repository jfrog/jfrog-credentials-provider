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
	switch {
	case len(os.Args) > 1 && os.Args[1] == "add-provider-config":
		// Parse flags for the subcommand
		addProviderConfigCmd.Parse(os.Args[2:])

		if *generateConfig {
			provider.CreateProviderConfigFromEnv(*isYaml)
		} else {
			provider.MergeConfig(*dryRun, *isYaml)
		}
		return
	default:
		// Default behavior
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		provider.StartProvider(ctx, Version)
	}
}
