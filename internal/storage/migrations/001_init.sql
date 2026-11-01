CREATE TABLE IF NOT EXISTS endpoints (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    event_types TEXT NOT NULL DEFAULT '[]',
    rate_limit INTEGER NOT NULL DEFAULT 100,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    tripped_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_endpoints_status ON endpoints(status);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT UNIQUE,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_event_type ON messages(event_type);
CREATE INDEX IF NOT EXISTS idx_messages_idempotency_key ON messages(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

CREATE TABLE IF NOT EXISTS deliveries (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    endpoint_id TEXT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'PENDING',
    attempt_number INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    http_status INTEGER,
    duration_ms INTEGER,
    request_headers TEXT,
    response_headers TEXT,
    response_body TEXT,
    error_message TEXT,
    next_retry_at DATETIME,
    sent_at DATETIME,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_deliveries_status ON deliveries(status);
CREATE INDEX IF NOT EXISTS idx_deliveries_endpoint_id ON deliveries(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_message_id ON deliveries(message_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_retry ON deliveries(status, next_retry_at);
CREATE INDEX IF NOT EXISTS idx_deliveries_created_at ON deliveries(created_at);
