package service

import (
	"context"
	"testing"

	"time"
)

type signupBonusIPLimiterFake struct{ counts map[string]int }

func (f *signupBonusIPLimiterFake) Allow(_ context.Context, ip string, _ time.Time) (bool, error) {
	if f.counts == nil {
		f.counts = map[string]int{}
	}
	f.counts[ip]++
	return f.counts[ip] <= 2, nil
}

func TestAuthServiceAllowSignupBalanceGiftForIPLimitsFirstTwoPerDay(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &signupBonusIPLimiterFake{})
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
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &signupBonusIPLimiterFake{})
	ctx := context.Background()

	_ = svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10")
	_ = svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10")

	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.11") {
		t.Fatal("different IP should have its own daily gift quota")
	}
}

func TestAuthServiceAllowSignupBalanceGiftForIPFailsOpenWithoutRedisOrIP(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := context.Background()

	if !svc.allowSignupBalanceGiftForIP(ctx, "203.0.113.10") {
		t.Fatal("missing redis should fail open")
	}
	if !NewAuthService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).allowSignupBalanceGiftForIP(ctx, "") {
		t.Fatal("missing IP should fail open")
	}
}
