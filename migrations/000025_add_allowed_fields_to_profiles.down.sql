-- Migration: Remove allowed configuration fields from profiles (rollback)

DROP INDEX IF EXISTS idx_service_profiles_allowed_fields;

ALTER TABLE service_profiles
DROP COLUMN IF EXISTS allowed_configuration_fields;
