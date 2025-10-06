-- Add authentication and role fields to users table

ALTER TABLE users
ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255),
ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'user' CHECK (role IN ('admin', 'user', 'viewer')),
ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Update default user to be admin
UPDATE users
SET role = 'admin', is_active = true
WHERE id = 1;
