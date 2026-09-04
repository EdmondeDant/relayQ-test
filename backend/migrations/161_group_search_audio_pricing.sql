-- Grok Search/Voice 显式定价：NULL 使用默认价，0 表示免费。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS search_price_per_1k DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN IF NOT EXISTS audio_realtime_price_per_min DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN IF NOT EXISTS audio_tts_price_per_million_chars DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN IF NOT EXISTS audio_stt_price_per_hour DECIMAL(20,8);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'groups_search_price_per_1k_nonnegative') THEN
        ALTER TABLE groups ADD CONSTRAINT groups_search_price_per_1k_nonnegative CHECK (search_price_per_1k >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'groups_audio_realtime_price_per_min_nonnegative') THEN
        ALTER TABLE groups ADD CONSTRAINT groups_audio_realtime_price_per_min_nonnegative CHECK (audio_realtime_price_per_min >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'groups_audio_tts_price_per_million_chars_nonnegative') THEN
        ALTER TABLE groups ADD CONSTRAINT groups_audio_tts_price_per_million_chars_nonnegative CHECK (audio_tts_price_per_million_chars >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'groups_audio_stt_price_per_hour_nonnegative') THEN
        ALTER TABLE groups ADD CONSTRAINT groups_audio_stt_price_per_hour_nonnegative CHECK (audio_stt_price_per_hour >= 0);
    END IF;
END $$;
