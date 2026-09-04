package service

import (
	"testing"
)

func TestFallbackPricingClaudeFable51(t *testing.T) {
	t.Parallel()

	pricing := NewBillingService(nil, nil).getFallbackPricing("claude-fable-5-1")
	if pricing == nil {
		t.Fatal("expected Claude Fable 5.1 fallback pricing")
	}
	if pricing.InputPricePerToken != 10e-6 || pricing.OutputPricePerToken != 50e-6 || pricing.CacheReadPricePerToken != 0.25e-6 {
		t.Fatalf("unexpected Claude Fable 5.1 pricing: %+v", pricing)
	}
}
