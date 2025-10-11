package categories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Category represents a change type category
type Category struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`          // Unique lowercase identifier
	DisplayName  string     `json:"display_name"`  // Human-readable name
	Description  string     `json:"description"`
	Color        string     `json:"color"`         // Hex color or Tailwind class
	Icon         string     `json:"icon"`
	DisplayOrder int        `json:"display_order"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

// Store provides database operations for categories
type Store struct {
	db *sql.DB
}

// NewStore creates a new category store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all active categories ordered by display_order
func (s *Store) List(ctx context.Context) ([]Category, error) {
	query := `
		SELECT id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
		FROM change_type_categories
		WHERE is_active = TRUE
		ORDER BY display_order ASC, display_name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var description, color, icon sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
			&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}

		c.Description = description.String
		c.Color = color.String
		c.Icon = icon.String

		categories = append(categories, c)
	}

	return categories, rows.Err()
}

// ListAll returns all categories (including inactive)
func (s *Store) ListAll(ctx context.Context) ([]Category, error) {
	query := `
		SELECT id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
		FROM change_type_categories
		ORDER BY display_order ASC, display_name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var description, color, icon sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
			&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}

		c.Description = description.String
		c.Color = color.String
		c.Icon = icon.String

		categories = append(categories, c)
	}

	return categories, rows.Err()
}

// GetByID retrieves a category by its ID
func (s *Store) GetByID(ctx context.Context, id int) (*Category, error) {
	query := `
		SELECT id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
		FROM change_type_categories
		WHERE id = $1
	`

	var c Category
	var description, color, icon sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
		&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}

	c.Description = description.String
	c.Color = color.String
	c.Icon = icon.String

	return &c, nil
}

// GetByName retrieves a category by its name
func (s *Store) GetByName(ctx context.Context, name string) (*Category, error) {
	query := `
		SELECT id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
		FROM change_type_categories
		WHERE name = $1
	`

	var c Category
	var description, color, icon sql.NullString

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
		&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category with name '%s' not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}

	c.Description = description.String
	c.Color = color.String
	c.Icon = icon.String

	return &c, nil
}

// CreateInput represents input for creating a category
type CreateInput struct {
	Name         string
	DisplayName  string
	Description  string
	Color        string
	Icon         string
	DisplayOrder int
	IsActive     bool
}

// UpdateInput represents input for updating a category
type UpdateInput struct {
	DisplayName  string
	Description  string
	Color        string
	Icon         string
	DisplayOrder int
	IsActive     bool
}

// Create creates a new category
func (s *Store) Create(ctx context.Context, input CreateInput) (*Category, error) {
	query := `
		INSERT INTO change_type_categories (name, display_name, description, color, icon, display_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
	`

	var c Category
	var description, color, icon sql.NullString

	err := s.db.QueryRowContext(ctx, query,
		input.Name, input.DisplayName, input.Description, input.Color,
		input.Icon, input.DisplayOrder, input.IsActive,
	).Scan(
		&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
		&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	c.Description = description.String
	c.Color = color.String
	c.Icon = icon.String

	return &c, nil
}

// Update updates an existing category
func (s *Store) Update(ctx context.Context, id int, input UpdateInput) (*Category, error) {
	query := `
		UPDATE change_type_categories
		SET display_name = $1, description = $2, color = $3, icon = $4, display_order = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, name, display_name, description, color, icon, display_order, is_active, created_at, updated_at
	`

	var c Category
	var description, color, icon sql.NullString

	err := s.db.QueryRowContext(ctx, query,
		input.DisplayName, input.Description, input.Color,
		input.Icon, input.DisplayOrder, input.IsActive, id,
	).Scan(
		&c.ID, &c.Name, &c.DisplayName, &description, &color, &icon,
		&c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}

	c.Description = description.String
	c.Color = color.String
	c.Icon = icon.String

	return &c, nil
}

// Delete deletes a category
func (s *Store) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM change_type_categories WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}
