-- Migration: Drop obsolete change_type_fields table
-- Description: Remove the change_type_fields table as we now reference canonical_metrics directly

-- Drop the table and its indexes
DROP INDEX IF EXISTS idx_change_type_fields_active;
DROP INDEX IF EXISTS idx_change_type_fields_type_id;
DROP TABLE IF EXISTS change_type_fields;
