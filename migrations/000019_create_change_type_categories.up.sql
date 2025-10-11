-- Migration: Create change type categories
-- Description: Normalizes categories into a separate table with management capabilities

-- ============================================================================
-- Change Type Categories Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS change_type_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,           -- e.g., "application", "infrastructure"
    display_name VARCHAR(100) NOT NULL,          -- e.g., "Application", "Infrastructure"
    description TEXT,
    color VARCHAR(50),                            -- Hex color or Tailwind color class
    icon VARCHAR(50),                             -- Optional icon identifier
    display_order INTEGER DEFAULT 0,              -- Sort order
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_change_type_categories_active ON change_type_categories(is_active, display_order);
CREATE INDEX idx_change_type_categories_name ON change_type_categories(name);

-- Comments
COMMENT ON TABLE change_type_categories IS 'Manages categories for configuration change types';
COMMENT ON COLUMN change_type_categories.name IS 'Unique lowercase identifier for the category';
COMMENT ON COLUMN change_type_categories.display_name IS 'Human-readable name shown in UI';

-- ============================================================================
-- Seed Default Categories
-- ============================================================================

INSERT INTO change_type_categories (name, display_name, description, color, icon, display_order) VALUES
    ('application', 'Application', 'Application-level configurations (JVM, runtime, etc.)', 'blue', 'cpu', 1),
    ('infrastructure', 'Infrastructure', 'Infrastructure and container resources', 'green', 'box', 2),
    ('platform', 'Platform', 'Platform-wide configurations and settings', 'purple', 'settings', 3),
    ('network', 'Network', 'Network and connectivity configurations', 'orange', 'network', 4),
    ('security', 'Security', 'Security and access control settings', 'red', 'shield', 5);

-- ============================================================================
-- Update configuration_change_types to use foreign key
-- ============================================================================

-- Add new category_id column
ALTER TABLE configuration_change_types ADD COLUMN category_id INTEGER;

-- Migrate existing category strings to foreign keys
UPDATE configuration_change_types
SET category_id = (
    SELECT id FROM change_type_categories WHERE name = configuration_change_types.category
)
WHERE category IS NOT NULL AND category != '';

-- Add foreign key constraint
ALTER TABLE configuration_change_types
ADD CONSTRAINT fk_change_type_category
FOREIGN KEY (category_id) REFERENCES change_type_categories(id) ON DELETE SET NULL;

-- Create index on foreign key
CREATE INDEX idx_change_types_category_id ON configuration_change_types(category_id);

-- Keep the old category column for now (we can drop it in a future migration after verification)
-- ALTER TABLE configuration_change_types DROP COLUMN category;

COMMENT ON COLUMN configuration_change_types.category_id IS 'Foreign key to change_type_categories table';
