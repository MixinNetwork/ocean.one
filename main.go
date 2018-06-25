package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/persistence"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClientWithConfig(ctx, config.GoogleCloudSpanner, spanner.ClientConfig{NumChannels: 4,
		SessionPoolConfig: spanner.SessionPoolConfig{
			HealthCheckInterval: 5 * time.Second,
		},
	})
	if err != nil {
		log.Panicln(err)
	}

	ctx = persistence.SetupSpanner(ctx, client)
	NewExchange().Run(ctx)
}
