package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// QuickTemplate represents a saved workbench configuration template
type QuickTemplate struct {
	ID                int             `json:"id"`
	UserID            int             `json:"user_id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	ConfigurationData json.RawMessage `json:"configuration_data"` // {llm_provider_id, prompt_version_id, proposed_changes[]}
	IsShared          bool            `json:"is_shared"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         *time.Time      `json:"updated_at,omitempty"`
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	UserID            int
	Name              string
	Description       string
	ConfigurationData json.RawMessage
	IsShared          bool
}

// UpdateTemplateInput represents input for updating a template
type UpdateTemplateInput struct {
	Name              string
	Description       string
	ConfigurationData json.RawMessage
	IsShared          bool
}

// Store provides database operations for quick templates
type Store struct {
	db *sql.DB
}

// NewStore creates a new template store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all templates for a user (personal + shared)
func (s *Store) List(ctx context.Context, userID int) ([]QuickTemplate, error) {
	query := `
		SELECT id, user_id, name, description, configuration_data, is_shared, created_at, updated_at
		FROM quick_templates
		WHERE user_id = $1 OR is_shared = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []QuickTemplate
	for rows.Next() {
		var t QuickTemplate
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.ConfigurationData, &t.IsShared, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// Get retrieves a template by ID
func (s *Store) Get(ctx context.Context, id int, userID int) (*QuickTemplate, error) {
	query := `
		SELECT id, user_id, name, description, configuration_data, is_shared, created_at, updated_at
		FROM quick_templates
		WHERE id = $1 AND (user_id = $2 OR is_shared = TRUE)
	`

	var t QuickTemplate
	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Description, &t.ConfigurationData, &t.IsShared, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &t, nil
}

// Create creates a new template
func (s *Store) Create(ctx context.Context, input CreateTemplateInput) (*QuickTemplate, error) {
	query := `
		INSERT INTO quick_templates (user_id, name, description, configuration_data, is_shared)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, description, configuration_data, is_shared, created_at, updated_at
	`

	var t QuickTemplate
	err := s.db.QueryRowContext(ctx, query, input.UserID, input.Name, input.Description, input.ConfigurationData, input.IsShared).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Description, &t.ConfigurationData, &t.IsShared, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return &t, nil
}

// Update updates a template
func (s *Store) Update(ctx context.Context, id int, userID int, input UpdateTemplateInput) (*QuickTemplate, error) {
	query := `
		UPDATE quick_templates
		SET name = $1, description = $2, configuration_data = $3, is_shared = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, name, description, configuration_data, is_shared, created_at, updated_at
	`

	var t QuickTemplate
	err := s.db.QueryRowContext(ctx, query, input.Name, input.Description, input.ConfigurationData, input.IsShared, id, userID).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Description, &t.ConfigurationData, &t.IsShared, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found or not owned by user")
	}
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return &t, nil
}

// Delete deletes a template
func (s *Store) Delete(ctx context.Context, id int, userID int) error {
	query := `DELETE FROM quick_templates WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("template not found or not owned by user")
	}
	return nil
}
