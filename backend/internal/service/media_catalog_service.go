package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrMediaCatalogProductNotFound = infraerrors.NotFound("MEDIA_PRODUCT_NOT_FOUND", "media product not found")

type MediaCatalogService struct {
	repo MediaProductRepository
	now  func() time.Time
}

type MediaRuntimeModel struct {
	Model    string
	Modality string
}

func NewMediaCatalogService(repo MediaProductRepository) *MediaCatalogService {
	return &MediaCatalogService{repo: repo, now: time.Now}
}

func (s *MediaCatalogService) List(ctx context.Context, page, pageSize int, search, modality string) ([]MediaCatalogProduct, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, (page-1)*pageSize, pageSize, strings.TrimSpace(search), strings.TrimSpace(modality))
}

func (s *MediaCatalogService) GetByID(ctx context.Context, id int64) (*MediaCatalogProduct, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MEDIA_PRODUCT_ID", "invalid media product ID")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *MediaCatalogService) GetRuntime(ctx context.Context, groupID int64, publicModel, modality string) (*MediaCatalogProduct, error) {
	publicModel = strings.TrimSpace(publicModel)
	modality = strings.ToLower(strings.TrimSpace(modality))
	if groupID <= 0 || publicModel == "" || modality != "image" && modality != "video" {
		return nil, infraerrors.BadRequest("INVALID_MEDIA_RUNTIME_QUERY", "group, public model, and modality are required")
	}
	return s.repo.GetRuntime(ctx, groupID, publicModel, modality, s.now().UTC())
}

func (s *MediaCatalogService) ListRuntimeModels(ctx context.Context, groupID int64) ([]string, error) {
	if s == nil || s.repo == nil || groupID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MEDIA_RUNTIME_QUERY", "group is required")
	}
	return s.repo.ListRuntimeModels(ctx, groupID, s.now().UTC())
}

func (s *MediaCatalogService) ListRuntimeModelModalities(ctx context.Context, groupID int64) ([]MediaRuntimeModel, error) {
	if s == nil || s.repo == nil || groupID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MEDIA_RUNTIME_QUERY", "group is required")
	}
	repo, ok := s.repo.(interface {
		ListRuntimeModelModalities(context.Context, int64, time.Time) ([]MediaRuntimeModel, error)
	})
	if !ok {
		return nil, infraerrors.InternalServer("MEDIA_RUNTIME_MODALITY_UNAVAILABLE", "media runtime modality lookup is unavailable")
	}
	return repo.ListRuntimeModelModalities(ctx, groupID, s.now().UTC())
}

func (s *MediaCatalogService) Create(ctx context.Context, product *MediaCatalogProduct) (*MediaCatalogProduct, error) {
	if err := s.validate(ctx, product); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, product.ID)
}

func (s *MediaCatalogService) Update(ctx context.Context, id int64, product *MediaCatalogProduct) (*MediaCatalogProduct, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MEDIA_PRODUCT_ID", "invalid media product ID")
	}
	product.ID = id
	if err := s.validate(ctx, product); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *MediaCatalogService) Disable(ctx context.Context, id int64) error {
	if id <= 0 {
		return infraerrors.BadRequest("INVALID_MEDIA_PRODUCT_ID", "invalid media product ID")
	}
	return s.repo.Disable(ctx, id)
}

