-- Rollback migration for RBAC tables

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_permission_audit_timestamp;
DROP INDEX IF EXISTS idx_permission_audit_user;
DROP INDEX IF EXISTS idx_user_roles_role;
DROP INDEX IF EXISTS idx_user_roles_user;
DROP INDEX IF EXISTS idx_role_permissions_permission;
DROP INDEX IF EXISTS idx_role_permissions_role;

DROP TABLE IF EXISTS permission_audit;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
