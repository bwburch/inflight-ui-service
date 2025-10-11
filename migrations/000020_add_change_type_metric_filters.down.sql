-- Rollback: Remove metric filter columns

ALTER TABLE configuration_change_types DROP COLUMN IF EXISTS metric_name_pattern;
ALTER TABLE configuration_change_types DROP COLUMN IF EXISTS metric_subcategory;
ALTER TABLE configuration_change_types DROP COLUMN IF EXISTS metric_category;
