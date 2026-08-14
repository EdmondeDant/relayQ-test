package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mediaProductHandlerRepo struct{}

func (mediaProductHandlerRepo) List(context.Context, int, int, string, string) ([]service.MediaCatalogProduct, int64, error) {
	return []service.MediaCatalogProduct{{ID: 7, PublicModel: "relayq-video-v1", Modality: "video"}}, 1, nil
}
func (mediaProductHandlerRepo) GetByID(context.Context, int64) (*service.MediaCatalogProduct, error) {
	return nil, service.ErrMediaCatalogProductNotFound
}
func (mediaProductHandlerRepo) GetRuntime(context.Context, int64, string, string, time.Time) (*service.MediaCatalogProduct, error) {
	return nil, service.ErrMediaCatalogProductNotFound
}
func (mediaProductHandlerRepo) ListRuntimeModels(context.Context, int64, time.Time) ([]string, error) {
	return nil, nil
}
func (mediaProductHandlerRepo) GetGroups(context.Context, []int64) (map[int64]service.MediaCatalogGroup, error) {
	return nil, nil
}
func (mediaProductHandlerRepo) Create(context.Context, *service.MediaCatalogProduct) error {
	return nil
}
func (mediaProductHandlerRepo) Update(context.Context, *service.MediaCatalogProduct) error {
	return nil
}
func (mediaProductHandlerRepo) Disable(context.Context, int64) error { return nil }

func TestMediaProductHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewMediaProductHandler(service.NewMediaCatalogService(mediaProductHandlerRepo{}))
	router.GET("/media-products", handler.List)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/media-products?page=1&page_size=10&modality=video", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"public_model":"relayq-video-v1"`)
	require.Contains(t, recorder.Body.String(), `"total":1`)
}

func TestMediaProductHandlerRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewMediaProductHandler(service.NewMediaCatalogService(mediaProductHandlerRepo{}))
	router.GET("/media-products/:id", handler.Get)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/media-products/invalid", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_MEDIA_PRODUCT_ID")
}
