-- Migration: Add metric profile permissions
-- Description: Adds permissions for managing service metric profiles

-- ============================================================================
-- Add Profile Permissions
-- ============================================================================

INSERT INTO permissions (name, resource, action, description, category) VALUES
  ('nav.profiles', 'nav', 'profiles', 'Access metric profiles navigation tab', 'navigation'),
  ('profiles.view', 'profiles', 'view', 'View metric profiles and coverage', 'data'),
  ('profiles.assign', 'profiles', 'assign', 'Assign profiles to services', 'data'),
  ('profiles.create_template', 'profiles', 'create_template', 'Create new profile templates', 'configuration'),
  ('profiles.edit_template', 'profiles', 'edit_template', 'Edit existing profile templates', 'configuration'),
  ('profiles.delete_template', 'profiles', 'delete_template', 'Delete profile templates', 'configuration')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Grant Permissions to Admin Role
-- ============================================================================

-- Admin gets all profile permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('nav', 'profiles')
  AND (p.action IN ('profiles', 'view', 'assign', 'create_template', 'edit_template', 'delete_template'))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- Grant Permissions to Operator Role
-- ============================================================================

-- Operator can view and assign profiles (but not create/edit/delete templates)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'operator'
  AND (
    (p.resource = 'nav' AND p.action = 'profiles')
    OR (p.resource = 'profiles' AND p.action IN ('view', 'assign'))
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- Grant Permissions to Viewer Role
-- ============================================================================

-- Viewer can only view profiles (no assignment or template management)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'viewer'
  AND (
    (p.resource = 'nav' AND p.action = 'profiles')
    OR (p.resource = 'profiles' AND p.action = 'view')
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
