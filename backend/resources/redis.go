package resources

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() (*redis.Client, error) {
	return ConnectRedis(context.Background(), RedisCfg)
}

func ConnectRedis(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	addr := cfg.Addr
	if addr == "" {
		if cfg.Host != "" && cfg.Port != "" {
			addr = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		} else {
			addr = "localhost:6379"
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	RDB = client
	fmt.Println("redis connected successfully")
	return client, nil
}
