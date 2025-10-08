-- Migration rollback: Remove workbench permissions
-- Description: Removes workbench permissions added in 000013

-- Remove role-permission associations for workbench
DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id FROM permissions WHERE resource = 'workbench'
);

-- Remove workbench permissions
DELETE FROM permissions WHERE resource = 'workbench';
