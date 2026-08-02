CREATE TABLE IF NOT EXISTS leonardo_image_funds_reservations (
    id BIGSERIAL PRIMARY KEY,
    reference VARCHAR(128) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    public_id VARCHAR(64) NOT NULL,
    amount_usd DECIMAL(20, 8) NOT NULL,
    pricing_version VARCHAR(64) NOT NULL,
    pricing_source VARCHAR(128) NOT NULL,
    pricing_match_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'reserved',
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT leonardo_image_funds_amount_positive CHECK (amount_usd > 0),
    CONSTRAINT leonardo_image_funds_status_valid CHECK (status IN ('reserved', 'released', 'settled')),
    CONSTRAINT leonardo_image_funds_user_public_unique UNIQUE (user_id, public_id)
);

CREATE INDEX IF NOT EXISTS leonardo_image_funds_user_status_idx
    ON leonardo_image_funds_reservations (user_id, status);

CREATE INDEX IF NOT EXISTS leonardo_image_funds_status_created_idx
    ON leonardo_image_funds_reservations (status, created_at);
