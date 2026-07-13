-- 统一创作记录：最多保留最近 10 条，过期时间改为 5 天

ALTER TABLE playground_tasks ALTER COLUMN expires_at SET DEFAULT (NOW() + INTERVAL '5 days');
ALTER TABLE playground_assets ALTER COLUMN expires_at SET DEFAULT (NOW() + INTERVAL '5 days');

-- 已有数据按新策略收紧过期时间（不超过 5 天）
UPDATE playground_tasks
SET expires_at = LEAST(expires_at, created_at + INTERVAL '5 days')
WHERE expires_at > created_at + INTERVAL '5 days';

UPDATE playground_assets
SET expires_at = LEAST(expires_at, created_at + INTERVAL '5 days')
WHERE expires_at > created_at + INTERVAL '5 days';
