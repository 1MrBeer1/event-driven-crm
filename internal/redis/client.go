package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mr-beer/event-driven-crm/internal/config"
)

const NotificationsChannel = "crm.notifications"

func Connect(ctx context.Context, cfg config.RedisConfig) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
