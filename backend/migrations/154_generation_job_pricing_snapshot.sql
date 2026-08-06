ALTER TABLE generation_jobs
    ADD COLUMN IF NOT EXISTS estimated_upstream_cost_amount DECIMAL(20, 10),
    ADD COLUMN IF NOT EXISTS estimated_upstream_cost_unit VARCHAR(32),
    ADD COLUMN IF NOT EXISTS pricing_snapshot_version VARCHAR(64),
    ADD COLUMN IF NOT EXISTS pricing_source VARCHAR(128),
	ADD COLUMN IF NOT EXISTS pricing_match_type VARCHAR(32),
	ADD COLUMN IF NOT EXISTS gross_margin DECIMAL(20, 10),
	ADD COLUMN IF NOT EXISTS cost_variance DECIMAL(20, 10);
