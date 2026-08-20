package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
	hash := service.LeonardoImageSHA256([]byte("image"))
	require.NoError(t, cache.Set(ctx, 1, hash, "uploaded-1"))
	uploadedID, found, err := cache.Get(ctx, 1, hash)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "uploaded-1", uploadedID)
	_, found, err = cache.Get(ctx, 2, hash)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, time.Hour, server.TTL("leonardo:image-upload:1:"+hash))
	server.FastForward(time.Hour)
	_, found, err = cache.Get(ctx, 1, hash)
	require.NoError(t, err)
	require.False(t, found)
}
