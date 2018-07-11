package main

import (
	"context"
	"flag"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/ocean.one/cache"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/persistence"
	"github.com/go-redis/redis"
)

func main() {
	service := flag.String("service", "http", "run a service")
	flag.Parse()

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
		Addr:         config.RedisEngineCacheAddress,
		DB:           config.RedisEngineCacheDatabase,
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

	switch *service {
	case "engine":
		NewExchange().Run(ctx)
	case "http":
		StartHTTP(ctx)
	}
}
