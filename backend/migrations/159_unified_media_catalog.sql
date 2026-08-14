CREATE TABLE IF NOT EXISTS media_products (
    id BIGSERIAL PRIMARY KEY,
    public_model VARCHAR(200) NOT NULL,
    modality VARCHAR(16) NOT NULL CHECK (modality IN ('image', 'video')),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (public_model, modality)
);

CREATE TABLE IF NOT EXISTS media_product_group_bindings (
    product_id BIGINT NOT NULL REFERENCES media_products(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    PRIMARY KEY (product_id, group_id)
);

CREATE TABLE IF NOT EXISTS media_product_prices (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES media_products(id) ON DELETE CASCADE,
    operation VARCHAR(32) NOT NULL,
    spec_key VARCHAR(500) NOT NULL,
    unit_price_usd DECIMAL(20,10) NOT NULL CHECK (unit_price_usd > 0),
    currency VARCHAR(8) NOT NULL DEFAULT 'USD',
    version VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (product_id, operation, spec_key, version)
);

CREATE TABLE IF NOT EXISTS media_offers (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES media_products(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    source_group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    upstream_model VARCHAR(200) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 100,
    operations JSONB NOT NULL DEFAULT '[]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    cost_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    cost_source TEXT NOT NULL,
    cost_version VARCHAR(64) NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, provider, source_group_id, upstream_model)
);

CREATE INDEX IF NOT EXISTS idx_media_product_bindings_group ON media_product_group_bindings(group_id);
CREATE INDEX IF NOT EXISTS idx_media_product_prices_product ON media_product_prices(product_id);
CREATE INDEX IF NOT EXISTS idx_media_offers_product ON media_offers(product_id);
