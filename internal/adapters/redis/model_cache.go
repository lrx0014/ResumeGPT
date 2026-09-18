package redis

import (
	"context"
	"encoding/json"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

type ModelCache struct {
	client *redisclient.Client
}

func NewModelCache(rawURL string) (*ModelCache, error) {
	options, err := redisclient.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	return &ModelCache{client: redisclient.NewClient(options)}, nil
}

func (c *ModelCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *ModelCache) Get(ctx context.Context, key string) ([]string, bool, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if err == redisclient.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var models []string
	if err := json.Unmarshal(value, &models); err != nil {
		return nil, false, err
	}
	return models, true, nil
}

func (c *ModelCache) Set(ctx context.Context, key string, models []string, ttl time.Duration) error {
	value, err := json.Marshal(models)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *ModelCache) Close() error {
	return c.client.Close()
}
