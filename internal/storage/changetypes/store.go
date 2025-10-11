package changetypes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ChangeType represents a configuration change type
type ChangeType struct {
	ID                 int        `json:"id"`
	Code               string     `json:"code"`            // e.g., "jvm", "container", "platform"
	DisplayName        string     `json:"display_name"`    // e.g., "JVM Configuration"
	Description        string     `json:"description"`     // Detailed description
	CategoryID         *int       `json:"category_id"`     // Foreign key to change_type_categories
	Category           string     `json:"category"`        // Category name (joined from categories table)
	CategoryInfo       *CategoryInfo `json:"category_info,omitempty"` // Full category details
	MetricCategory     string     `json:"metric_category"` // Canonical metric category filter
	MetricSubcategory  string     `json:"metric_subcategory"` // Canonical metric subcategory filter
	MetricNamePattern  string     `json:"metric_name_pattern"` // Optional metric name regex
	AllowedFields      []string   `json:"allowed_fields"`  // Array of allowed canonical metric names
	IsActive           bool       `json:"is_active"`       // Whether this type is available
	DisplayOrder       int        `json:"display_order"`   // Sort order for UI
	Icon               string     `json:"icon"`            // Optional icon identifier
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

// CategoryInfo is embedded category information
type CategoryInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
}

// Store provides database operations for configuration change types
type Store struct {
	db *sql.DB
}

// NewStore creates a new change type store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all active change types ordered by display_order (with category info)
func (s *Store) List(ctx context.Context) ([]ChangeType, error) {
	query := `
		SELECT
			ct.id, ct.code, ct.display_name, ct.description, ct.category_id,
			ct.metric_category, ct.metric_subcategory, ct.metric_name_pattern, ct.allowed_fields,
			ct.is_active, ct.display_order, ct.icon, ct.created_at, ct.updated_at,
			c.id, c.name, c.display_name, c.color, c.icon
		FROM configuration_change_types ct
		LEFT JOIN change_type_categories c ON ct.category_id = c.id
		WHERE ct.is_active = TRUE
		ORDER BY ct.display_order ASC, ct.display_name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list change types: %w", err)
	}
	defer rows.Close()

	var types []ChangeType
	for rows.Next() {
		var t ChangeType
		var description, icon, metricCat, metricSubcat, metricPattern sql.NullString
		var allowedFieldsJSON sql.NullString
		var catInfoID sql.NullInt32
		var catName, catDisplayName, catColor, catIcon sql.NullString

		if err := rows.Scan(
			&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
			&metricCat, &metricSubcat, &metricPattern, &allowedFieldsJSON,
			&t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
			&catInfoID, &catName, &catDisplayName, &catColor, &catIcon,
		); err != nil {
			return nil, fmt.Errorf("scan change type: %w", err)
		}

		t.Description = description.String
		t.Icon = icon.String
		t.MetricCategory = metricCat.String
		t.MetricSubcategory = metricSubcat.String
		t.MetricNamePattern = metricPattern.String

		// Parse allowed fields JSON array
		if allowedFieldsJSON.Valid && allowedFieldsJSON.String != "" {
			if err := json.Unmarshal([]byte(allowedFieldsJSON.String), &t.AllowedFields); err != nil {
				t.AllowedFields = []string{}
			}
		} else {
			t.AllowedFields = []string{}
		}

		// Populate category info if exists
		if catInfoID.Valid {
			t.Category = catName.String
			t.CategoryInfo = &CategoryInfo{
				ID:          int(catInfoID.Int32),
				Name:        catName.String,
				DisplayName: catDisplayName.String,
				Color:       catColor.String,
				Icon:        catIcon.String,
			}
		}

		types = append(types, t)
	}

	return types, rows.Err()
}

// ListAll returns all change types (including inactive) with category info
func (s *Store) ListAll(ctx context.Context) ([]ChangeType, error) {
	query := `
		SELECT
			ct.id, ct.code, ct.display_name, ct.description, ct.category_id,
			ct.metric_category, ct.metric_subcategory, ct.metric_name_pattern, ct.allowed_fields,
			ct.is_active, ct.display_order, ct.icon, ct.created_at, ct.updated_at,
			c.id, c.name, c.display_name, c.color, c.icon
		FROM configuration_change_types ct
		LEFT JOIN change_type_categories c ON ct.category_id = c.id
		ORDER BY ct.display_order ASC, ct.display_name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all change types: %w", err)
	}
	defer rows.Close()

	var types []ChangeType
	for rows.Next() {
		var t ChangeType
		var description, icon, metricCat, metricSubcat, metricPattern sql.NullString
		var allowedFieldsJSON sql.NullString
		var catInfoID sql.NullInt32
		var catName, catDisplayName, catColor, catIcon sql.NullString

		if err := rows.Scan(
			&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
			&metricCat, &metricSubcat, &metricPattern, &allowedFieldsJSON,
			&t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
			&catInfoID, &catName, &catDisplayName, &catColor, &catIcon,
		); err != nil {
			return nil, fmt.Errorf("scan change type: %w", err)
		}

		t.Description = description.String
		t.Icon = icon.String
		t.MetricCategory = metricCat.String
		t.MetricSubcategory = metricSubcat.String
		t.MetricNamePattern = metricPattern.String

		// Parse allowed fields JSON array
		if allowedFieldsJSON.Valid && allowedFieldsJSON.String != "" {
			if err := json.Unmarshal([]byte(allowedFieldsJSON.String), &t.AllowedFields); err != nil {
				t.AllowedFields = []string{}
			}
		} else {
			t.AllowedFields = []string{}
		}

		// Populate category info if exists
		if catInfoID.Valid {
			t.Category = catName.String
			t.CategoryInfo = &CategoryInfo{
				ID:          int(catInfoID.Int32),
				Name:        catName.String,
				DisplayName: catDisplayName.String,
				Color:       catColor.String,
				Icon:        catIcon.String,
			}
		}

		types = append(types, t)
	}

	return types, rows.Err()
}

