package main

import (
	"context"
	"jfrog-credential-provider/internal/provider"
	"time"
)

var Version string

func main() {
	if Version == "" {
		Version = "unknown"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provider.StartProvider(ctx, Version)
}
