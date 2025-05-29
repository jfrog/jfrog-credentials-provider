package main

import (
	"context"
	"jfrog-credential-provider/internal/provider"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provider.StartProvider(ctx)
}
