package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type leonardoImageUploadCache struct{ redis *redis.Client }

func NewLeonardoImageUploadCache(r *redis.Client) service.LeonardoImageUploadCache {
	return &leonardoImageUploadCache{redis: r}
}
func (c *leonardoImageUploadCache) key(a int64, h string) (string, bool) {
	h, ok := service.NormalizeLeonardoImageHash(h)
	if c == nil || c.redis == nil || a <= 0 || !ok {
		return "", false
	}
	return fmt.Sprintf("leonardo:image-upload:%d:%s", a, h), true
}
func (c *leonardoImageUploadCache) Get(x context.Context, a int64, h string) (string, bool, error) {
	k, ok := c.key(a, h)
	if !ok {
		return "", false, nil
	}
	v, e := c.redis.Get(x, k).Result()
	if errors.Is(e, redis.Nil) {
		return "", false, nil
	}
	if e != nil {
		return "", false, fmt.Errorf("get Leonardo image upload cache: %w", e)
	}
	v = strings.TrimSpace(v)
	return v, v != "", nil
}
func (c *leonardoImageUploadCache) Set(x context.Context, a int64, h, id string) error {
	k, ok := c.key(a, h)
	id = strings.TrimSpace(id)
	if !ok || id == "" {
		return nil
	}
	if e := c.redis.Set(x, k, id, time.Hour).Err(); e != nil {
		return fmt.Errorf("set Leonardo image upload cache: %w", e)
	}
	return nil
}
