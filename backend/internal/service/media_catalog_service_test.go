package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type mediaCatalogRepoStub struct {
	groups map[int64]MediaCatalogGroup
	saved  *MediaCatalogProduct
}

func (r *mediaCatalogRepoStub) List(context.Context, int, int, string, string) ([]MediaCatalogProduct, int64, error) {
	return nil, 0, nil
}

func (r *mediaCatalogRepoStub) GetByID(_ context.Context, id int64) (*MediaCatalogProduct, error) {
	if r.saved == nil || r.saved.ID != id {
		return nil, ErrMediaCatalogProductNotFound
	}
	return r.saved, nil
}

func (r *mediaCatalogRepoStub) GetRuntime(_ context.Context, _ int64, _, _ string, _ time.Time) (*MediaCatalogProduct, error) {
	if r.saved == nil {
		return nil, ErrMediaCatalogProductNotFound
	}
	return r.saved, nil
}

func (r *mediaCatalogRepoStub) ListRuntimeModels(context.Context, int64, time.Time) ([]string, error) {
	if r.saved == nil {
		return nil, nil
	}
	return []string{r.saved.PublicModel}, nil
}

func (r *mediaCatalogRepoStub) GetGroups(context.Context, []int64) (map[int64]MediaCatalogGroup, error) {
	return r.groups, nil
}

func (r *mediaCatalogRepoStub) Create(_ context.Context, product *MediaCatalogProduct) error {
	product.ID = 9
	r.saved = product
	return nil
}

func (r *mediaCatalogRepoStub) Update(_ context.Context, product *MediaCatalogProduct) error {
	r.saved = product
	return nil
}

func (r *mediaCatalogRepoStub) Disable(context.Context, int64) error { return nil }

func TestMediaCatalogServiceCreate(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	repo := &mediaCatalogRepoStub{groups: map[int64]MediaCatalogGroup{
		1: {ID: 1, Platform: PlatformOpenAI, Status: StatusActive},
		2: {ID: 2, Platform: PlatformLeonardo, Status: StatusActive},
	}}
	service := NewMediaCatalogService(repo)
	service.now = func() time.Time { return now }
	product := validMediaCatalogProduct(now)
	created, err := service.Create(context.Background(), product)
	require.NoError(t, err)
	require.Equal(t, int64(9), created.ID)
	require.Equal(t, "USD", created.Prices[0].Currency)
}

func TestMediaCatalogServiceRejectsProviderMismatch(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	repo := &mediaCatalogRepoStub{groups: map[int64]MediaCatalogGroup{
		1: {ID: 1, Platform: PlatformOpenAI, Status: StatusActive},
		2: {ID: 2, Platform: PlatformOpenAI, Status: StatusActive},
	}}
	service := NewMediaCatalogService(repo)
	service.now = func() time.Time { return now }
	_, err := service.Create(context.Background(), validMediaCatalogProduct(now))
	require.Equal(t, "INVALID_MEDIA_SOURCE_GROUP", infraerrors.Reason(err))
}

func TestMediaCatalogServiceRejectsExpiredCost(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	repo := &mediaCatalogRepoStub{groups: map[int64]MediaCatalogGroup{
		1: {ID: 1, Platform: PlatformOpenAI, Status: StatusActive},
		2: {ID: 2, Platform: PlatformLeonardo, Status: StatusActive},
	}}
	service := NewMediaCatalogService(repo)
	service.now = func() time.Time { return now }
	product := validMediaCatalogProduct(now)
	product.Offers[0].ExpiresAt = now.Add(-time.Minute)
	_, err := service.Create(context.Background(), product)
	require.Equal(t, "INVALID_MEDIA_OFFER_COST_EXPIRY", infraerrors.Reason(err))
}

func TestMediaCatalogServiceRuntimeQuery(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	product := validMediaCatalogProduct(now)
	product.ID = 9
	repo := &mediaCatalogRepoStub{saved: product}
	service := NewMediaCatalogService(repo)
	service.now = func() time.Time { return now }
	got, err := service.GetRuntime(context.Background(), 1, " relayq-image-v1 ", "IMAGE")
	require.NoError(t, err)
	require.Equal(t, int64(9), got.ID)
	_, err = service.GetRuntime(context.Background(), 0, "", "image")
	require.Equal(t, "INVALID_MEDIA_RUNTIME_QUERY", infraerrors.Reason(err))
}

func validMediaCatalogProduct(now time.Time) *MediaCatalogProduct {
	return &MediaCatalogProduct{
		PublicModel: "relayq-image-v1",
		Modality:    "image",
		Enabled:     true,
		GroupIDs:    []int64{1},
		Prices: []MediaCatalogPrice{{
			Operation: "generations", SpecKey: "size=1024x1024;n=1", UnitPriceUSD: decimal.RequireFromString("0.08"), Version: "2026-08-14", Enabled: true,
		}},
		Offers: []MediaCatalogOffer{{
			Provider: "leonardo", SourceGroupID: 2, UpstreamModel: "lucid-origin", Enabled: true,
			Operations: []string{"generations"}, Capabilities: map[string]any{"sizes": []any{"1024x1024"}},
			CostRules: map[string]any{"size=1024x1024;n=1": "0.01"}, CostSource: "vendor price sheet", CostVersion: "2026-08-14",
			VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(24 * time.Hour),
		}},
	}
}
