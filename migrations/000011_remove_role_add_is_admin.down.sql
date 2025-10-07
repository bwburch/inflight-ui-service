-- Rollback: Restore role column and remove is_admin

-- Add back the role column
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'user';

-- Restore role values from is_admin flag
UPDATE users SET role = 'admin' WHERE is_admin = true;

-- Add back the check constraint
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('admin', 'user', 'viewer'));

-- Recreate role index
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Drop is_admin column and index
DROP INDEX IF EXISTS idx_users_is_admin;
ALTER TABLE users DROP COLUMN IF EXISTS is_admin;
