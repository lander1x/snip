CREATE DATABASE IF NOT EXISTS snip;

CREATE TABLE IF NOT EXISTS snip.click_events (
    event_id   UUID DEFAULT generateUUIDv4(),
    code       String,
    ip         String,
    user_agent String,
    referer    String,
    country    String DEFAULT '',
    timestamp  DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (code, timestamp)
PARTITION BY toYYYYMM(timestamp);
