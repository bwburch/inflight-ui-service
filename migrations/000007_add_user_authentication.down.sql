-- Remove authentication fields

DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_role;

ALTER TABLE users
DROP COLUMN IF EXISTS is_active,
DROP COLUMN IF EXISTS role,
DROP COLUMN IF EXISTS password_hash;
