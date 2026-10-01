-- Migration V15: Add analytics_daily_summary table for security and engagement metrics
-- This table tracks daily performance of the Privacy Firewall and Chat Module.

CREATE TABLE IF NOT EXISTS analytics_daily_summary (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_date               DATE UNIQUE NOT NULL,
    sensitive_detections      INT DEFAULT 0,
    channel_switch_rate       FLOAT DEFAULT 0.0,
    public_leak_prevent_count INT DEFAULT 0,
    false_positive_rate       FLOAT DEFAULT 0.0,
    created_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for date-based lookups
CREATE INDEX IF NOT EXISTS idx_analytics_daily_summary_report_date ON analytics_daily_summary(report_date);
