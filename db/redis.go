package db

import (
	"context"
	"voute/pkg/config"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() (*redis.Client, error) {
	cfg := config.LoadRedisConfig()
	var opts *redis.Options
	var err error

	if cfg.URL != "" {
		opts, err = redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, err
		}
	} else {
		opts = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       cfg.DB,
		}
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
