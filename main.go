package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/ocean.one/cache"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/persistence"
	"github.com/go-redis/redis"
)

func main() {
	ctx := context.Background()
	spannerClient, err := spanner.NewClientWithConfig(ctx, config.GoogleCloudSpanner, spanner.ClientConfig{NumChannels: 4,
		SessionPoolConfig: spanner.SessionPoolConfig{
			HealthCheckInterval: 5 * time.Second,
		},
	})
	if err != nil {
		log.Panicln(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddress,
		DB:           config.RedisDatabase,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  60 * time.Second,
		PoolSize:     1024,
	})
	err = redisClient.Ping().Err()
	if err != nil {
		log.Panicln(err)
	}

	ctx = persistence.SetupSpanner(ctx, spannerClient)
	ctx = cache.SetupRedis(ctx, redisClient)
	NewExchange().Run(ctx)
}
