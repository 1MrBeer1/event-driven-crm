package leadservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mr-beer/event-driven-crm/internal/models"
)

type Cache interface {
	GetLead(ctx context.Context, id string) (models.Lead, bool, error)
	SetLead(ctx context.Context, lead models.Lead) error
}

type RedisCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisCache(client *goredis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{client: client, ttl: ttl}
}

func (c *RedisCache) GetLead(ctx context.Context, id string) (models.Lead, bool, error) {
	raw, err := c.client.Get(ctx, leadCacheKey(id)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return models.Lead{}, false, nil
	}
	if err != nil {
		return models.Lead{}, false, err
	}

	var lead models.Lead
	if err := json.Unmarshal(raw, &lead); err != nil {
		return models.Lead{}, false, err
	}
	return lead, true, nil
}

func (c *RedisCache) SetLead(ctx context.Context, lead models.Lead) error {
	raw, err := json.Marshal(lead)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, leadCacheKey(lead.ID), raw, c.ttl).Err()
}

func leadCacheKey(id string) string {
	return fmt.Sprintf("crm:lead:%s", id)
}
