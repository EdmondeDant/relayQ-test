-- Persist native Search/Audio usage metadata without changing existing billing results.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS search_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS audio_mode VARCHAR(16);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS audio_units DECIMAL(20,10);
