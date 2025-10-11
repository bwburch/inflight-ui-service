-- Migration: Create profiles and change type associations
-- Description: Manages service profiles with required metrics and associates them with change types

-- ============================================================================
-- Profiles Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS service_profiles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,           -- e.g., "Batch Service", "High Throughput"
    display_name VARCHAR(100) NOT NULL,          -- Human-readable name
    description TEXT,
    required_metrics JSONB,                       -- Array of required canonical metric names
    recommended_metrics JSONB,                    -- Array of recommended canonical metric names
    icon VARCHAR(50),
    color VARCHAR(50),                            -- Badge color
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_service_profiles_active ON service_profiles(is_active, display_order);
CREATE INDEX idx_service_profiles_name ON service_profiles(name);
CREATE INDEX idx_service_profiles_required_metrics ON service_profiles USING gin(required_metrics);

-- Comments
COMMENT ON TABLE service_profiles IS 'Service profile types with metric requirements';
COMMENT ON COLUMN service_profiles.required_metrics IS 'JSON array of required canonical metric names';
COMMENT ON COLUMN service_profiles.recommended_metrics IS 'JSON array of recommended canonical metric names';

-- ============================================================================
-- Change Type Profiles Junction Table (Many-to-Many)
-- ============================================================================

CREATE TABLE IF NOT EXISTS change_type_profiles (
    id SERIAL PRIMARY KEY,
    change_type_id INTEGER NOT NULL REFERENCES configuration_change_types(id) ON DELETE CASCADE,
    profile_id INTEGER NOT NULL REFERENCES service_profiles(id) ON DELETE CASCADE,
    is_default BOOLEAN DEFAULT FALSE,             -- Whether this is the default profile for this change type
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ensure unique combinations
    CONSTRAINT unique_change_type_profile UNIQUE (change_type_id, profile_id)
);

-- Indexes
CREATE INDEX idx_change_type_profiles_change_type ON change_type_profiles(change_type_id);
CREATE INDEX idx_change_type_profiles_profile ON change_type_profiles(profile_id);
CREATE INDEX idx_change_type_profiles_default ON change_type_profiles(change_type_id, is_default) WHERE is_default = TRUE;

COMMENT ON TABLE change_type_profiles IS 'Associates profiles with change types (many-to-many)';

-- ============================================================================
-- Seed Default Profiles
-- ============================================================================

INSERT INTO service_profiles (name, display_name, description, required_metrics, recommended_metrics, icon, color, display_order) VALUES
    (
        'batch_service',
        'Batch Service',
        'For batch processing services with periodic job execution',
        '["app.throughput", "app.latency.p95", "jvm.memory.heap.used", "container.cpu.usage.percent"]'::jsonb,
        '["jvm.gc.collection.time", "container.memory.usage.percent"]'::jsonb,
        'layers',
        'blue',
        1
    ),
    (
        'high_throughput',
        'High Throughput',
        'For high-throughput request/response services',
        '["app.throughput", "app.latency.p95", "app.request.error.rate", "container.cpu.usage.percent"]'::jsonb,
        '["jvm.gc.pause.p99", "container.memory.usage.percent"]'::jsonb,
        'zap',
        'green',
        2
    ),
    (
        'streaming',
        'Streaming',
        'For event-driven streaming services',
        '["app.latency.p99", "jvm.gc.pause.p99", "container.memory.usage.percent"]'::jsonb,
        '["jvm.threads.count", "container.cpu.usage.percent"]'::jsonb,
        'activity',
        'purple',
        3
    );

-- ============================================================================
-- Associate Profiles with Change Types
-- ============================================================================

-- JVM change type gets all profiles (since all profiles have JVM metrics)
INSERT INTO change_type_profiles (change_type_id, profile_id, is_default)
SELECT ct.id, sp.id, (sp.name = 'high_throughput')
FROM configuration_change_types ct
CROSS JOIN service_profiles sp
WHERE ct.code = 'jvm';

-- Container change type gets all profiles (since all profiles have container metrics)
INSERT INTO change_type_profiles (change_type_id, profile_id, is_default)
SELECT ct.id, sp.id, (sp.name = 'high_throughput')
FROM configuration_change_types ct
CROSS JOIN service_profiles sp
WHERE ct.code = 'container';
