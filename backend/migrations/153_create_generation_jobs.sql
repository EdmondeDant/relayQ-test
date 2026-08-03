CREATE TABLE IF NOT EXISTS generation_jobs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    public_id VARCHAR(64) NOT NULL UNIQUE,
    provider VARCHAR(32) NOT NULL,
    modality VARCHAR(32) NOT NULL,
    model VARCHAR(200) NOT NULL,
    upstream_model VARCHAR(200) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    group_id BIGINT,
    account_id BIGINT NOT NULL,
    upstream_generation_id VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'created',
    upstream_status VARCHAR(64),
    request_hash VARCHAR(128) NOT NULL,
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_code VARCHAR(64),
    error_message TEXT,
    output_count INTEGER NOT NULL DEFAULT 0,
    actual_upstream_cost_amount DECIMAL(20, 10),
    actual_upstream_cost_unit VARCHAR(32),
    customer_cost DECIMAL(20, 10),
    billing_status VARCHAR(16) NOT NULL DEFAULT 'unpriced',
    billing_reference VARCHAR(128),
    poll_attempts INTEGER NOT NULL DEFAULT 0,
    next_poll_at TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    CONSTRAINT generation_jobs_status_check CHECK (status IN ('created', 'submitting', 'queued', 'running', 'succeeded', 'failed', 'cancelled', 'unknown')),
    CONSTRAINT generation_jobs_billing_status_check CHECK (billing_status IN ('unpriced', 'estimated', 'reserved', 'submitted', 'settled', 'refunded', 'manual_review')),
    CONSTRAINT generation_jobs_output_count_check CHECK (output_count >= 0),
    CONSTRAINT generation_jobs_poll_attempts_check CHECK (poll_attempts >= 0)
);

CREATE INDEX IF NOT EXISTS generationjob_provider ON generation_jobs (provider);
CREATE INDEX IF NOT EXISTS generationjob_modality ON generation_jobs (modality);
CREATE INDEX IF NOT EXISTS generationjob_user_id ON generation_jobs (user_id);
CREATE INDEX IF NOT EXISTS generationjob_api_key_id ON generation_jobs (api_key_id);
CREATE INDEX IF NOT EXISTS generationjob_group_id ON generation_jobs (group_id);
CREATE INDEX IF NOT EXISTS generationjob_account_id ON generation_jobs (account_id);
CREATE INDEX IF NOT EXISTS generationjob_upstream_generation_id ON generation_jobs (upstream_generation_id);
CREATE INDEX IF NOT EXISTS generationjob_status ON generation_jobs (status);
CREATE INDEX IF NOT EXISTS generationjob_next_poll_at ON generation_jobs (next_poll_at);
CREATE INDEX IF NOT EXISTS generationjob_status_next_poll_at ON generation_jobs (status, next_poll_at);
CREATE UNIQUE INDEX IF NOT EXISTS generationjob_billing_reference ON generation_jobs (billing_reference);
CREATE INDEX IF NOT EXISTS generationjob_created_at ON generation_jobs (created_at);
