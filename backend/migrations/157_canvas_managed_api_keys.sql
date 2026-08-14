ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS client_app VARCHAR(50) NOT NULL DEFAULT 'api';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS managed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS managed_purpose VARCHAR(50);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS request_source VARCHAR(50);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_canvas_managed
    ON api_keys (user_id, client_app, managed_purpose)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usage_logs_request_source_created_at
    ON usage_logs (request_source, created_at);

CREATE OR REPLACE FUNCTION set_usage_log_request_source()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.request_source IS NULL OR NEW.request_source = '' THEN
        SELECT NULLIF(client_app, 'api') INTO NEW.request_source
        FROM api_keys
        WHERE id = NEW.api_key_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_usage_logs_request_source ON usage_logs;
CREATE TRIGGER trg_usage_logs_request_source
    BEFORE INSERT ON usage_logs
    FOR EACH ROW EXECUTE FUNCTION set_usage_log_request_source();
