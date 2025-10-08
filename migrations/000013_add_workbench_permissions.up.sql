-- Migration: Add workbench permissions
-- Description: Adds permissions for Scientific Workbench simulation features

-- ============================================================================
-- Insert Workbench Permissions
-- ============================================================================

-- Insert workbench permissions
INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('workbench.run_simulation', 'workbench', 'run', 'workbench', 'Run JVM configuration simulations'),
  ('workbench.view_history', 'workbench', 'view', 'workbench', 'View simulation history and results')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Assign Workbench Permissions to Roles
-- ============================================================================

-- Grant workbench permissions to roles (only if roles exist)

-- ADMIN: Grant all workbench permissions
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'admin'
    AND p.resource = 'workbench'
    AND EXISTS (SELECT 1 FROM roles WHERE name = 'admin')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ANALYST: Grant all workbench permissions (they need to run simulations)
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'analyst'
    AND p.resource = 'workbench'
    AND EXISTS (SELECT 1 FROM roles WHERE name = 'analyst')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- OPERATOR: Grant all workbench permissions
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'operator'
    AND p.resource = 'workbench'
    AND EXISTS (SELECT 1 FROM roles WHERE name = 'operator')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- VIEWER: Grant only view_history permission
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r, permissions p
  WHERE r.name = 'viewer'
    AND p.name = 'workbench.view_history'
    AND EXISTS (SELECT 1 FROM roles WHERE name = 'viewer')
ON CONFLICT (role_id, permission_id) DO NOTHING;
