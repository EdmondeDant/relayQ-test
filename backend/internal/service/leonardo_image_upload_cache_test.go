package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLeonardoImageUploadCacheScopesByAccountAndExpires(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewLeonardoImageUploadCache(client)
	ctx := context.Background()
	hash := LeonardoImageSHA256([]byte("image"))

	require.NoError(t, cache.Set(ctx, 1, hash, "uploaded-1"))
	uploadedID, found, err := cache.Get(ctx, 1, hash)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "uploaded-1", uploadedID)
	_, found, err = cache.Get(ctx, 2, hash)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, leonardoImageUploadCacheTTL, server.TTL(leonardoImageUploadCacheKeyPrefix+"1:"+hash))

	server.FastForward(leonardoImageUploadCacheTTL)
	_, found, err = cache.Get(ctx, 1, hash)
	require.NoError(t, err)
	require.False(t, found)
}

func TestLeonardoImageUploadCacheIgnoresInvalidKeys(t *testing.T) {
	cache := NewLeonardoImageUploadCache(nil)
	require.NoError(t, cache.Set(context.Background(), 0, "invalid", "uploaded"))
	_, found, err := cache.Get(context.Background(), 0, "invalid")
	require.NoError(t, err)
	require.False(t, found)
}
