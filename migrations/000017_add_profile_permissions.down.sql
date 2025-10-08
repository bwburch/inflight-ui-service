-- Migration: Remove metric profile permissions
-- Description: Rollback for adding profile permissions

-- ============================================================================
-- Remove Role-Permission Assignments
-- ============================================================================

DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id FROM permissions
  WHERE resource = 'profiles'
    OR (resource = 'nav' AND action = 'profiles')
);

-- ============================================================================
-- Remove Permissions
-- ============================================================================

DELETE FROM permissions
WHERE resource = 'profiles'
  OR (resource = 'nav' AND action = 'profiles');
