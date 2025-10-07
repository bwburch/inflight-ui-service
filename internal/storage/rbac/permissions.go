package rbac

import (
	"context"
	"database/sql"
	"time"
)

// Permission represents a system permission
type Permission struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Resource    string    `db:"resource" json:"resource"`
	Action      string    `db:"action" json:"action"`
	Description string    `db:"description" json:"description"`
	Category    string    `db:"category" json:"category"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// PermissionStore handles database operations for permissions
type PermissionStore struct {
	db *sql.DB
}

// NewPermissionStore creates a new permission store
func NewPermissionStore(db *sql.DB) *PermissionStore {
	return &PermissionStore{db: db}
}

// List retrieves all permissions
func (s *PermissionStore) List(ctx context.Context) ([]Permission, error) {
	query := `
		SELECT id, name, resource, action, description, category, created_at
		FROM permissions
		ORDER BY category, resource, action
	`

	rows, err := s.db.QueryContext(ctx, query)
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

// GetByID retrieves a permission by ID
func (s *PermissionStore) GetByID(ctx context.Context, id int) (*Permission, error) {
	query := `
		SELECT id, name, resource, action, description, category, created_at
		FROM permissions
		WHERE id = $1
	`

	var perm Permission
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.Description, &perm.Category, &perm.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &perm, nil
}

// GetByName retrieves a permission by name
func (s *PermissionStore) GetByName(ctx context.Context, name string) (*Permission, error) {
	query := `
		SELECT id, name, resource, action, description, category, created_at
		FROM permissions
		WHERE name = $1
	`

	var perm Permission
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&perm.ID, &perm.Name, &perm.Resource, &perm.Action, &perm.Description, &perm.Category, &perm.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &perm, nil
}

// ListByCategory retrieves permissions filtered by category
func (s *PermissionStore) ListByCategory(ctx context.Context, category string) ([]Permission, error) {
	query := `
		SELECT id, name, resource, action, description, category, created_at
		FROM permissions
		WHERE category = $1
		ORDER BY resource, action
	`

	rows, err := s.db.QueryContext(ctx, query, category)
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