// GetByCode retrieves a change type by its code (with category info)
func (s *Store) GetByCode(ctx context.Context, code string) (*ChangeType, error) {
	query := `
		SELECT
			ct.id, ct.code, ct.display_name, ct.description, ct.category_id,
			ct.metric_category, ct.metric_subcategory, ct.metric_name_pattern, ct.allowed_fields,
			ct.is_active, ct.display_order, ct.icon, ct.created_at, ct.updated_at,
			c.id, c.name, c.display_name, c.color, c.icon
		FROM configuration_change_types ct
		LEFT JOIN change_type_categories c ON ct.category_id = c.id
		WHERE ct.code = $1
	`

	var t ChangeType
	var description, icon, metricCat, metricSubcat, metricPattern sql.NullString
	var allowedFieldsJSON sql.NullString
	var catInfoID sql.NullInt32
	var catName, catDisplayName, catColor, catIcon sql.NullString

	err := s.db.QueryRowContext(ctx, query, code).Scan(
		&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
		&metricCat, &metricSubcat, &metricPattern, &allowedFieldsJSON,
		&t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
		&catInfoID, &catName, &catDisplayName, &catColor, &catIcon,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("change type with code '%s' not found", code)
	}
	if err != nil {
		return nil, fmt.Errorf("get change type: %w", err)
	}

	t.Description = description.String
	t.Icon = icon.String
	t.MetricCategory = metricCat.String
	t.MetricSubcategory = metricSubcat.String
	t.MetricNamePattern = metricPattern.String

	// Parse allowed fields JSON array
	if allowedFieldsJSON.Valid && allowedFieldsJSON.String != "" {
		if err := json.Unmarshal([]byte(allowedFieldsJSON.String), &t.AllowedFields); err != nil {
			t.AllowedFields = []string{}
		}
	} else {
		t.AllowedFields = []string{}
	}

	// Populate category info if exists
	if catInfoID.Valid {
		t.Category = catName.String
		t.CategoryInfo = &CategoryInfo{
			ID:          int(catInfoID.Int32),
			Name:        catName.String,
			DisplayName: catDisplayName.String,
			Color:       catColor.String,
			Icon:        catIcon.String,
		}
	}

	return &t, nil
}

