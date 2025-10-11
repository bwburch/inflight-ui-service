-- Rollback: Remove allowed_fields column

DROP INDEX IF EXISTS idx_change_types_allowed_fields;
ALTER TABLE configuration_change_types DROP COLUMN IF EXISTS allowed_fields;
