CREATE TABLE IF NOT EXISTS retail_grok_keys (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    group_id BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    token_limit_total BIGINT NOT NULL DEFAULT 0,
    token_used_total BIGINT NOT NULL DEFAULT 0,
    image_limit_total BIGINT NOT NULL DEFAULT 0,
    image_used_total BIGINT NOT NULL DEFAULT 0,
    video_limit_total BIGINT NOT NULL DEFAULT 0,
    video_used_total BIGINT NOT NULL DEFAULT 0,
    created_by_admin_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_retail_grok_keys_group_id ON retail_grok_keys(group_id);
CREATE INDEX IF NOT EXISTS idx_retail_grok_keys_status ON retail_grok_keys(status);
