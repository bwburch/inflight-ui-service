-- Migration: Add metric filter configuration to change types
-- Description: Maps change types to canonical metric categories for dynamic field loading

-- Add columns to store metric filtering criteria
ALTER TABLE configuration_change_types ADD COLUMN metric_category VARCHAR(100);
ALTER TABLE configuration_change_types ADD COLUMN metric_subcategory VARCHAR(100);
ALTER TABLE configuration_change_types ADD COLUMN metric_name_pattern VARCHAR(255);

-- Update existing change types with metric filters
UPDATE configuration_change_types SET
    metric_category = 'jvm',
    metric_subcategory = NULL,
    metric_name_pattern = NULL
WHERE code = 'jvm';

UPDATE configuration_change_types SET
    metric_category = 'infrastructure',
    metric_subcategory = 'container',
    metric_name_pattern = NULL
WHERE code = 'container';

UPDATE configuration_change_types SET
    metric_category = NULL,
    metric_subcategory = NULL,
    metric_name_pattern = NULL
WHERE code = 'platform';

-- Comments
COMMENT ON COLUMN configuration_change_types.metric_category IS 'Canonical metric category to filter fields from metrics-collector';
COMMENT ON COLUMN configuration_change_types.metric_subcategory IS 'Optional subcategory filter';
COMMENT ON COLUMN configuration_change_types.metric_name_pattern IS 'Optional regex pattern to filter metric names';
