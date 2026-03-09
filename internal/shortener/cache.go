package shortener

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{client: client, ttl: ttl}
}

func (c *Cache) Get(ctx context.Context, code string) (string, error) {
	url, err := c.client.Get(ctx, cacheKey(code)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return url, err
}

func (c *Cache) Set(ctx context.Context, code, url string) error {
	return c.client.Set(ctx, cacheKey(code), url, c.ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, code string) error {
	return c.client.Del(ctx, cacheKey(code)).Err()
}

func cacheKey(code string) string {
	return fmt.Sprintf("link:%s", code)
}
