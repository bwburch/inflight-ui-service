-- Migration: Drop configuration change types tables
-- Description: Remove configuration_change_types and related tables as they are no longer needed
--              Profiles now directly contain allowed_configuration_fields

-- Drop junction table first (has foreign keys)
DROP TABLE IF EXISTS change_type_profiles CASCADE;

-- Drop categories table
DROP TABLE IF EXISTS change_type_categories CASCADE;

-- Drop main change types table
DROP TABLE IF EXISTS configuration_change_types CASCADE;
