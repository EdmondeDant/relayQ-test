package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

var (
	ErrCanvasRouteNotFound      = errors.New("canvas route not found")
	ErrCanvasRouteAmbiguous     = errors.New("canvas route ambiguous")
	ErrCanvasPricingUnavailable = errors.New("canvas pricing unavailable")
)

type CanvasRouteRequest struct {
	UserID     int64
	APIKeyID   int64
	Method     string
	Endpoint   string
	Model      string
	ResourceID string
	Modality   string
}

type CanvasRoute struct {
	Group    *Group
	Platform string
	Model    string
	Protocol string
}

type CanvasModel struct {
	ID        string   `json:"id"`
	Modality  string   `json:"modality"`
	Platform  string   `json:"platform"`
	Protocol  string   `json:"protocol"`
	Endpoints []string `json:"endpoints"`
}

type CanvasResourceRoute struct {
	APIKeyID       int64
	UserID         int64
	ResourceID     string
	GroupID        int64
	Platform       string
	Model          string
	EndpointFamily string
	ExpiresAt      time.Time
}

type CanvasResourceRouteRepository interface {
	GetActive(ctx context.Context, userID, apiKeyID int64, resourceID string) (*CanvasResourceRoute, error)
	Upsert(ctx context.Context, route *CanvasResourceRoute) error
}

type CanvasRoutingService struct {
	users     UserRepository
	groups    GroupRepository
	subs      UserSubscriptionRepository
	accounts  AccountRepository
	billing   *BillingService
	channels  *ChannelService
	jobs      GenerationJobRepository
	resources CanvasResourceRouteRepository
}

func NewCanvasRoutingService(users UserRepository, groups GroupRepository, subs UserSubscriptionRepository, accounts AccountRepository, billing *BillingService) *CanvasRoutingService {
	return &CanvasRoutingService{users: users, groups: groups, subs: subs, accounts: accounts, billing: billing}
}

func (s *CanvasRoutingService) SetGenerationJobRepository(jobs GenerationJobRepository) {
	s.jobs = jobs
}

func (s *CanvasRoutingService) SetResourceRouteRepository(resources CanvasResourceRouteRepository) {
	s.resources = resources
}

func (s *CanvasRoutingService) SetChannelService(channels *ChannelService) {
	s.channels = channels
}

func (s *CanvasRoutingService) Resolve(ctx context.Context, req CanvasRouteRequest) (*CanvasRoute, error) {
	groups, err := s.availableGroups(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if req.ResourceID != "" && s.jobs != nil {
		job, jobErr := s.jobs.GetByPublicID(ctx, req.ResourceID)
		if jobErr == nil && job.UserID == req.UserID && job.APIKeyID == req.APIKeyID && job.GroupID != nil {
			for i := range groups {
				if groups[i].ID == *job.GroupID && canvasPlatformSupportsEndpoint(groups[i].Platform, req.Endpoint) {
					group := groups[i]
					return &CanvasRoute{Group: &group, Platform: group.Platform, Model: job.Model, Protocol: canvasProtocol(group.Platform, req.Endpoint)}, nil
				}
			}
			return nil, ErrCanvasRouteNotFound
		}
		if jobErr == nil || !errors.Is(jobErr, ErrGenerationJobNotFound) {
			return nil, ErrCanvasRouteNotFound
		}
	}
	if req.ResourceID != "" && s.resources != nil {
		resource, resourceErr := s.resources.GetActive(ctx, req.UserID, req.APIKeyID, req.ResourceID)
		if resourceErr != nil {
			return nil, resourceErr
		}
		if resource != nil {
			for i := range groups {
				if groups[i].ID == resource.GroupID && canvasPlatformSupportsEndpoint(groups[i].Platform, req.Endpoint) {
					group := groups[i]
					return &CanvasRoute{Group: &group, Platform: group.Platform, Model: resource.Model, Protocol: canvasProtocol(group.Platform, req.Endpoint)}, nil
				}
			}
			return nil, ErrCanvasRouteNotFound
		}
	}
	if req.ResourceID != "" {
		return nil, ErrCanvasRouteNotFound
	}
	candidates := make([]Group, 0, len(groups))
	hadModel := false
	hadPricing := false
	for i := range groups {
		group := groups[i]
		if !canvasPlatformSupportsEndpoint(group.Platform, req.Endpoint) {
			continue
		}
		if req.Model != "" && s.channels != nil && s.channels.IsModelRestricted(ctx, group.ID, req.Model) {
			continue
		}
		if req.Model != "" && canvasModelForPlatform(req.Model, group.Platform).Modality == "image" && !group.AllowImageGeneration {
			continue
		}
		accounts, listErr := s.accounts.ListSchedulableByGroupID(ctx, group.ID)
		if listErr != nil {
			return nil, listErr
		}
		supported := req.Model == ""
		for j := range accounts {
			if req.Model == "" || accounts[j].IsModelSupported(req.Model) {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}
		hadModel = true
		if req.Model != "" && !s.pricingAvailable(ctx, group.ID, group.Platform, req.Model) {
			continue
		}
		hadPricing = true
		candidates = append(candidates, group)
	}
	if len(candidates) == 0 {
		if hadModel && !hadPricing {
			return nil, ErrCanvasPricingUnavailable
		}
		return nil, ErrCanvasRouteNotFound
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].SortOrder < candidates[j].SortOrder })
	if len(candidates) > 1 && candidates[0].SortOrder == candidates[1].SortOrder {
		return nil, ErrCanvasRouteAmbiguous
	}
	group := candidates[0]
	return &CanvasRoute{Group: &group, Platform: group.Platform, Model: req.Model, Protocol: canvasProtocol(group.Platform, req.Endpoint)}, nil
}

