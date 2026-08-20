package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSignupBonusIPLimiterCountsByIPAndExpiresAtShanghaiMidnight(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	limiter := NewSignupBonusIPLimiter(client)
	ctx := context.Background()
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 8, 21, 23, 30, 0, 0, location)

	allowed, err := limiter.Allow(ctx, "203.0.113.10", now)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = limiter.Allow(ctx, "203.0.113.10", now)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = limiter.Allow(ctx, "203.0.113.10", now)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = limiter.Allow(ctx, "203.0.113.11", now)
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, 30*time.Minute, server.TTL("signup_bonus_ip:20260821:203.0.113.10"))
}
