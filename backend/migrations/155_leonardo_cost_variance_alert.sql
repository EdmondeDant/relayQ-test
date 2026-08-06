INSERT INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    filters, created_at, updated_at
) VALUES (
    'Leonardo上游成本偏差过高',
    '当Leonardo实际上游成本与价格快照估价的最大绝对偏差达到20%时触发告警',
    true, 'leonardo_cost_variance_ratio', '>=', 20.0,
    5, 1, 'P1', true, 30,
    '{"platform":"leonardo"}'::jsonb, NOW(), NOW()
) ON CONFLICT (name) DO NOTHING;
