package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type canvasResourceRouteRepository struct {
	db *sql.DB
}

func NewCanvasResourceRouteRepository(db *sql.DB) service.CanvasResourceRouteRepository {
	return &canvasResourceRouteRepository{db: db}
}

func (r *canvasResourceRouteRepository) GetActive(ctx context.Context, userID, apiKeyID int64, resourceID string) (*service.CanvasResourceRoute, error) {
	route := &service.CanvasResourceRoute{}
	err := r.db.QueryRowContext(ctx, `
		SELECT api_key_id, user_id, resource_id, group_id, platform, model, endpoint_family, expires_at
		FROM canvas_resource_routes
		WHERE user_id = $1 AND api_key_id = $2 AND resource_id = $3 AND expires_at > NOW()
	`, userID, apiKeyID, resourceID).Scan(&route.APIKeyID, &route.UserID, &route.ResourceID, &route.GroupID, &route.Platform, &route.Model, &route.EndpointFamily, &route.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return route, err
}

func (r *canvasResourceRouteRepository) Upsert(ctx context.Context, route *service.CanvasResourceRoute) error {
	if route == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO canvas_resource_routes (api_key_id, user_id, resource_id, group_id, platform, model, endpoint_family, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (api_key_id, resource_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			group_id = EXCLUDED.group_id,
			platform = EXCLUDED.platform,
			model = EXCLUDED.model,
			endpoint_family = EXCLUDED.endpoint_family,
			expires_at = EXCLUDED.expires_at
	`, route.APIKeyID, route.UserID, route.ResourceID, route.GroupID, route.Platform, route.Model, route.EndpointFamily, route.ExpiresAt)
	return err
}
