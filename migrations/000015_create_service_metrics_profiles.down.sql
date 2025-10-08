-- Migration rollback: Drop service metric profiles tables

DROP TABLE IF EXISTS service_metric_requirements;
DROP TABLE IF EXISTS service_metric_profiles;
DROP TABLE IF EXISTS metric_profile_templates;
