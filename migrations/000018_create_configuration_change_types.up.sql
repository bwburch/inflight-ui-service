-- Migration: Create configuration change types
-- Description: Adds table for managing configuration change type definitions

-- ============================================================================
-- Configuration Change Types Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS configuration_change_types (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,        -- Unique identifier: 'jvm', 'container', 'platform'
    display_name VARCHAR(100) NOT NULL,      -- Human-readable name: 'JVM Configuration'
    description TEXT,                        -- Detailed description of the change type
    category VARCHAR(50),                    -- Category: 'application', 'infrastructure', 'platform'
    is_active BOOLEAN DEFAULT TRUE,          -- Enable/disable change types
    display_order INTEGER DEFAULT 0,         -- Order for UI display
    icon VARCHAR(50),                        -- Optional icon identifier
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_change_types_active ON configuration_change_types(is_active, display_order);
CREATE INDEX idx_change_types_category ON configuration_change_types(category) WHERE is_active = TRUE;

-- Comments
COMMENT ON TABLE configuration_change_types IS 'Defines available configuration change types for workbench simulations';
COMMENT ON COLUMN configuration_change_types.code IS 'Unique code identifier used in API and UI';
COMMENT ON COLUMN configuration_change_types.display_name IS 'Human-readable name shown in UI';
COMMENT ON COLUMN configuration_change_types.category IS 'Groups related change types together';
COMMENT ON COLUMN configuration_change_types.display_order IS 'Sort order for UI display (lower = first)';

-- ============================================================================
-- Seed Data - Initial Change Types
-- ============================================================================

INSERT INTO configuration_change_types (code, display_name, description, category, display_order, icon) VALUES
    (
        'jvm',
        'JVM Configuration',
        'Java Virtual Machine settings including heap size, GC algorithm, thread pools, and memory parameters',
        'application',
        1,
        'cpu'
    ),
    (
        'container',
        'Container Resources',
        'Kubernetes container resource limits and requests for CPU, memory, and replica counts',
        'infrastructure',
        2,
        'box'
    ),
    (
        'platform',
        'Platform Configuration',
        'Platform-level configurations and system-wide settings (future use)',
        'platform',
        3,
        'settings'
    );
