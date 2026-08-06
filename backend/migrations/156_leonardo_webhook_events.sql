CREATE TABLE IF NOT EXISTS leonardo_webhook_events (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    event_key VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL DEFAULT '',
    upstream_generation_id VARCHAR(128),
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT leonardo_webhook_events_status_check CHECK (status IN ('pending', 'processing', 'processed', 'failed', 'dead_letter')),
    CONSTRAINT leonardo_webhook_events_account_event_unique UNIQUE (account_id, event_key)
);

CREATE INDEX IF NOT EXISTS idx_leonardo_webhook_events_pending
    ON leonardo_webhook_events (status, next_attempt_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS idx_leonardo_webhook_events_generation
    ON leonardo_webhook_events (upstream_generation_id)
    WHERE upstream_generation_id IS NOT NULL;
