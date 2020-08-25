package durable

import (
	"fmt"
	"time"

	"github.com/MixinNetwork/ocean.one/example/uuid"
	"github.com/go-redis/redis"
)

type Limiter struct {
	pool *redis.Client
}

func NewLimiter(addr string, db int) (*Limiter, error) {
	options := &redis.Options{
		Addr:         addr,
		DB:           db,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  60 * time.Second,
		PoolSize:     1024,
	}
	redisPool := redis.NewClient(options)
	if err := redisPool.Ping().Err(); err != nil {
		return nil, err
	}

	return &Limiter{pool: redisPool}, nil
}

func (limiter *Limiter) Available(key string, window time.Duration, max int, shouldIncr bool) (int, error) {
	now := time.Now()
	key = fmt.Sprintf("%s:%d", key, int64(window.Seconds()))
	var zcount *redis.IntCmd
	_, err := limiter.pool.Pipelined(func(pipe redis.Pipeliner) error {
		pipe.ZRemRangeByScore(key, "-inf", fmt.Sprint(now.Add(-window).UnixNano()/1000000))
		if shouldIncr {
			pipe.ZAdd(key, redis.Z{Score: float64(now.UnixNano() / 1000000), Member: uuid.NewV4().String()})
		}
		pipe.Expire(key, time.Second*time.Duration(int64(window.Seconds())+60))
		zcount = pipe.ZCount(key, "-inf", "+inf")
		return nil
	})
	if err != nil {
		return 0, err
	}
	count, err := zcount.Result()
	return max - int(count), err
}

func (limiter *Limiter) Clear(key string, window time.Duration) error {
	key = fmt.Sprintf("%s:%d", key, int64(window.Seconds()))
	zcount := limiter.pool.ZRemRangeByScore(key, "-inf", "+inf")
	_, err := zcount.Result()
	return err
}
