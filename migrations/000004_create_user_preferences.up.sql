-- User preferences for UI customization
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(50) DEFAULT 'dark', -- dark | light
    default_service_id VARCHAR(255),
    preferences_data JSONB, -- Flexible JSON for future settings
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);
