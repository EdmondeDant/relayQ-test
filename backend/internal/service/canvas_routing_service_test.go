//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type canvasRoutingUserRepo struct{ UserRepository }

func (canvasRoutingUserRepo) GetByID(context.Context, int64) (*User, error) {
	return &User{ID: 1, Status: StatusActive}, nil
}

type canvasRoutingGroupRepo struct {
	GroupRepository
	groups []Group
}

func (r canvasRoutingGroupRepo) ListActive(context.Context) ([]Group, error) {
	return append([]Group(nil), r.groups...), nil
}

type canvasRoutingSubscriptionRepo struct{ UserSubscriptionRepository }

func (canvasRoutingSubscriptionRepo) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}

type canvasRoutingAccountRepo struct{ AccountRepository }

func (canvasRoutingAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}

type canvasRoutingAccountRepoWithModels struct {
	AccountRepository
	accounts []Account
}

func (r canvasRoutingAccountRepoWithModels) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return append([]Account(nil), r.accounts...), nil
}

type canvasRoutingAccountRepoByGroup struct {
	AccountRepository
	accounts map[int64][]Account
}

func (r canvasRoutingAccountRepoByGroup) ListSchedulableByGroupID(_ context.Context, groupID int64) ([]Account, error) {
	return append([]Account(nil), r.accounts[groupID]...), nil
}

type canvasRoutingMediaRepo struct {
	MediaProductRepository
	models []MediaRuntimeModel
}

func (r canvasRoutingMediaRepo) ListRuntimeModelModalities(context.Context, int64, time.Time) ([]MediaRuntimeModel, error) {
	return append([]MediaRuntimeModel(nil), r.models...), nil
}

type canvasRoutingJobRepo struct {
	GenerationJobRepository
	job *GenerationJob
}

func (r canvasRoutingJobRepo) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	return r.job, nil
}

func TestCanvasRoutingCatalogPreservesRuntimeModalities(t *testing.T) {
	group := Group{ID: 4, Platform: PlatformOpenAI, Status: StatusActive, AllowImageGeneration: true}
	media := NewMediaCatalogService(canvasRoutingMediaRepo{models: []MediaRuntimeModel{{Model: "shared-model", Modality: "image"}, {Model: "shared-model", Modality: "video"}}})
	service := NewCanvasRoutingService(canvasRoutingUserRepo{}, canvasRoutingGroupRepo{groups: []Group{group}}, canvasRoutingSubscriptionRepo{}, canvasRoutingAccountRepo{}, nil, media)

	models, err := service.Catalog(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, []CanvasModel{
		{ID: "shared-model", Modality: "image", Platform: PlatformOpenAI, Protocol: "openai", Endpoints: []string{"/v1/images/generations", "/v1/images/edits"}},
		{ID: "shared-model", Modality: "video", Platform: PlatformOpenAI, Protocol: "openai-async", Endpoints: []string{"/v1/videos"}},
	}, models)
}

