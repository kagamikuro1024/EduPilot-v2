CREATE TABLE IF NOT EXISTS analytics_privacy_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(64) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    reason_code VARCHAR(128),
    confidence FLOAT,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_privacy_events_created_at
    ON analytics_privacy_events(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_privacy_events_event_type
    ON analytics_privacy_events(event_type);
