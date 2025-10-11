package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ServiceProfile represents a service profile type
type ServiceProfile struct {
	ID                         int         `json:"id"`
	Name                       string      `json:"name"`                           // Unique identifier
	DisplayName                string      `json:"display_name"`                   // Human-readable name
	Description                string      `json:"description"`
	RequiredMetrics            []string    `json:"required_metrics"`               // Required observable metrics
	RecommendedMetrics         []string    `json:"recommended_metrics"`            // Recommended observable metrics
	AllowedConfigurationFields []string    `json:"allowed_configuration_fields"`   // Configurable metrics that can be modified
	Icon                       string      `json:"icon"`
	Color                      string      `json:"color"`
	DisplayOrder               int         `json:"display_order"`
	IsActive                   bool        `json:"is_active"`
	CreatedAt                  time.Time   `json:"created_at"`
	UpdatedAt                  *time.Time  `json:"updated_at,omitempty"`
}

// Store provides database operations for service profiles
type Store struct {
	db *sql.DB
}

// NewStore creates a new profile store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all active profiles
func (s *Store) List(ctx context.Context) ([]ServiceProfile, error) {
	query := `
		SELECT id, name, display_name, description, required_metrics, recommended_metrics,
		       allowed_configuration_fields, icon, color, display_order, is_active, created_at, updated_at
		FROM service_profiles
		WHERE is_active = TRUE
		ORDER BY display_order ASC, display_name ASC
	`

	return s.scanProfiles(ctx, query)
}

// ListAll returns all profiles (including inactive)
func (s *Store) ListAll(ctx context.Context) ([]ServiceProfile, error) {
	query := `
		SELECT id, name, display_name, description, required_metrics, recommended_metrics,
		       allowed_configuration_fields, icon, color, display_order, is_active, created_at, updated_at
		FROM service_profiles
		ORDER BY display_order ASC, display_name ASC
	`

	return s.scanProfiles(ctx, query)
}

// GetByID retrieves a profile by its ID
func (s *Store) GetByID(ctx context.Context, id int) (*ServiceProfile, error) {
	query := `
		SELECT id, name, display_name, description, required_metrics, recommended_metrics,
		       allowed_configuration_fields, icon, color, display_order, is_active, created_at, updated_at
		FROM service_profiles
		WHERE id = $1
	`

	var p ServiceProfile
	var description, requiredMetrics, recommendedMetrics, allowedFields, icon, color sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.DisplayName, &description, &requiredMetrics, &recommendedMetrics,
		&allowedFields, &p.Icon, &p.Color, &p.DisplayOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	p.Description = description.String
	p.Icon = icon.String
	p.Color = color.String

	// Parse JSON arrays
	if requiredMetrics.Valid && requiredMetrics.String != "" {
		json.Unmarshal([]byte(requiredMetrics.String), &p.RequiredMetrics)
	}
	if recommendedMetrics.Valid && recommendedMetrics.String != "" {
		json.Unmarshal([]byte(recommendedMetrics.String), &p.RecommendedMetrics)
	}
	if allowedFields.Valid && allowedFields.String != "" {
		json.Unmarshal([]byte(allowedFields.String), &p.AllowedConfigurationFields)
	}

	return &p, nil
}

// CreateInput represents input for creating a profile
type CreateInput struct {
	Name                       string
	DisplayName                string
	Description                string
	RequiredMetrics            []string
	RecommendedMetrics         []string
	AllowedConfigurationFields []string
	Icon                       string
	Color                      string
	DisplayOrder               int
	IsActive                   bool
}

// UpdateInput represents input for updating a profile
type UpdateInput struct {
	DisplayName                string
	Description                string
	RequiredMetrics            []string
	RecommendedMetrics         []string
	AllowedConfigurationFields []string
	Icon                       string
	Color                      string
	DisplayOrder               int
	IsActive                   bool
}

