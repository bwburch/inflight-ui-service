-- Rollback: Drop configuration change types table

DROP INDEX IF EXISTS idx_change_types_category;
DROP INDEX IF EXISTS idx_change_types_active;
DROP TABLE IF EXISTS configuration_change_types;
