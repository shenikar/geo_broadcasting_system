package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient создает и возвращает новый клиент Redis
func NewRedisClient(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 10,
	})

	// Проверяем соединение с Redis
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rdb, nil
}
