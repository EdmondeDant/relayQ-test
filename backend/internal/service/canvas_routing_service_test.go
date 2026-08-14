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
