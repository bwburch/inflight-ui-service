-- Migration: Create RBAC (Role-Based Access Control) tables
-- Description: Adds roles, permissions, and user-role mapping for granular access control

-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  description TEXT,
  is_system BOOLEAN DEFAULT false,  -- System roles cannot be deleted
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Create permissions table
CREATE TABLE IF NOT EXISTS permissions (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,  -- e.g., 'services.view', 'models.edit'
  resource VARCHAR(50) NOT NULL,      -- e.g., 'services', 'models', 'metrics'
  action VARCHAR(50) NOT NULL,        -- e.g., 'view', 'create', 'edit', 'delete'
  description TEXT,
  category VARCHAR(50),               -- e.g., 'navigation', 'data', 'configuration', 'admin'
  created_at TIMESTAMP DEFAULT NOW()
);

-- Create role-permission mapping table (many-to-many)
CREATE TABLE IF NOT EXISTS role_permissions (
  id SERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  granted_at TIMESTAMP DEFAULT NOW(),
  granted_by INTEGER REFERENCES users(id),
  UNIQUE(role_id, permission_id)
);

-- Create user-role mapping table (many-to-many)
CREATE TABLE IF NOT EXISTS user_roles (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  assigned_at TIMESTAMP DEFAULT NOW(),
  assigned_by INTEGER REFERENCES users(id),
  expires_at TIMESTAMP,  -- Optional: time-limited role assignments
  UNIQUE(user_id, role_id)
);

-- Create permission audit log table
CREATE TABLE IF NOT EXISTS permission_audit (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  role_id INTEGER,
  permission_id INTEGER,
  action VARCHAR(50),  -- 'role_granted', 'role_revoked', 'permission_granted', 'permission_revoked'
  changed_by INTEGER REFERENCES users(id),
  timestamp TIMESTAMP DEFAULT NOW(),
  metadata JSONB
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_permission_audit_user ON permission_audit(user_id);
CREATE INDEX IF NOT EXISTS idx_permission_audit_timestamp ON permission_audit(timestamp DESC);

-- Add updated_at trigger for roles table
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE roles IS 'User roles for RBAC system';
COMMENT ON TABLE permissions IS 'Available permissions in the system';
COMMENT ON TABLE role_permissions IS 'Maps permissions to roles (many-to-many)';
COMMENT ON TABLE user_roles IS 'Maps users to roles (many-to-many)';
COMMENT ON TABLE permission_audit IS 'Audit log for all permission changes';
