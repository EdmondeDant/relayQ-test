ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS product_id BIGINT REFERENCES media_products(id) ON DELETE RESTRICT;
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS offer_id BIGINT REFERENCES media_offers(id) ON DELETE RESTRICT;
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS source_group_id BIGINT REFERENCES groups(id) ON DELETE RESTRICT;
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS operation VARCHAR(32);
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS customer_price_version VARCHAR(64);

CREATE INDEX IF NOT EXISTS generationjob_product_id ON generation_jobs (product_id);
CREATE INDEX IF NOT EXISTS generationjob_offer_id ON generation_jobs (offer_id);
CREATE INDEX IF NOT EXISTS generationjob_source_group_id ON generation_jobs (source_group_id);

CREATE TABLE IF NOT EXISTS media_job_attempts (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES generation_jobs(id) ON DELETE RESTRICT,
    offer_id BIGINT NOT NULL REFERENCES media_offers(id) ON DELETE RESTRICT,
    provider VARCHAR(32) NOT NULL,
    source_group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    account_id BIGINT,
    upstream_model VARCHAR(200) NOT NULL,
    trusted_cost_snapshot JSONB NOT NULL,
    submission_state VARCHAR(32) NOT NULL,
    error_code VARCHAR(64),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_job_attempts_submission_state_check CHECK (submission_state IN ('not_written', 'submitted', 'side_effect_unknown'))
);

CREATE INDEX IF NOT EXISTS media_job_attempts_job_id ON media_job_attempts (job_id, id);
CREATE INDEX IF NOT EXISTS media_job_attempts_offer_id ON media_job_attempts (offer_id);

CREATE TABLE IF NOT EXISTS media_funds_reservations (
    id BIGSERIAL PRIMARY KEY,
    reference VARCHAR(128) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    public_id VARCHAR(64) NOT NULL,
    product_id BIGINT NOT NULL REFERENCES media_products(id) ON DELETE RESTRICT,
    amount DECIMAL(20, 10) NOT NULL,
    price_version VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'reserved',
    released_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_funds_reservations_amount_check CHECK (amount > 0),
    CONSTRAINT media_funds_reservations_status_check CHECK (status IN ('reserved', 'settled', 'released')),
    CONSTRAINT media_funds_reservations_user_public_key UNIQUE (user_id, public_id)
);

CREATE INDEX IF NOT EXISTS media_funds_reservations_product_id ON media_funds_reservations (product_id);
CREATE INDEX IF NOT EXISTS media_funds_reservations_status ON media_funds_reservations (status);

ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS media_product_id BIGINT REFERENCES media_products(id) ON DELETE RESTRICT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS media_offer_id BIGINT REFERENCES media_offers(id) ON DELETE RESTRICT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS upstream_platform VARCHAR(32);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS source_group_id BIGINT REFERENCES groups(id) ON DELETE RESTRICT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS trusted_cost_amount DECIMAL(20, 10);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS trusted_cost_unit VARCHAR(32);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS trusted_cost_source VARCHAR(500);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS trusted_cost_version VARCHAR(64);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS customer_price_version VARCHAR(64);

CREATE INDEX IF NOT EXISTS usage_logs_media_product_id ON usage_logs (media_product_id);
CREATE INDEX IF NOT EXISTS usage_logs_media_offer_id ON usage_logs (media_offer_id);
CREATE INDEX IF NOT EXISTS usage_logs_source_group_id ON usage_logs (source_group_id);
