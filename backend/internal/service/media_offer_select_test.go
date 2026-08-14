package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMediaOfferSelectChoosesLowestTrustedCostWithoutChangingCustomerPrice(t *testing.T) {
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	product := mediaTestProduct()
	offers := []MediaOffer{
		mediaTestOffer(2, "leonardo", 17, 0.03, now),
		mediaTestOffer(1, "openai_compat", 18, 0.02, now),
	}
	result := SelectMediaOffer(product, offers, mediaTestRequest(now))

	require.NoError(t, result.Err)
	require.Equal(t, int64(1), result.Selected.Offer.ID)
	require.Equal(t, 0.10, result.Selected.CustomerCharge)
	require.Equal(t, result.RankedEligible[0].CustomerCharge, result.RankedEligible[1].CustomerCharge)
	require.Equal(t, int64(2), offers[0].ID)
}

func TestMediaOfferSelectSkipsUnsupportedAndUntrustedOffers(t *testing.T) {
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	product := mediaTestProduct()
	unsupported := mediaTestOffer(1, "openai_compat", 17, 0.001, now)
	delete(unsupported.Capability.SupportedFields, "size")
	stale := mediaTestOffer(2, "leonardo", 18, 0.002, now.Add(-2*time.Hour))
	valid := mediaTestOffer(3, "leonardo", 19, 0.03, now)
	result := SelectMediaOffer(product, []MediaOffer{unsupported, stale, valid}, mediaTestRequest(now))

	require.NoError(t, result.Err)
	require.Equal(t, int64(3), result.Selected.Offer.ID)
	require.Equal(t, []MediaSkipReason{MediaSkipUnsupportedField, MediaSkipUntrustedCost}, []MediaSkipReason{result.Skipped[0].SkipReason, result.Skipped[1].SkipReason})
}

func TestMediaOfferSelectFailsClosedWhenAllCostsUntrusted(t *testing.T) {
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	offer := mediaTestOffer(1, "openai_compat", 17, 0.01, now)
	offer.Cost.TrustState = "unknown"
	result := SelectMediaOffer(mediaTestProduct(), []MediaOffer{offer}, mediaTestRequest(now))

	require.ErrorIs(t, result.Err, ErrNoTrustedMediaOffer)
	require.Nil(t, result.Selected)
	require.Equal(t, MediaSkipUntrustedCost, result.Skipped[0].SkipReason)
}

func TestMediaOfferSelectUsesStableTiebreakAndSourceACL(t *testing.T) {
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	first := mediaTestOffer(2, "openai_compat", 17, 0.02, now)
	first.Priority = 1
	second := mediaTestOffer(1, "leonardo", 18, 0.02, now)
	second.Priority = 1
	blocked := mediaTestOffer(3, "leonardo", 99, 0.001, now)
	result := SelectMediaOffer(mediaTestProduct(), []MediaOffer{first, blocked, second}, mediaTestRequest(now))

	require.NoError(t, result.Err)
	require.Equal(t, int64(1), result.Selected.Offer.ID)
	require.Equal(t, MediaSkipSourceGroupNotAllowed, result.Skipped[0].SkipReason)
}

func TestMediaOfferSelectIgnoresOtherProviderExtensionButRejectsPublicEnumMismatch(t *testing.T) {
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	offer := mediaTestOffer(1, "openai_compat", 17, 0.02, now)
	req := mediaTestRequest(now)
	req.Fields["flux_guidance"] = float64(4)
	req.ProviderExtensionFields = map[string]string{"flux_guidance": "leonardo"}
	result := SelectMediaOffer(mediaTestProduct(), []MediaOffer{offer}, req)
	require.NoError(t, result.Err)

	req.Fields["size"] = "4k"
	result = SelectMediaOffer(mediaTestProduct(), []MediaOffer{offer}, req)
	require.ErrorIs(t, result.Err, ErrNoTrustedMediaOffer)
	require.Equal(t, MediaSkipFieldEnumMismatch, result.Skipped[0].SkipReason)
}

func mediaTestProduct() MediaProduct {
	return MediaProduct{ID: 1, PublicModel: "relay-image", Modality: "image", Enabled: true, CustomerPrice: MediaCustomerPrice{Basis: "per_image", UnitPrice: 0.05, Currency: "USD", Version: "v1"}}
}

func mediaTestOffer(id int64, provider string, groupID int64, cost float64, verifiedAt time.Time) MediaOffer {
	return MediaOffer{
		ID: id, ProductID: 1, Provider: provider, SourceGroupID: groupID,
		UpstreamModel: "test-model", Enabled: true,
		Capability: MediaCapabilityProfile{Ops: []string{"generations"}, MaxN: 4, SupportedFields: map[string]MediaFieldCapability{"size": {Enum: []string{"1k", "2k"}}}},
		Cost:       TrustedCostPolicy{Basis: "per_image", UnitCost: cost, Currency: "USD", TrustState: "verified", VerifiedAt: verifiedAt, MaxAge: time.Hour, Source: "probe", Version: "v1"},
	}
}

func mediaTestRequest(now time.Time) MediaSelectRequest {
	return MediaSelectRequest{
		ProductPublicModel: "relay-image", Modality: MediaModalityImage, Op: "generations",
		Fields: map[string]any{"size": "1k"}, N: 2, SizeTier: "1k",
		AllowedSourceGroupIDs: map[int64]struct{}{17: {}, 18: {}, 19: {}}, Now: now,
	}
}
