-- Rollback: Recreate change_type_fields table (empty, just for rollback compatibility)

CREATE TABLE IF NOT EXISTS change_type_fields (
    id SERIAL PRIMARY KEY,
    change_type_id INTEGER NOT NULL REFERENCES configuration_change_types(id) ON DELETE CASCADE,
    field_name VARCHAR(100) NOT NULL,
    field_label VARCHAR(100) NOT NULL,
    field_type VARCHAR(50) NOT NULL,
    has_unit BOOLEAN DEFAULT FALSE,
    default_unit VARCHAR(20),
    available_units TEXT,
    validation_rules JSONB,
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,
    CONSTRAINT unique_field_per_type UNIQUE (change_type_id, field_name)
);

CREATE INDEX idx_change_type_fields_type_id ON change_type_fields(change_type_id);
CREATE INDEX idx_change_type_fields_active ON change_type_fields(is_active, display_order);