// CreateInput represents input for creating a change type
type CreateInput struct {
	Code          string
	DisplayName   string
	Description   string
	CategoryID    *int  // Foreign key to change_type_categories
	AllowedFields []string
	IsActive      bool
	DisplayOrder  int
	Icon          string
}

// UpdateInput represents input for updating a change type
type UpdateInput struct {
	DisplayName   string
	Description   string
	CategoryID    *int  // Foreign key to change_type_categories
	AllowedFields []string
	IsActive      bool
	DisplayOrder  int
	Icon          string
}

// Create creates a new change type
func (s *Store) Create(ctx context.Context, input CreateInput) (*ChangeType, error) {
	// Marshal allowed fields to JSON
	var allowedFieldsJSON []byte
	var err error
	if len(input.AllowedFields) > 0 {
		allowedFieldsJSON, err = json.Marshal(input.AllowedFields)
		if err != nil {
			return nil, fmt.Errorf("marshal allowed fields: %w", err)
		}
	}

	query := `
		INSERT INTO configuration_change_types (code, display_name, description, category_id, allowed_fields, is_active, display_order, icon)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, code, display_name, description, category_id, allowed_fields, is_active, display_order, icon, created_at, updated_at
	`

	var t ChangeType
	var description, icon sql.NullString
	var allowedFieldsJSONResult sql.NullString

	err = s.db.QueryRowContext(ctx, query,
		input.Code, input.DisplayName, input.Description, input.CategoryID,
		allowedFieldsJSON, input.IsActive, input.DisplayOrder, input.Icon,
	).Scan(
		&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
		&allowedFieldsJSONResult, &t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create change type: %w", err)
	}

	t.Description = description.String
	t.Icon = icon.String

	// Parse allowed fields JSON array
	if allowedFieldsJSONResult.Valid && allowedFieldsJSONResult.String != "" {
		if err := json.Unmarshal([]byte(allowedFieldsJSONResult.String), &t.AllowedFields); err != nil {
			t.AllowedFields = []string{}
		}
	} else {
		t.AllowedFields = []string{}
	}

	// Fetch category info if category_id is set
	if t.CategoryID != nil {
		catQuery := `SELECT name, display_name, color, icon FROM change_type_categories WHERE id = $1`
		var catName, catDisplayName, catColor, catIcon sql.NullString
		s.db.QueryRowContext(ctx, catQuery, *t.CategoryID).Scan(&catName, &catDisplayName, &catColor, &catIcon)
		if catName.Valid {
			t.Category = catName.String
			t.CategoryInfo = &CategoryInfo{
				ID:          *t.CategoryID,
				Name:        catName.String,
				DisplayName: catDisplayName.String,
				Color:       catColor.String,
				Icon:        catIcon.String,
			}
		}
	}

	return &t, nil
}

