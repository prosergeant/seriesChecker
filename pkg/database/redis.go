package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/prosergeant/seriesChecker/pkg/config"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(ctx context.Context, cfg config.RedisConfig) (*RedisClient, error) {
	var client *redis.Client

	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis url: %w", err)
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.Client.Close()
}