func (s *CanvasRoutingService) Catalog(ctx context.Context, userID int64) ([]CanvasModel, error) {
	groups, err := s.availableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	models := map[string]CanvasModel{}
	for i := range groups {
		accounts, listErr := s.accounts.ListSchedulableByGroupID(ctx, groups[i].ID)
		if listErr != nil {
			return nil, listErr
		}
		groupModels := make(map[string]struct{})
		for j := range accounts {
			for model := range accounts[j].GetModelMapping() {
				groupModels[model] = struct{}{}
			}
		}
		if groups[i].Platform == PlatformLeonardo && len(accounts) > 0 {
			for _, model := range leonardo.ListVerifiedModels() {
				for j := range accounts {
					if accounts[j].IsModelSupported(model.RequestModelSlug) {
						groupModels[model.RequestModelSlug] = struct{}{}
						break
					}
				}
			}
		} else if len(groupModels) == 0 && len(accounts) > 0 {
			for _, model := range canvasDefaultModels(groups[i].Platform) {
				groupModels[model] = struct{}{}
			}
		}
		for model := range groupModels {
			if strings.ContainsAny(model, "*?") || !s.pricingAvailable(ctx, groups[i].ID, groups[i].Platform, model) || s.channels != nil && s.channels.IsModelRestricted(ctx, groups[i].ID, model) {
				continue
			}
			item := canvasModelForPlatform(model, groups[i].Platform)
			if item.Modality == "image" && !groups[i].AllowImageGeneration {
				continue
			}
			models[item.Platform+"\x00"+item.ID] = item
		}
	}
	out := make([]CanvasModel, 0, len(models))
	for _, model := range models {
		out = append(out, model)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform == out[j].Platform {
			return out[i].ID < out[j].ID
		}
		return out[i].Platform < out[j].Platform
	})
	return out, nil
}

func canvasDefaultModels(platform string) []string {
	switch platform {
	case PlatformOpenAI:
		return openai.DefaultModelIDs()
	case PlatformXAI:
		return xai.DefaultModelIDs()
	case PlatformGemini, PlatformAntigravity:
		models := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			models = append(models, model.ID)
		}
		return models
	case PlatformLeonardo:
		models := leonardo.ListVerifiedModels()
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.RequestModelSlug)
		}
		return ids
	default:
		return claude.DefaultModelIDs()
	}
}

func (s *CanvasRoutingService) availableGroups(ctx context.Context, userID int64) ([]Group, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, err := s.groups.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	subscriptions, err := s.subs.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	subscribed := make(map[int64]bool, len(subscriptions))
	for i := range subscriptions {
		subscribed[subscriptions[i].GroupID] = true
	}
	available := groups[:0]
	for i := range groups {
		if (groups[i].IsSubscriptionType() && subscribed[groups[i].ID]) || (!groups[i].IsSubscriptionType() && user.CanBindGroup(groups[i].ID, groups[i].IsExclusive)) {
			available = append(available, groups[i])
		}
	}
	return available, nil
}