// Update updates an existing change type
func (s *Store) Update(ctx context.Context, id int, input UpdateInput) (*ChangeType, error) {
	// Marshal allowed fields to JSON
	var allowedFieldsJSON []byte
	var err error
	if len(input.AllowedFields) > 0 {
		allowedFieldsJSON, err = json.Marshal(input.AllowedFields)
		if err != nil {
			return nil, fmt.Errorf("marshal allowed fields: %w", err)
		}
	}

	query := `
		UPDATE configuration_change_types
		SET display_name = $1, description = $2, category_id = $3, allowed_fields = $4, is_active = $5, display_order = $6, icon = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING id, code, display_name, description, category_id, allowed_fields, is_active, display_order, icon, created_at, updated_at
	`

	var t ChangeType
	var description, icon sql.NullString
	var allowedFieldsJSONResult sql.NullString

	err = s.db.QueryRowContext(ctx, query,
		input.DisplayName, input.Description, input.CategoryID,
		allowedFieldsJSON, input.IsActive, input.DisplayOrder, input.Icon, id,
	).Scan(
		&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
		&allowedFieldsJSONResult, &t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("change type not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update change type: %w", err)
	}

	t.Description = description.String
	t.Icon = icon.String

	// Parse allowed fields JSON array
	if allowedFieldsJSONResult.Valid && allowedFieldsJSONResult.String != "" {
		if err := json.Unmarshal([]byte(allowedFieldsJSONResult.String), &t.AllowedFields); err != nil {
			t.AllowedFields = []string{}
		}
	} else {
		t.AllowedFields = []string{}
	}

	// Fetch category info if category_id is set
	if t.CategoryID != nil {
		catQuery := `SELECT name, display_name, color, icon FROM change_type_categories WHERE id = $1`
		var catName, catDisplayName, catColor, catIcon sql.NullString
		s.db.QueryRowContext(ctx, catQuery, *t.CategoryID).Scan(&catName, &catDisplayName, &catColor, &catIcon)
		if catName.Valid {
			t.Category = catName.String
			t.CategoryInfo = &CategoryInfo{
				ID:          *t.CategoryID,
				Name:        catName.String,
				DisplayName: catDisplayName.String,
				Color:       catColor.String,
				Icon:        catIcon.String,
			}
		}
	}

	return &t, nil
}

// Delete deletes a change type
func (s *Store) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM configuration_change_types WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete change type: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("change type not found")
	}
	return nil
}

// GetByID retrieves a change type by its ID (with category info)
func (s *Store) GetByID(ctx context.Context, id int) (*ChangeType, error) {
	query := `
		SELECT
			ct.id, ct.code, ct.display_name, ct.description, ct.category_id,
			ct.metric_category, ct.metric_subcategory, ct.metric_name_pattern, ct.allowed_fields,
			ct.is_active, ct.display_order, ct.icon, ct.created_at, ct.updated_at,
			c.id, c.name, c.display_name, c.color, c.icon
		FROM configuration_change_types ct
		LEFT JOIN change_type_categories c ON ct.category_id = c.id
		WHERE ct.id = $1
	`

	var t ChangeType
	var description, icon, metricCat, metricSubcat, metricPattern sql.NullString
	var allowedFieldsJSON sql.NullString
	var catInfoID sql.NullInt32
	var catName, catDisplayName, catColor, catIcon sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Code, &t.DisplayName, &description, &t.CategoryID,
		&metricCat, &metricSubcat, &metricPattern, &allowedFieldsJSON,
		&t.IsActive, &t.DisplayOrder, &icon, &t.CreatedAt, &t.UpdatedAt,
		&catInfoID, &catName, &catDisplayName, &catColor, &catIcon,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("change type with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get change type: %w", err)
	}

	t.Description = description.String
	t.Icon = icon.String
	t.MetricCategory = metricCat.String
	t.MetricSubcategory = metricSubcat.String
	t.MetricNamePattern = metricPattern.String

	// Parse allowed fields JSON array
	if allowedFieldsJSON.Valid && allowedFieldsJSON.String != "" {
		if err := json.Unmarshal([]byte(allowedFieldsJSON.String), &t.AllowedFields); err != nil {
			t.AllowedFields = []string{}
		}
	} else {
		t.AllowedFields = []string{}
	}

	// Populate category info if exists
	if catInfoID.Valid {
		t.Category = catName.String
		t.CategoryInfo = &CategoryInfo{
			ID:          int(catInfoID.Int32),
			Name:        catName.String,
			DisplayName: catDisplayName.String,
			Color:       catColor.String,
			Icon:        catIcon.String,
		}
	}

	return &t, nil
}

// GetCategories returns distinct categories from all change types
func (s *Store) GetCategories(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM configuration_change_types
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}
