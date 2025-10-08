-- Migration: Drop is_admin column and fully migrate to RBAC
-- Description: Removes the is_admin boolean flag and ensures all admin users have the 'admin' role in RBAC

-- Step 1: Ensure all users with is_admin=true have the 'admin' role in user_roles
-- Use the user's own ID as assigned_by (self-assigned during migration)
INSERT INTO user_roles (user_id, role_id, assigned_by)
SELECT u.id, r.id, u.id
FROM users u
CROSS JOIN roles r
WHERE u.is_admin = true
  AND r.name = 'admin'
  AND NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
  );

-- Step 2: Drop the is_admin index
DROP INDEX IF EXISTS idx_users_is_admin;

-- Step 3: Drop the is_admin column
ALTER TABLE users DROP COLUMN IF EXISTS is_admin;

-- Comment for documentation
COMMENT ON TABLE users IS 'Users table - permissions managed via RBAC (roles, permissions, user_roles tables)';
