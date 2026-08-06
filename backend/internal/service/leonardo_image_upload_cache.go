package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	leonardoImageUploadCacheKeyPrefix = "leonardo:image-upload:"
	leonardoImageUploadCacheTTL       = time.Hour
)

type LeonardoImageUploadCache struct {
	redis *redis.Client
}

func NewLeonardoImageUploadCache(redisClient *redis.Client) *LeonardoImageUploadCache {
	return &LeonardoImageUploadCache{redis: redisClient}
}

func LeonardoImageSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *LeonardoImageUploadCache) Get(ctx context.Context, accountID int64, imageSHA256 string) (string, bool, error) {
	key, ok := c.key(accountID, imageSHA256)
	if !ok {
		return "", false, nil
	}
	uploadedID, err := c.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get Leonardo image upload cache: %w", err)
	}
	uploadedID = strings.TrimSpace(uploadedID)
	return uploadedID, uploadedID != "", nil
}

func (c *LeonardoImageUploadCache) Set(ctx context.Context, accountID int64, imageSHA256, uploadedID string) error {
	key, ok := c.key(accountID, imageSHA256)
	uploadedID = strings.TrimSpace(uploadedID)
	if !ok || uploadedID == "" {
		return nil
	}
	if err := c.redis.Set(ctx, key, uploadedID, leonardoImageUploadCacheTTL).Err(); err != nil {
		return fmt.Errorf("set Leonardo image upload cache: %w", err)
	}
	return nil
}

func (c *LeonardoImageUploadCache) key(accountID int64, imageSHA256 string) (string, bool) {
	imageSHA256 = strings.ToLower(strings.TrimSpace(imageSHA256))
	if c == nil || c.redis == nil || accountID <= 0 || len(imageSHA256) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(imageSHA256); err != nil {
		return "", false
	}
	return fmt.Sprintf("%s%d:%s", leonardoImageUploadCacheKeyPrefix, accountID, imageSHA256), true
}