func (s *MediaCatalogService) validate(ctx context.Context, product *MediaCatalogProduct) error {
	if product == nil {
		return infraerrors.BadRequest("INVALID_MEDIA_PRODUCT", "media product is required")
	}
	product.PublicModel = strings.TrimSpace(product.PublicModel)
	product.Modality = strings.ToLower(strings.TrimSpace(product.Modality))
	if product.PublicModel == "" || len(product.PublicModel) > 200 {
		return infraerrors.BadRequest("INVALID_PUBLIC_MODEL", "public_model is required and must not exceed 200 characters")
	}
	if product.Modality != "image" && product.Modality != "video" {
		return infraerrors.BadRequest("INVALID_MEDIA_MODALITY", "modality must be image or video")
	}
	if len(product.GroupIDs) == 0 {
		return infraerrors.BadRequest("MEDIA_PRODUCT_GROUP_REQUIRED", "at least one entry group is required")
	}
	if len(product.Prices) == 0 {
		return infraerrors.BadRequest("MEDIA_PRODUCT_PRICE_REQUIRED", "at least one fixed price is required")
	}
	if len(product.Offers) == 0 {
		return infraerrors.BadRequest("MEDIA_PRODUCT_OFFER_REQUIRED", "at least one offer is required")
	}
	ids := append([]int64{}, product.GroupIDs...)
	for _, offer := range product.Offers {
		ids = append(ids, offer.SourceGroupID)
	}
	groups, err := s.repo.GetGroups(ctx, ids)
	if err != nil {
		return err
	}
	for _, id := range product.GroupIDs {
		group, ok := groups[id]
		if !ok || group.Platform != PlatformOpenAI || group.Status != StatusActive {
			return infraerrors.BadRequest("INVALID_MEDIA_ENTRY_GROUP", fmt.Sprintf("entry group %d must be an active openai group", id))
		}
	}
	seenPrices := map[string]struct{}{}
	for i := range product.Prices {
		price := &product.Prices[i]
		price.Operation = strings.ToLower(strings.TrimSpace(price.Operation))
		price.SpecKey = strings.TrimSpace(price.SpecKey)
		price.Version = strings.TrimSpace(price.Version)
		price.Currency = strings.ToUpper(strings.TrimSpace(price.Currency))
		if price.Currency == "" {
			price.Currency = "USD"
		}
		if !validMediaOperation(product.Modality, price.Operation) || price.SpecKey == "" || price.Version == "" || !price.UnitPriceUSD.IsPositive() || price.Currency != "USD" {
			return infraerrors.BadRequest("INVALID_MEDIA_PRODUCT_PRICE", fmt.Sprintf("price %d has invalid operation, specification, version, currency, or amount", i+1))
		}
		key := price.Operation + "\x00" + price.SpecKey + "\x00" + price.Version
		if _, ok := seenPrices[key]; ok {
			return infraerrors.BadRequest("DUPLICATE_MEDIA_PRODUCT_PRICE", "duplicate operation, spec_key, and version")
		}
		seenPrices[key] = struct{}{}
	}
	now := s.now().UTC()
	for i := range product.Offers {
		offer := &product.Offers[i]
		offer.Provider = strings.ToLower(strings.TrimSpace(offer.Provider))
		offer.UpstreamModel = strings.TrimSpace(offer.UpstreamModel)
		offer.CostSource = strings.TrimSpace(offer.CostSource)
		offer.CostVersion = strings.TrimSpace(offer.CostVersion)
		group, ok := groups[offer.SourceGroupID]
		if !ok || group.Status != StatusActive || group.Platform != offer.Provider {
			return infraerrors.BadRequest("INVALID_MEDIA_SOURCE_GROUP", fmt.Sprintf("offer %d source group must be active and match provider", i+1))
		}
		if offer.UpstreamModel == "" || offer.Priority < 0 || len(offer.Operations) == 0 || len(offer.Capabilities) == 0 || len(offer.CostRules) == 0 || offer.CostSource == "" || offer.CostVersion == "" {
			return infraerrors.BadRequest("INVALID_MEDIA_OFFER", fmt.Sprintf("offer %d is incomplete", i+1))
		}
		for _, operation := range offer.Operations {
			if !validMediaOperation(product.Modality, strings.ToLower(strings.TrimSpace(operation))) {
				return infraerrors.BadRequest("INVALID_MEDIA_OFFER_OPERATION", fmt.Sprintf("offer %d has an unsupported operation", i+1))
			}
		}
		if offer.VerifiedAt.IsZero() || !offer.ExpiresAt.After(offer.VerifiedAt) || !offer.ExpiresAt.After(now) {
			return infraerrors.BadRequest("INVALID_MEDIA_OFFER_COST_EXPIRY", fmt.Sprintf("offer %d cost verification must be current and unexpired", i+1))
		}
	}
	return nil
}

func validMediaOperation(modality, operation string) bool {
	switch modality {
	case "image":
		return operation == "generations" || operation == "edits"
	case "video":
		return operation == "generations" || operation == "edits" || operation == "extensions" || operation == "status" || operation == "content"
	default:
		return false
	}
}
