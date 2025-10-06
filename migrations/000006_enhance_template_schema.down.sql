DROP INDEX IF EXISTS idx_templates_created;
ALTER TABLE quick_templates RENAME COLUMN configuration_data TO template_data;
