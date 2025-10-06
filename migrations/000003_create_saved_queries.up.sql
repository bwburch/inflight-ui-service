-- Saved queries for quick access to common metric queries
CREATE TABLE IF NOT EXISTS saved_queries (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query_data JSONB NOT NULL, -- Stores query parameters
    is_shared BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

CREATE INDEX idx_queries_user ON saved_queries(user_id);
CREATE INDEX idx_queries_shared ON saved_queries(is_shared) WHERE is_shared = TRUE;
CREATE INDEX idx_queries_name ON saved_queries(name);
