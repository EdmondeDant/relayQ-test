CREATE TABLE IF NOT EXISTS retail_grok_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    retail_grok_key_id BIGINT NOT NULL,
    request_id VARCHAR(255) NOT NULL DEFAULT '',
    inbound_endpoint VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL DEFAULT '',
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    image_count BIGINT NOT NULL DEFAULT 0,
    video_count BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'success',
    error_message TEXT NOT NULL DEFAULT '',
    upstream_model VARCHAR(255) NOT NULL DEFAULT '',
    upstream_request_id VARCHAR(255) NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_retail_grok_usage_logs_key_id ON retail_grok_usage_logs(retail_grok_key_id);
CREATE INDEX IF NOT EXISTS idx_retail_grok_usage_logs_request_id ON retail_grok_usage_logs(request_id);
