-- Quick templates for workbench configuration presets
CREATE TABLE IF NOT EXISTS quick_templates (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_data JSONB NOT NULL, -- Stores proposed_changes array
    is_shared BOOLEAN DEFAULT FALSE, -- Team-wide vs personal
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

CREATE INDEX idx_templates_user ON quick_templates(user_id);
CREATE INDEX idx_templates_shared ON quick_templates(is_shared) WHERE is_shared = TRUE;
CREATE INDEX idx_templates_name ON quick_templates(name);
