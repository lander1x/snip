CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS links (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(12) NOT NULL UNIQUE,
    url        TEXT NOT NULL,
    clicks     BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_links_code ON links (code);
CREATE INDEX idx_links_created_at ON links (created_at);
