ALTER TABLE playground_tasks ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE playground_assets ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

UPDATE playground_tasks SET expires_at = created_at + INTERVAL '30 days' WHERE expires_at IS NULL;
UPDATE playground_assets SET expires_at = created_at + INTERVAL '30 days' WHERE expires_at IS NULL;

ALTER TABLE playground_tasks ALTER COLUMN expires_at SET DEFAULT (NOW() + INTERVAL '30 days');
ALTER TABLE playground_tasks ALTER COLUMN expires_at SET NOT NULL;
ALTER TABLE playground_assets ALTER COLUMN expires_at SET DEFAULT (NOW() + INTERVAL '30 days');
ALTER TABLE playground_assets ALTER COLUMN expires_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_playground_tasks_expires_at ON playground_tasks(expires_at);
CREATE INDEX IF NOT EXISTS idx_playground_assets_expires_at ON playground_assets(expires_at);
