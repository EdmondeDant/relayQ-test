package service

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAuthServiceAllowSignupBalanceGiftForIPLimitsFirstTwoPerDay(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := &AuthService{redisClient: rdb}
	ctx := context.Background()

	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10") {
		t.Fatal("first signup from IP should receive gift")
	}
	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10") {
		t.Fatal("second signup from IP should receive gift")
	}
	if svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10") {
		t.Fatal("third signup from same IP should not receive gift")
	}
}

func TestAuthServiceAllowSignupBalanceGiftForIPSeparatesIPs(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := &AuthService{redisClient: rdb}
	ctx := context.Background()

	_ = svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10")
	_ = svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10")

	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.11") {
		t.Fatal("different IP should have its own daily gift quota")
	}
}

func TestAuthServiceAllowSignupBalanceGiftForIPFailsOpenWithoutRedisOrIP(t *testing.T) {
	svc := &AuthService{}
	ctx := context.Background()

	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10") {
		t.Fatal("missing redis should fail open")
	}
	if !(&AuthService{redisClient: redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})}).allowSignupBalanceGiftForIP(ctx, "") {
		t.Fatal("missing IP should fail open")
	}
}
