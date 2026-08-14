UPDATE api_keys
SET group_id = NULL, updated_at = NOW()
WHERE deleted_at IS NULL
  AND managed = TRUE
  AND client_app = 'infinite-canvas'
  AND managed_purpose = 'canvas_bootstrap';

CREATE TABLE IF NOT EXISTS canvas_resource_routes (
    id BIGSERIAL PRIMARY KEY,
    api_key_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    group_id BIGINT NOT NULL,
    platform VARCHAR(32) NOT NULL,
    model VARCHAR(200) NOT NULL,
    endpoint_family VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS canvas_resource_routes_key_resource
    ON canvas_resource_routes (api_key_id, resource_id);
CREATE INDEX IF NOT EXISTS canvas_resource_routes_expires_at
    ON canvas_resource_routes (expires_at);