func (s *CanvasRoutingService) pricingAvailable(ctx context.Context, groupID int64, platform, model string) bool {
	if platform == PlatformLeonardo {
		verified, ok := leonardo.ResolveByRequestModelSlug(model)
		if !ok {
			return false
		}
		if verified.Modality == leonardo.ModelModalityVideo {
			_, price, err := EstimateLeonardoVideoCustomerPrice(ctx, LeonardoDefaultVideoPriceRequest(model))
			return err == nil && price.Sign() > 0
		}
		_, price, err := EstimateLeonardoCustomerPrice(ctx, LeonardoDefaultImagePriceRequest(model))
		return err == nil && price.Sign() > 0
	}
	if s.channels != nil && canvasChannelPricingAvailable(s.channels.GetChannelModelPricing(ctx, groupID, model)) {
		return true
	}
	pricing, err := s.billing.GetModelPricing(model)
	return err == nil && pricing != nil
}

func canvasChannelPricingAvailable(pricing *ChannelModelPricing) bool {
	if pricing == nil {
		return false
	}
	prices := []*float64{pricing.PerRequestPrice, pricing.ImageOutputPrice, pricing.InputPrice, pricing.OutputPrice}
	for _, price := range prices {
		if price != nil && *price > 0 {
			return true
		}
	}
	for i := range pricing.Intervals {
		interval := pricing.Intervals[i]
		if interval.PerRequestPrice != nil && *interval.PerRequestPrice > 0 || interval.InputPrice != nil && *interval.InputPrice > 0 || interval.OutputPrice != nil && *interval.OutputPrice > 0 {
			return true
		}
	}
	return false
}

func canvasPlatformSupportsEndpoint(platform, endpoint string) bool {
	path := strings.ToLower(endpoint)
	switch {
	case strings.HasPrefix(path, "/v1beta"):
		return platform == PlatformGemini || platform == PlatformAntigravity
	case strings.Contains(path, "/media/"):
		return platform == PlatformLeonardo
	case strings.Contains(path, "/embeddings"):
		return platform == PlatformOpenAI
	case strings.Contains(path, "/images"):
		return platform == PlatformOpenAI || platform == PlatformXAI || platform == PlatformLeonardo || platform == PlatformGemini || platform == PlatformAntigravity
	case strings.Contains(path, "/videos/edits") || strings.Contains(path, "/videos/extensions"):
		return platform == PlatformXAI
	case strings.Contains(path, "/videos"):
		return platform == PlatformOpenAI || platform == PlatformXAI || platform == PlatformLeonardo
	default:
		return platform != PlatformLeonardo
	}
}

func canvasProtocol(platform, endpoint string) string {
	if platform == PlatformLeonardo || strings.Contains(endpoint, "/media/") {
		return "relayq-media"
	}
	if strings.Contains(endpoint, "/videos") {
		return "openai-async"
	}
	if strings.HasPrefix(endpoint, "/v1beta") {
		return "gemini"
	}
	return "openai"
}

func canvasModelForPlatform(model, platform string) CanvasModel {
	if platform == PlatformLeonardo {
		verified, ok := leonardo.ResolveByRequestModelSlug(model)
		if ok && verified.Modality == leonardo.ModelModalityVideo {
			return CanvasModel{ID: model, Modality: "video", Platform: platform, Protocol: "relayq-media", Endpoints: []string{"/v1/media/generations", "/v1/media/videos/generations"}}
		}
		return CanvasModel{ID: model, Modality: "image", Platform: platform, Protocol: "relayq-media", Endpoints: []string{"/v1/media/generations"}}
	}
	if modality := openai.ModelModality(model); modality == "video" {
		return CanvasModel{ID: model, Modality: modality, Platform: platform, Protocol: "openai-async", Endpoints: []string{"/v1/videos/generations"}}
	} else if modality == "image" {
		return CanvasModel{ID: model, Modality: modality, Platform: platform, Protocol: "openai", Endpoints: []string{"/v1/images/generations", "/v1/images/edits"}}
	}
	lower := strings.ToLower(model)
	if strings.Contains(lower, "video") || strings.Contains(lower, "sora") {
		return CanvasModel{ID: model, Modality: "video", Platform: platform, Protocol: "openai-async", Endpoints: []string{"/v1/videos/generations"}}
	}
	if strings.Contains(lower, "image") || strings.Contains(lower, "imagen") {
		return CanvasModel{ID: model, Modality: "image", Platform: platform, Protocol: "openai", Endpoints: []string{"/v1/images/generations", "/v1/images/edits"}}
	}
	return CanvasModel{ID: model, Modality: "text", Platform: platform, Protocol: "openai", Endpoints: []string{"/v1/chat/completions", "/v1/responses"}}
}
