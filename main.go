package main

import (
	"cloud.google.com/go/spanner"
	"context"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/persistence"
	"log"
	"time"
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

	NewExchange(persistence.CreateSpanner(client)).Run(ctx)
}
