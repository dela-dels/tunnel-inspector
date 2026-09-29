CREATE TABLE IF NOT EXISTS requests (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    query TEXT,
    request_headers TEXT,
    request_body BLOB,
    request_size INTEGER,
    response_status INTEGER,
    response_headers TEXT,
    response_body BLOB,
    response_size INTEGER,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_requests_method ON requests(method);
CREATE INDEX IF NOT EXISTS idx_requests_path ON requests(path);
CREATE INDEX IF NOT EXISTS idx_requests_response_status ON requests(response_status);
