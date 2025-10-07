-- Rollback seed data for RBAC

-- Remove all role-permission mappings
DELETE FROM role_permissions;

-- Remove all user-role assignments
DELETE FROM user_roles;

-- Remove all permissions
DELETE FROM permissions;

-- Remove all roles
DELETE FROM roles;
