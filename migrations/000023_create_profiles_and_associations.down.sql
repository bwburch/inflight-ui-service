-- Rollback: Drop profiles and associations

DROP INDEX IF EXISTS idx_change_type_profiles_default;
DROP INDEX IF EXISTS idx_change_type_profiles_profile;
DROP INDEX IF EXISTS idx_change_type_profiles_change_type;
DROP TABLE IF EXISTS change_type_profiles;

DROP INDEX IF EXISTS idx_service_profiles_required_metrics;
DROP INDEX IF EXISTS idx_service_profiles_name;
DROP INDEX IF EXISTS idx_service_profiles_active;
DROP TABLE IF EXISTS service_profiles;
