-- Migration: Add allowed configuration fields to profiles
-- Description: Profiles now directly define which configuration parameters can be modified

ALTER TABLE service_profiles
ADD COLUMN allowed_configuration_fields JSONB DEFAULT '[]'::jsonb;

-- Index for querying by allowed fields
CREATE INDEX idx_service_profiles_allowed_fields ON service_profiles USING gin(allowed_configuration_fields);

-- Comment
COMMENT ON COLUMN service_profiles.allowed_configuration_fields IS 'JSON array of canonical metric names that can be modified (configurable metrics)';

-- Update existing profiles with example allowed fields
UPDATE service_profiles
SET allowed_configuration_fields = '["jvm.heap.max", "jvm.heap.min", "jvm.gc.type", "container.cpu.limit", "container.memory.limit"]'::jsonb
WHERE allowed_configuration_fields IS NULL OR allowed_configuration_fields = '[]'::jsonb;
