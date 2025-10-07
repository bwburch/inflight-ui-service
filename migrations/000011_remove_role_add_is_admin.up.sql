-- Migration: Remove old role column and add is_admin flag
-- Description: Migrates from old role-based system to pure RBAC with is_admin flag for convenience

-- Add is_admin column (default false)
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;

-- Set is_admin=true for users with role='admin'
UPDATE users SET is_admin = true WHERE role = 'admin';

-- Drop the old role column and its check constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- Drop the role index if it exists
DROP INDEX IF EXISTS idx_users_role;

-- Create index on is_admin for performance
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- Comment for documentation
COMMENT ON COLUMN users.is_admin IS 'Admin flag - users with is_admin=true bypass all permission checks';
