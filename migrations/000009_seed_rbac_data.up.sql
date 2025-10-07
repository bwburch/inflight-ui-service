-- Seed data for RBAC system
-- Inserts default roles and permissions

-- ============================================================================
-- Insert Default Roles
-- ============================================================================

INSERT INTO roles (name, description, is_system) VALUES
  ('admin', 'Full system access - all permissions', true),
  ('operator', 'Operational monitoring and configuration management', true),
  ('viewer', 'Read-only access to all data', true),
  ('analyst', 'Data analysis, model calibration, and query execution', true)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Navigation Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('nav.overview', 'navigation', 'view', 'navigation', 'Access Overview dashboard'),
  ('nav.services', 'navigation', 'view', 'navigation', 'Access Services page'),
  ('nav.models', 'navigation', 'view', 'navigation', 'Access Calibrated Models page'),
  ('nav.alerts', 'navigation', 'view', 'navigation', 'Access Alerts page'),
  ('nav.metrics', 'navigation', 'view', 'navigation', 'Access Metrics Configuration page'),
  ('nav.query', 'navigation', 'view', 'navigation', 'Access Query Explorer page'),
  ('nav.workbench', 'navigation', 'view', 'navigation', 'Access Scientific Workbench page'),
  ('nav.reports', 'navigation', 'view', 'navigation', 'Access Reports page'),
  ('nav.settings', 'navigation', 'view', 'navigation', 'Access Settings page')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Service Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('services.view', 'services', 'view', 'data', 'View service list and details'),
  ('services.create', 'services', 'create', 'data', 'Register new services'),
  ('services.edit', 'services', 'edit', 'data', 'Update service configuration'),
  ('services.delete', 'services', 'delete', 'data', 'Delete services')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Model Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('models.view', 'models', 'view', 'data', 'View calibrated models'),
  ('models.calibrate', 'models', 'calibrate', 'data', 'Trigger model calibration'),
  ('models.edit', 'models', 'edit', 'data', 'Update model metadata'),
  ('models.delete', 'models', 'delete', 'data', 'Delete model versions')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Metrics & Configuration Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('metrics.view', 'metrics', 'view', 'data', 'View metrics data'),
  ('canonical_metrics.view', 'canonical_metrics', 'view', 'configuration', 'View canonical metrics registry'),
  ('canonical_metrics.create', 'canonical_metrics', 'create', 'configuration', 'Create new canonical metrics'),
  ('canonical_metrics.edit', 'canonical_metrics', 'edit', 'configuration', 'Edit canonical metric definitions'),
  ('canonical_metrics.delete', 'canonical_metrics', 'delete', 'configuration', 'Delete canonical metrics'),
  ('apm_mappings.view', 'apm_mappings', 'view', 'configuration', 'View APM provider mappings'),
  ('apm_mappings.create', 'apm_mappings', 'create', 'configuration', 'Create APM metric mappings'),
  ('apm_mappings.edit', 'apm_mappings', 'edit', 'configuration', 'Edit APM metric mappings'),
  ('apm_mappings.delete', 'apm_mappings', 'delete', 'configuration', 'Delete APM metric mappings'),
  ('apm_providers.view', 'apm_providers', 'view', 'configuration', 'View APM provider configurations'),
  ('apm_providers.configure', 'apm_providers', 'configure', 'configuration', 'Configure APM provider settings'),
  ('apm_providers.create', 'apm_providers', 'create', 'configuration', 'Create new APM providers'),
  ('apm_providers.toggle', 'apm_providers', 'toggle', 'configuration', 'Enable/disable APM providers')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Collection Job Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('collection_jobs.view', 'collection_jobs', 'view', 'configuration', 'View collection schedules'),
  ('collection_jobs.create', 'collection_jobs', 'create', 'configuration', 'Create collection schedules'),
  ('collection_jobs.edit', 'collection_jobs', 'edit', 'configuration', 'Edit collection schedules'),
  ('collection_jobs.delete', 'collection_jobs', 'delete', 'configuration', 'Delete collection schedules'),
  ('collection_jobs.trigger', 'collection_jobs', 'trigger', 'configuration', 'Manually trigger collection jobs')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Query Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('queries.execute', 'queries', 'execute', 'data', 'Execute queries against metrics'),
  ('queries.save', 'queries', 'save', 'data', 'Save queries for reuse'),
  ('queries.delete', 'queries', 'delete', 'data', 'Delete saved queries')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert User & Role Management Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, category, description) VALUES
  ('users.view', 'users', 'view', 'admin', 'View user accounts'),
  ('users.create', 'users', 'create', 'admin', 'Create new user accounts'),
  ('users.edit', 'users', 'edit', 'admin', 'Edit user account details'),
  ('users.delete', 'users', 'delete', 'admin', 'Delete user accounts'),
  ('users.manage_roles', 'users', 'manage_roles', 'admin', 'Assign and revoke user roles'),
  ('roles.view', 'roles', 'view', 'admin', 'View roles and permissions'),
  ('roles.create', 'roles', 'create', 'admin', 'Create custom roles'),
  ('roles.edit', 'roles', 'edit', 'admin', 'Edit role permissions'),
  ('roles.delete', 'roles', 'delete', 'admin', 'Delete custom roles')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Assign Permissions to Default Roles
-- ============================================================================

-- ADMIN: Grant ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  CROSS JOIN permissions p
  WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- VIEWER: Grant all VIEW permissions only
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r, permissions p
  WHERE r.name = 'viewer'
    AND p.action = 'view'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- OPERATOR: View + Create/Edit/Configure/Toggle/Trigger (no Delete, no Admin)
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r, permissions p
  WHERE r.name = 'operator'
    AND (
      p.action IN ('view', 'create', 'edit', 'configure', 'toggle', 'trigger', 'execute', 'save')
      AND p.category != 'admin'  -- No admin permissions
    )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ANALYST: View all + Model calibration + Query execution + Save queries
INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r, permissions p
  WHERE r.name = 'analyst'
    AND (
      p.action = 'view' OR
      p.name IN ('models.calibrate', 'queries.execute', 'queries.save', 'queries.delete')
    )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- Assign Admin Role to Existing Admin User
-- ============================================================================

-- Assign admin role to user with username 'admin' if exists
INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u, roles r
  WHERE u.username = 'admin' AND r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;
