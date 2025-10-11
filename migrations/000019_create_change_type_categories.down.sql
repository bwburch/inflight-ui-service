-- Rollback: Drop change type categories

-- Remove foreign key constraint and category_id column from configuration_change_types
ALTER TABLE configuration_change_types DROP CONSTRAINT IF EXISTS fk_change_type_category;
DROP INDEX IF EXISTS idx_change_types_category_id;
ALTER TABLE configuration_change_types DROP COLUMN IF EXISTS category_id;

-- Drop the categories table
DROP INDEX IF EXISTS idx_change_type_categories_name;
DROP INDEX IF EXISTS idx_change_type_categories_active;
DROP TABLE IF EXISTS change_type_categories;
