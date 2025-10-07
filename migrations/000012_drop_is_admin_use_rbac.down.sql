-- Migration rollback: Restore is_admin column from RBAC roles
-- Description: Re-adds the is_admin boolean flag and populates it from user_roles

-- Step 1: Add the is_admin column back
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;

-- Step 2: Set is_admin=true for users who have the 'admin' role
UPDATE users u
SET is_admin = true
FROM user_roles ur
JOIN roles r ON ur.role_id = r.id
WHERE ur.user_id = u.id
  AND r.name = 'admin'
  AND (ur.expires_at IS NULL OR ur.expires_at > NOW());

-- Step 3: Create index on is_admin
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- Step 4: Add comment
COMMENT ON COLUMN users.is_admin IS 'Admin flag - users with is_admin=true bypass all permission checks';
