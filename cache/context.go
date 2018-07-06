package cache

import (
	"context"

	"github.com/go-redis/redis"
)

type contextValueKey int

const (
	keyRedis contextValueKey = 1
)

func SetupRedis(ctx context.Context, client *redis.Client) context.Context {
	return context.WithValue(ctx, keyRedis, client)
}

func Redis(ctx context.Context) *redis.Client {
	v, _ := ctx.Value(keyRedis).(*redis.Client)
	return v
}