// Create creates a new profile
func (s *Store) Create(ctx context.Context, input CreateInput) (*ServiceProfile, error) {
	requiredJSON, _ := json.Marshal(input.RequiredMetrics)
	recommendedJSON, _ := json.Marshal(input.RecommendedMetrics)
	allowedFieldsJSON, _ := json.Marshal(input.AllowedConfigurationFields)

	query := `
		INSERT INTO service_profiles (name, display_name, description, required_metrics, recommended_metrics, allowed_configuration_fields, icon, color, display_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, display_name, description, required_metrics, recommended_metrics, allowed_configuration_fields, icon, color, display_order, is_active, created_at, updated_at
	`

	var p ServiceProfile
	var description, requiredMetrics, recommendedMetrics, allowedFields, icon, color sql.NullString

	err := s.db.QueryRowContext(ctx, query,
		input.Name, input.DisplayName, input.Description, requiredJSON, recommendedJSON, allowedFieldsJSON,
		input.Icon, input.Color, input.DisplayOrder, input.IsActive,
	).Scan(
		&p.ID, &p.Name, &p.DisplayName, &description, &requiredMetrics, &recommendedMetrics, &allowedFields,
		&p.Icon, &p.Color, &p.DisplayOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	p.Description = description.String
	p.Icon = icon.String
	p.Color = color.String

	if requiredMetrics.Valid {
		json.Unmarshal([]byte(requiredMetrics.String), &p.RequiredMetrics)
	}
	if recommendedMetrics.Valid {
		json.Unmarshal([]byte(recommendedMetrics.String), &p.RecommendedMetrics)
	}
	if allowedFields.Valid {
		json.Unmarshal([]byte(allowedFields.String), &p.AllowedConfigurationFields)
	}

	return &p, nil
}

// Update updates an existing profile
func (s *Store) Update(ctx context.Context, id int, input UpdateInput) (*ServiceProfile, error) {
	requiredJSON, _ := json.Marshal(input.RequiredMetrics)
	recommendedJSON, _ := json.Marshal(input.RecommendedMetrics)
	allowedFieldsJSON, _ := json.Marshal(input.AllowedConfigurationFields)

	query := `
		UPDATE service_profiles
		SET display_name = $1, description = $2, required_metrics = $3, recommended_metrics = $4,
		    allowed_configuration_fields = $5, icon = $6, color = $7, display_order = $8, is_active = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING id, name, display_name, description, required_metrics, recommended_metrics, allowed_configuration_fields, icon, color, display_order, is_active, created_at, updated_at
	`

	var p ServiceProfile
	var description, requiredMetrics, recommendedMetrics, allowedFields, icon, color sql.NullString

	err := s.db.QueryRowContext(ctx, query,
		input.DisplayName, input.Description, requiredJSON, recommendedJSON, allowedFieldsJSON,
		input.Icon, input.Color, input.DisplayOrder, input.IsActive, id,
	).Scan(
		&p.ID, &p.Name, &p.DisplayName, &description, &requiredMetrics, &recommendedMetrics, &allowedFields,
		&p.Icon, &p.Color, &p.DisplayOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	p.Description = description.String
	p.Icon = icon.String
	p.Color = color.String

	if requiredMetrics.Valid {
		json.Unmarshal([]byte(requiredMetrics.String), &p.RequiredMetrics)
	}
	if recommendedMetrics.Valid {
		json.Unmarshal([]byte(recommendedMetrics.String), &p.RecommendedMetrics)
	}
	if allowedFields.Valid {
		json.Unmarshal([]byte(allowedFields.String), &p.AllowedConfigurationFields)
	}

	return &p, nil
}

// Delete deletes a profile
func (s *Store) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM service_profiles WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("profile not found")
	}
	return nil
}

// Helper function to scan profiles
func (s *Store) scanProfiles(ctx context.Context, query string, args ...interface{}) ([]ServiceProfile, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanProfileRows(rows)
}

func (s *Store) scanProfileRows(rows *sql.Rows) ([]ServiceProfile, error) {
	var profiles []ServiceProfile

	for rows.Next() {
		var p ServiceProfile
		var description, requiredMetrics, recommendedMetrics, allowedFields, icon, color sql.NullString

		if err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &description, &requiredMetrics, &recommendedMetrics,
			&allowedFields, &p.Icon, &p.Color, &p.DisplayOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan profile: %w", err)
		}

		p.Description = description.String
		p.Icon = icon.String
		p.Color = color.String

		// Parse JSON arrays
		if requiredMetrics.Valid && requiredMetrics.String != "" {
			json.Unmarshal([]byte(requiredMetrics.String), &p.RequiredMetrics)
		} else {
			p.RequiredMetrics = []string{}
		}

		if recommendedMetrics.Valid && recommendedMetrics.String != "" {
			json.Unmarshal([]byte(recommendedMetrics.String), &p.RecommendedMetrics)
		} else {
			p.RecommendedMetrics = []string{}
		}

		if allowedFields.Valid && allowedFields.String != "" {
			json.Unmarshal([]byte(allowedFields.String), &p.AllowedConfigurationFields)
		} else {
			p.AllowedConfigurationFields = []string{}
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}
