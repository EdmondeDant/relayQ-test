package service

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type MediaCatalogProduct struct {
	ID          int64               `json:"id"`
	PublicModel string              `json:"public_model"`
	Modality    string              `json:"modality"`
	Enabled     bool                `json:"enabled"`
	Description *string             `json:"description"`
	GroupIDs    []int64             `json:"group_ids"`
	Prices      []MediaCatalogPrice `json:"prices"`
	Offers      []MediaCatalogOffer `json:"offers"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type MediaCatalogPrice struct {
	ID           int64           `json:"id"`
	Operation    string          `json:"operation"`
	SpecKey      string          `json:"spec_key"`
	UnitPriceUSD decimal.Decimal `json:"unit_price_usd"`
	Currency     string          `json:"currency"`
	Version      string          `json:"version"`
	Enabled      bool            `json:"enabled"`
}

type MediaCatalogOffer struct {
	ID            int64          `json:"id"`
	Provider      string         `json:"provider"`
	SourceGroupID int64          `json:"source_group_id"`
	UpstreamModel string         `json:"upstream_model"`
	Enabled       bool           `json:"enabled"`
	Priority      int            `json:"priority"`
	Operations    []string       `json:"operations"`
	Capabilities  map[string]any `json:"capabilities"`
	CostRules     map[string]any `json:"cost_rules"`
	CostSource    string         `json:"cost_source"`
	CostVersion   string         `json:"cost_version"`
	VerifiedAt    time.Time      `json:"verified_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
}

type MediaCatalogGroup struct {
	ID       int64
	Platform string
	Status   string
}

type MediaProductRepository interface {
	List(ctx context.Context, offset, limit int, search, modality string) ([]MediaCatalogProduct, int64, error)
	GetByID(ctx context.Context, id int64) (*MediaCatalogProduct, error)
	GetRuntime(ctx context.Context, groupID int64, publicModel, modality string, now time.Time) (*MediaCatalogProduct, error)
	ListRuntimeModels(ctx context.Context, groupID int64, now time.Time) ([]string, error)
	GetGroups(ctx context.Context, ids []int64) (map[int64]MediaCatalogGroup, error)
	Create(ctx context.Context, product *MediaCatalogProduct) error
	Update(ctx context.Context, product *MediaCatalogProduct) error
	Disable(ctx context.Context, id int64) error
}
