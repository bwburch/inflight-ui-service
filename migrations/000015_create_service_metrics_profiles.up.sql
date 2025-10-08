-- Migration: Create service metric profiles
-- Description: Service-specific metric requirements and profiles for validation

-- ============================================================================
-- Service Metric Profiles Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS service_metric_profiles (
    id SERIAL PRIMARY KEY,
    service_id VARCHAR(255) NOT NULL UNIQUE,

    -- Profile configuration
    profile_type VARCHAR(50) NOT NULL DEFAULT 'custom', -- 'batch', 'high_throughput', 'streaming', 'custom'
    required_metrics TEXT[] NOT NULL DEFAULT '{}', -- Array of canonical metric names
    optional_metrics TEXT[] NOT NULL DEFAULT '{}',
    sampling_rate INTEGER DEFAULT 60, -- seconds

    -- Metadata
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,

    -- Constraints
    CONSTRAINT valid_profile_type CHECK (profile_type IN ('batch', 'high_throughput', 'streaming', 'custom'))
);

-- ============================================================================
-- Service Metric Requirements Table (granular control)
-- ============================================================================

CREATE TABLE IF NOT EXISTS service_metric_requirements (
    id SERIAL PRIMARY KEY,
    service_id VARCHAR(255) NOT NULL,
    canonical_metric_name VARCHAR(255) NOT NULL,

    -- Requirement details
    is_required BOOLEAN DEFAULT true,
    min_sample_rate INTEGER, -- Minimum samples per hour
    max_age_minutes INTEGER DEFAULT 5, -- Max age before considered stale

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,

    -- Unique constraint
    UNIQUE(service_id, canonical_metric_name)
);

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX idx_service_metric_profiles_service_id ON service_metric_profiles(service_id);
CREATE INDEX idx_service_metric_profiles_type ON service_metric_profiles(profile_type);
CREATE INDEX idx_service_metric_requirements_service_id ON service_metric_requirements(service_id);
CREATE INDEX idx_service_metric_requirements_metric_name ON service_metric_requirements(canonical_metric_name);

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE service_metric_profiles IS 'Service-specific metric profile configurations';
COMMENT ON COLUMN service_metric_profiles.profile_type IS 'Pre-defined profile type or custom';
COMMENT ON COLUMN service_metric_profiles.required_metrics IS 'Array of canonical metric names that must be present';

COMMENT ON TABLE service_metric_requirements IS 'Granular metric requirements per service';
COMMENT ON COLUMN service_metric_requirements.is_required IS 'If true, simulation is blocked without this metric';
COMMENT ON COLUMN service_metric_requirements.min_sample_rate IS 'Minimum data points per hour for metric';

-- ============================================================================
-- Seed Default Profiles (as reference data)
-- ============================================================================

-- Note: These are templates, not actual service configs
-- Users can reference these when creating custom profiles

CREATE TABLE IF NOT EXISTS metric_profile_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    profile_type VARCHAR(50) NOT NULL,
    description TEXT,
    required_metrics TEXT[] NOT NULL,
    optional_metrics TEXT[] NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO metric_profile_templates (name, profile_type, description, required_metrics, optional_metrics) VALUES
('Batch Service', 'batch', 'For batch processing services with periodic job execution',
    ARRAY['jvm.gc.pause.p99', 'jvm.heap.used', 'batch.job.duration', 'batch.records.processed'],
    ARRAY['jvm.threads.count', 'db.connections.active', 'jvm.gc.overhead']
),
('High Throughput', 'high_throughput', 'For high-throughput request/response services',
    ARRAY['jvm.gc.pause.p99', 'cpu.usage', 'http.requests.rate', 'http.response.time.p99'],
    ARRAY['jvm.safepoint.time', 'network.bytes.sent', 'jvm.threads.count']
),
('Streaming', 'streaming', 'For event-driven streaming services',
    ARRAY['kafka.lag', 'jvm.gc.pause.p99', 'messages.processed.rate'],
    ARRAY['jvm.heap.used', 'cpu.usage', 'kafka.consumer.offset']
);

COMMENT ON TABLE metric_profile_templates IS 'Pre-defined metric profile templates for reference';
