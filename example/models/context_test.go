package models

import (
	"context"
	"log"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
)

const (
	testEnvironment = "test"
	testDatabase    = "projects/mixin-183904/instances/development/databases/ocean-one-test"
)

func TestClear(t *testing.T) {
	ctx := setupTestContext()
	teardownTestContext(ctx)
}

func teardownTestContext(ctx context.Context) {
	db := session.Database(ctx)
	tables := []string{
		"properties",
		"pool_keys",
		"verifications",
		"users",
		"keys",
		"sessions",
		"candles",
		"markets",
	}
	for _, table := range tables {
		err := db.Apply(ctx, []*spanner.Mutation{spanner.Delete(table, spanner.AllKeys())}, "all", "DELETE", "DELETE FROM all")
		if err != nil {
			log.Println(table, err)
		}
	}
	db.Close()
}

func setupTestContext() context.Context {
	if config.Environment != testEnvironment || config.GoogleCloudSpanner != testDatabase {
		log.Panicln(config.Environment, config.GoogleCloudSpanner)
	}

	spanner, err := durable.OpenSpannerClient(context.Background(), config.GoogleCloudSpanner)
	if err != nil {
		log.Panicln(err)
	}
	limiter, err := durable.NewLimiter(config.RedisRateLimiterAddress, config.RedisRateLimiterDatabase)
	if err != nil {
		log.Panicln(err)
	}

	db := durable.WrapDatabase(spanner, nil)
	ctx := session.WithDatabase(context.Background(), db)
	return session.WithLimiter(ctx, limiter)
}
