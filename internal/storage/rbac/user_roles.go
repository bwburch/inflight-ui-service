package rbac

import (
	"context"
	"database/sql"
	"time"
)

// UserRole represents a user's role assignment
type UserRole struct {
	ID         int        `db:"id" json:"id"`
	UserID     int        `db:"user_id" json:"user_id"`
	RoleID     int        `db:"role_id" json:"role_id"`
	RoleName   string     `db:"role_name" json:"role_name"`
	AssignedAt time.Time  `db:"assigned_at" json:"assigned_at"`
	AssignedBy *int       `db:"assigned_by" json:"assigned_by,omitempty"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at,omitempty"`
}

// UserPermissions aggregates a user's roles and permissions
type UserPermissions struct {
	UserID      int      `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// UserRoleStore handles database operations for user-role mappings
type UserRoleStore struct {
	db *sql.DB
}

// NewUserRoleStore creates a new user role store
func NewUserRoleStore(db *sql.DB) *UserRoleStore {
	return &UserRoleStore{db: db}
}

// GetUserRoles retrieves all roles for a user
func (s *UserRoleStore) GetUserRoles(ctx context.Context, userID int) ([]UserRole, error) {
	query := `
		SELECT ur.id, ur.user_id, ur.role_id, r.name as role_name,
		       ur.assigned_at, ur.assigned_by, ur.expires_at
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		ORDER BY ur.assigned_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userRoles []UserRole
	for rows.Next() {
		var ur UserRole
		if err := rows.Scan(&ur.ID, &ur.UserID, &ur.RoleID, &ur.RoleName, &ur.AssignedAt, &ur.AssignedBy, &ur.ExpiresAt); err != nil {
			return nil, err
		}
		userRoles = append(userRoles, ur)
	}

	return userRoles, nil
}

// GetUserPermissions retrieves all effective permissions for a user (aggregated from all roles)
func (s *UserRoleStore) GetUserPermissions(ctx context.Context, userID int) (*UserPermissions, error) {
	// Get user's roles
	rolesQuery := `
		SELECT DISTINCT r.name
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		ORDER BY r.name
	`

	rows, err := s.db.QueryContext(ctx, rolesQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, err
		}
		roles = append(roles, roleName)
	}

	// Get user's permissions (aggregated from all roles)
	permsQuery := `
		SELECT DISTINCT p.name
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		ORDER BY p.name
	`

	permRows, err := s.db.QueryContext(ctx, permsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer permRows.Close()

	var permissions []string
	for permRows.Next() {
		var permName string
		if err := permRows.Scan(&permName); err != nil {
			return nil, err
		}
		permissions = append(permissions, permName)
	}

	return &UserPermissions{
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

// AssignRole assigns a role to a user
func (s *UserRoleStore) AssignRole(ctx context.Context, userID, roleID, assignedBy int, expiresAt *time.Time) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, role_id) DO UPDATE
		SET expires_at = EXCLUDED.expires_at
	`

	_, err := s.db.ExecContext(ctx, query, userID, roleID, assignedBy, expiresAt)
	return err
}

// RemoveRole removes a role from a user
func (s *UserRoleStore) RemoveRole(ctx context.Context, userID, roleID int) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`

	_, err := s.db.ExecContext(ctx, query, userID, roleID)
	return err
}

// CheckPermission checks if a user has a specific permission
func (s *UserRoleStore) CheckPermission(ctx context.Context, userID int, permission string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN user_roles ur ON rp.role_id = ur.role_id
			WHERE ur.user_id = $1
			  AND p.name = $2
			  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		)
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID, permission).Scan(&exists)
	return exists, err
}

// CheckAnyPermission checks if a user has any of the specified permissions
func (s *UserRoleStore) CheckAnyPermission(ctx context.Context, userID int, permissions []string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN user_roles ur ON rp.role_id = ur.role_id
			WHERE ur.user_id = $1
			  AND p.name = ANY($2)
			  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		)
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID, permissions).Scan(&exists)
	return exists, err
}

// IsAdmin checks if a user has the admin role
func (s *UserRoleStore) IsAdmin(ctx context.Context, userID int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM roles r
			JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = $1
			  AND r.name = 'admin'
			  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		)
	`

	var isAdmin bool
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&isAdmin)
	return isAdmin, err
}
