CREATE TABLE IF NOT EXISTS playground_tasks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    model VARCHAR(100) NOT NULL DEFAULT '',
    request_id VARCHAR(255),
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT playground_tasks_status_check CHECK (status IN ('pending', 'running', 'submitted', 'succeeded', 'failed', 'canceled'))
);

CREATE INDEX IF NOT EXISTS idx_playground_tasks_user_created ON playground_tasks(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_playground_tasks_status_created ON playground_tasks(status, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_playground_tasks_user_request ON playground_tasks(user_id, request_id) WHERE request_id IS NOT NULL AND request_id <> '';

CREATE TABLE IF NOT EXISTS playground_assets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id BIGINT REFERENCES playground_tasks(id) ON DELETE SET NULL,
    kind VARCHAR(40) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    content TEXT,
    url TEXT,
    storage_key TEXT,
    content_type VARCHAR(120),
    byte_size BIGINT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT playground_assets_content_check CHECK (content IS NOT NULL OR url IS NOT NULL OR storage_key IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_playground_assets_user_created ON playground_assets(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_playground_assets_task ON playground_assets(task_id);
