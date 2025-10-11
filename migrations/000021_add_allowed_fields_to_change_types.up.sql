-- Migration: Add allowed fields to change types
-- Description: Stores array of allowed canonical metric names for each change type

ALTER TABLE configuration_change_types ADD COLUMN allowed_fields JSONB;

-- Update existing change types with their allowed fields
UPDATE configuration_change_types SET
    allowed_fields = '["jvm.heap.size", "jvm.heap.max", "jvm.gc.algorithm", "jvm.threads.count"]'::jsonb
WHERE code = 'jvm';

UPDATE configuration_change_types SET
    allowed_fields = '["container.cpu.request", "container.cpu.limit", "container.memory.request", "container.memory.limit", "container.replicas"]'::jsonb
WHERE code = 'container';

-- Add index for JSONB queries
CREATE INDEX idx_change_types_allowed_fields ON configuration_change_types USING gin(allowed_fields);

COMMENT ON COLUMN configuration_change_types.allowed_fields IS 'JSON array of allowed canonical metric names for this change type';
