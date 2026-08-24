package service

import (
	"context"
	"testing"
)

func TestProvideOpenAIGatewayServiceWiresCanvasResourceRoutes(t *testing.T) {
	routes := &stubCanvasResourceRouteRepository{}
	svc := ProvideOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, routes)
	if svc.canvasResourceRoutes != routes {
		t.Fatal("canvas resource route repository was not wired")
	}
}

type stubCanvasResourceRouteRepository struct{}

func (*stubCanvasResourceRouteRepository) GetActive(_ context.Context, _, _ int64, _ string) (*CanvasResourceRoute, error) {
	return nil, nil
}

func (*stubCanvasResourceRouteRepository) Upsert(_ context.Context, _ *CanvasResourceRoute) error {
	return nil
}