func TestCanvasRoutingCatalogUsesChannelPricingForCustomModels(t *testing.T) {
	price := 0.01
	group := Group{ID: 4, Platform: PlatformOpenAI, Status: StatusActive, AllowImageGeneration: true}
	accounts := canvasRoutingAccountRepoWithModels{accounts: []Account{{Credentials: map[string]any{"model_mapping": map[string]any{
		"flux-2-klein-9b-kv": "flux-2-klein-9b-kv",
		"minimax-h3":         "minimax-h3",
	}}}}}
	channelRepo := &mockChannelRepository{
		listAllFn: func(context.Context) ([]Channel, error) {
			return []Channel{{ID: 8, Status: StatusActive, GroupIDs: []int64{group.ID}, ModelPricing: []ChannelModelPricing{{
				Platform: PlatformOpenAI, Models: []string{"flux-2-klein-9b-kv", "minimax-h3"}, PerRequestPrice: &price,
			}}}}, nil
		},
		getGroupPlatformsFn: func(context.Context, []int64) (map[int64]string, error) {
			return map[int64]string{group.ID: PlatformOpenAI}, nil
		},
	}
	routing := NewCanvasRoutingService(canvasRoutingUserRepo{}, canvasRoutingGroupRepo{groups: []Group{group}}, canvasRoutingSubscriptionRepo{}, accounts, nil, nil)
	routing.SetChannelService(NewChannelService(channelRepo, nil, nil, nil))

	models, err := routing.Catalog(context.Background(), 1)
	require.NoError(t, err)
	// flux-2-klein-9b-kv is an image model; minimax-h3 is an OpenAI-compatible
	// video model routed to the OpenAI platform group (account 73).
	require.Equal(t, []CanvasModel{
		{ID: "flux-2-klein-9b-kv", Modality: "image", Platform: PlatformOpenAI, Protocol: "openai", Endpoints: []string{"/v1/images/generations", "/v1/images/edits"}},
		{ID: "minimax-h3", Modality: "video", Platform: PlatformOpenAI, Protocol: "openai-async", Endpoints: []string{"/v1/videos/generations"}},
	}, models)
}

func TestCanvasRoutingUsesExplicitVideoModelPlatforms(t *testing.T) {
	leonardoGroup := Group{ID: 4, Platform: PlatformLeonardo, Status: StatusActive, SortOrder: 10}
	openAIGroup := Group{ID: 5, Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 1}
	accounts := canvasRoutingAccountRepoByGroup{accounts: map[int64][]Account{
		4: {{Credentials: map[string]any{"model_mapping": map[string]any{"wan-2.7": "wan-2.7"}}}},
		5: {{Credentials: map[string]any{"model_mapping": map[string]any{"minimax-h3": "minimax-h3"}}}},
	}}
	routing := NewCanvasRoutingService(canvasRoutingUserRepo{}, canvasRoutingGroupRepo{groups: []Group{leonardoGroup, openAIGroup}}, canvasRoutingSubscriptionRepo{}, accounts, nil, nil)

	wan, err := routing.Resolve(context.Background(), CanvasRouteRequest{UserID: 1, APIKeyID: 7, Endpoint: "/v1/videos/generations", Model: "wan-2.7"})
	require.NoError(t, err)
	require.Equal(t, PlatformLeonardo, wan.Platform)
	require.Equal(t, int64(4), wan.Group.ID)
	require.Equal(t, "wan-2.7", wan.Model)

	// minimax-h3 is an in-house OpenAI-compatible model (account 73), so it must
	// resolve to the OpenAI platform group, not Leonardo.
	h3, err := routing.Resolve(context.Background(), CanvasRouteRequest{UserID: 1, APIKeyID: 7, Endpoint: "/v1/videos/generations", Model: "minimax-h3"})
	require.NoError(t, err)
	require.Equal(t, PlatformOpenAI, h3.Platform)
	require.Equal(t, int64(5), h3.Group.ID)
	require.Equal(t, "minimax-h3", h3.Model)
}

func TestCanvasRoutingResolvesGenerationJobEntryGroup(t *testing.T) {
	groupID := int64(4)
	group := Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive}
	service := NewCanvasRoutingService(canvasRoutingUserRepo{}, canvasRoutingGroupRepo{groups: []Group{group}}, canvasRoutingSubscriptionRepo{}, canvasRoutingAccountRepo{}, nil, nil)
	service.SetGenerationJobRepository(canvasRoutingJobRepo{job: &GenerationJob{PublicID: "media_rq_1", UserID: 1, APIKeyID: 7, GroupID: &groupID, Model: "flux-schnell"}})

	route, err := service.Resolve(context.Background(), CanvasRouteRequest{UserID: 1, APIKeyID: 7, Endpoint: "/v1/videos/media_rq_1", ResourceID: "media_rq_1"})
	require.NoError(t, err)
	require.Equal(t, groupID, route.Group.ID)
	require.Equal(t, "flux-schnell", route.Model)
	require.Equal(t, "openai-async", route.Protocol)
}
