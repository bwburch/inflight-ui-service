package rbac

import (
	"context"
	"database/sql"
	"time"
)

// Role represents a user role
type Role struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsSystem    bool      `db:"is_system" json:"is_system"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// RoleWithPermissions includes the role's permissions
type RoleWithPermissions struct {
	Role
	Permissions      []Permission `json:"permissions"`
	PermissionCount  int          `json:"permission_count"`
	UserCount        int          `json:"user_count"`
}

// RoleStore handles database operations for roles
type RoleStore struct {
	db *sql.DB
}

// NewRoleStore creates a new role store
func NewRoleStore(db *sql.DB) *RoleStore {
	return &RoleStore{db: db}
}

// List retrieves all roles
func (s *RoleStore) List(ctx context.Context) ([]Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetByID retrieves a role by ID
func (s *RoleStore) GetByID(ctx context.Context, id int) (*Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE id = $1`

	var role Role
	err := s.db.QueryRowContext(ctx, query, id).Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// GetByName retrieves a role by name
func (s *RoleStore) GetByName(ctx context.Context, name string) (*Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE name = $1`

	var role Role
	err := s.db.QueryRowContext(ctx, query, name).Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// Create creates a new role
func (s *RoleStore) Create(ctx context.Context, name, description string) (*Role, error) {
	query := `
		INSERT INTO roles (name, description, is_system)
		VALUES ($1, $2, false)
		RETURNING id, name, description, is_system, created_at, updated_at
	`

	var role Role
	err := s.db.QueryRowContext(ctx, query, name, description).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// Update updates a role's name and description
func (s *RoleStore) Update(ctx context.Context, id int, name, description string) error {
	query := `
		UPDATE roles
		SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1 AND is_system = false
	`

	result, err := s.db.ExecContext(ctx, query, id, name, description)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete deletes a custom role (system roles cannot be deleted)
func (s *RoleStore) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM roles WHERE id = $1 AND is_system = false`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetPermissions retrieves all permissions for a role
func (s *RoleStore) GetPermissions(ctx context.Context, roleID int) ([]Permission, error) {
	query := `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.category, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.category, p.resource, p.action
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var perm Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.Description, &perm.Category, &perm.CreatedAt); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// GrantPermission grants a permission to a role
func (s *RoleStore) GrantPermission(ctx context.Context, roleID, permissionID, grantedBy int) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, roleID, permissionID, grantedBy)
	return err
}

// RevokePermission revokes a permission from a role
func (s *RoleStore) RevokePermission(ctx context.Context, roleID, permissionID int) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`

	_, err := s.db.ExecContext(ctx, query, roleID, permissionID)
	return err
}

// GetUserCount gets the number of users with this role
func (s *RoleStore) GetUserCount(ctx context.Context, roleID int) (int, error) {
	query := `SELECT COUNT(*) FROM user_roles WHERE role_id = $1`

	var count int
	err := s.db.QueryRowContext(ctx, query, roleID).Scan(&count)
	return count, err
}
