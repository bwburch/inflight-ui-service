package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ProfileType represents the type of metric profile
type ProfileType string

const (
	ProfileTypeBatch          ProfileType = "batch"
	ProfileTypeHighThroughput ProfileType = "high_throughput"
	ProfileTypeStreaming      ProfileType = "streaming"
	ProfileTypeCustom         ProfileType = "custom"
)

// ServiceMetricProfile represents a service's metric profile configuration
type ServiceMetricProfile struct {
	ID               int         `db:"id" json:"id"`
	ServiceID        string      `db:"service_id" json:"service_id"`
	ProfileType      ProfileType `db:"profile_type" json:"profile_type"`
	RequiredMetrics  []string    `db:"required_metrics" json:"required_metrics"`
	OptionalMetrics  []string    `db:"optional_metrics" json:"optional_metrics"`
	SamplingRate     int         `db:"sampling_rate" json:"sampling_rate"` // seconds
	CreatedBy        *int        `db:"created_by" json:"created_by,omitempty"`
	CreatedAt        time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt        *time.Time  `db:"updated_at" json:"updated_at,omitempty"`
}

// ServiceMetricRequirement represents a granular metric requirement
type ServiceMetricRequirement struct {
	ID                  int        `db:"id" json:"id"`
	ServiceID           string     `db:"service_id" json:"service_id"`
	CanonicalMetricName string     `db:"canonical_metric_name" json:"canonical_metric_name"`
	IsRequired          bool       `db:"is_required" json:"is_required"`
	MinSampleRate       *int       `db:"min_sample_rate" json:"min_sample_rate,omitempty"`
	MaxAgeMinutes       int        `db:"max_age_minutes" json:"max_age_minutes"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `db:"updated_at" json:"updated_at,omitempty"`
}

// MetricProfileTemplate represents a pre-defined profile template
type MetricProfileTemplate struct {
	ID              int         `db:"id" json:"id"`
	Name            string      `db:"name" json:"name"`
	ProfileType     ProfileType `db:"profile_type" json:"profile_type"`
	Description     string      `db:"description" json:"description"`
	RequiredMetrics []string    `db:"required_metrics" json:"required_metrics"`
	OptionalMetrics []string    `db:"optional_metrics" json:"optional_metrics"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
}

// MetricCoverageStatus represents the status of a metric's availability
type MetricCoverageStatus string

const (
	CoverageStatusOK      MetricCoverageStatus = "ok"
	CoverageStatusStale   MetricCoverageStatus = "stale"
	CoverageStatusMissing MetricCoverageStatus = "missing"
)

// MetricCoverage represents the availability status of a required metric
type MetricCoverage struct {
	MetricName    string               `json:"metric_name"`
	IsRequired    bool                 `json:"is_required"`
	HasData       bool                 `json:"has_data"`
	LastCollected *time.Time           `json:"last_collected,omitempty"`
	Status        MetricCoverageStatus `json:"status"`
	MaxAgeMinutes int                  `json:"max_age_minutes"`
}

// UpsertProfileInput represents input for creating or updating a profile
type UpsertProfileInput struct {
	ServiceID       string
	ProfileType     ProfileType
	RequiredMetrics []string
	OptionalMetrics []string
	SamplingRate    int
	UserID          int
}

// MetricProfileStore handles database operations for metric profiles
type MetricProfileStore struct {
	db *sql.DB
}

// NewMetricProfileStore creates a new metric profile store
func NewMetricProfileStore(db *sql.DB) *MetricProfileStore {
	return &MetricProfileStore{db: db}
}

// GetProfile retrieves a service's metric profile
func (s *MetricProfileStore) GetProfile(ctx context.Context, serviceID string) (*ServiceMetricProfile, error) {
	query := `
		SELECT id, service_id, profile_type, required_metrics, optional_metrics,
		       sampling_rate, created_by, created_at, updated_at
		FROM service_metric_profiles
		WHERE service_id = $1
	`

	var profile ServiceMetricProfile
	err := s.db.QueryRowContext(ctx, query, serviceID).Scan(
		&profile.ID, &profile.ServiceID, &profile.ProfileType,
		pq.Array(&profile.RequiredMetrics), pq.Array(&profile.OptionalMetrics),
		&profile.SamplingRate, &profile.CreatedBy, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No profile configured
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	return &profile, nil
}

// UpsertProfile creates or updates a service's metric profile
func (s *MetricProfileStore) UpsertProfile(ctx context.Context, input UpsertProfileInput) (*ServiceMetricProfile, error) {
	query := `
		INSERT INTO service_metric_profiles (
			service_id, profile_type, required_metrics, optional_metrics,
			sampling_rate, created_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (service_id) DO UPDATE SET
			profile_type = EXCLUDED.profile_type,
			required_metrics = EXCLUDED.required_metrics,
			optional_metrics = EXCLUDED.optional_metrics,
			sampling_rate = EXCLUDED.sampling_rate,
			updated_at = NOW()
		RETURNING id, service_id, profile_type, required_metrics, optional_metrics,
		          sampling_rate, created_by, created_at, updated_at
	`

	var profile ServiceMetricProfile
	err := s.db.QueryRowContext(ctx, query,
		input.ServiceID, input.ProfileType,
		pq.Array(input.RequiredMetrics), pq.Array(input.OptionalMetrics),
		input.SamplingRate, input.UserID,
	).Scan(
		&profile.ID, &profile.ServiceID, &profile.ProfileType,
		pq.Array(&profile.RequiredMetrics), pq.Array(&profile.OptionalMetrics),
		&profile.SamplingRate, &profile.CreatedBy, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("upsert profile: %w", err)
	}

	return &profile, nil
}

// DeleteProfile deletes a service's metric profile
func (s *MetricProfileStore) DeleteProfile(ctx context.Context, serviceID string) error {
	query := `DELETE FROM service_metric_profiles WHERE service_id = $1`

	result, err := s.db.ExecContext(ctx, query, serviceID)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// GetRequirement retrieves a specific metric requirement
func (s *MetricProfileStore) GetRequirement(ctx context.Context, serviceID, metricName string) (*ServiceMetricRequirement, error) {
	query := `
		SELECT id, service_id, canonical_metric_name, is_required,
		       min_sample_rate, max_age_minutes, created_at, updated_at
		FROM service_metric_requirements
		WHERE service_id = $1 AND canonical_metric_name = $2
	`

	var req ServiceMetricRequirement
	err := s.db.QueryRowContext(ctx, query, serviceID, metricName).Scan(
		&req.ID, &req.ServiceID, &req.CanonicalMetricName,
		&req.IsRequired, &req.MinSampleRate, &req.MaxAgeMinutes,
		&req.CreatedAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get requirement: %w", err)
	}

	return &req, nil
}

// AddRequirement adds or updates a metric requirement for a service
func (s *MetricProfileStore) AddRequirement(ctx context.Context, serviceID, metricName string, isRequired bool, minSampleRate *int, maxAgeMinutes int) (*ServiceMetricRequirement, error) {
	query := `
		INSERT INTO service_metric_requirements (
			service_id, canonical_metric_name, is_required, min_sample_rate, max_age_minutes
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (service_id, canonical_metric_name) DO UPDATE SET
			is_required = EXCLUDED.is_required,
			min_sample_rate = EXCLUDED.min_sample_rate,
			max_age_minutes = EXCLUDED.max_age_minutes,
			updated_at = NOW()
		RETURNING id, service_id, canonical_metric_name, is_required,
		          min_sample_rate, max_age_minutes, created_at, updated_at
	`

	var req ServiceMetricRequirement
	err := s.db.QueryRowContext(ctx, query,
		serviceID, metricName, isRequired, minSampleRate, maxAgeMinutes,
	).Scan(
		&req.ID, &req.ServiceID, &req.CanonicalMetricName,
		&req.IsRequired, &req.MinSampleRate, &req.MaxAgeMinutes,
		&req.CreatedAt, &req.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("add requirement: %w", err)
	}

	return &req, nil
}

// RemoveRequirement removes a metric requirement
func (s *MetricProfileStore) RemoveRequirement(ctx context.Context, serviceID, metricName string) error {
	query := `
		DELETE FROM service_metric_requirements
		WHERE service_id = $1 AND canonical_metric_name = $2
	`

	result, err := s.db.ExecContext(ctx, query, serviceID, metricName)
	if err != nil {
		return fmt.Errorf("remove requirement: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("requirement not found")
	}

	return nil
}

// ListRequirements lists all metric requirements for a service
func (s *MetricProfileStore) ListRequirements(ctx context.Context, serviceID string) ([]ServiceMetricRequirement, error) {
	query := `
		SELECT id, service_id, canonical_metric_name, is_required,
		       min_sample_rate, max_age_minutes, created_at, updated_at
		FROM service_metric_requirements
		WHERE service_id = $1
		ORDER BY canonical_metric_name
	`

	rows, err := s.db.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, fmt.Errorf("list requirements: %w", err)
	}
	defer rows.Close()

	var requirements []ServiceMetricRequirement
	for rows.Next() {
		var req ServiceMetricRequirement
		err := rows.Scan(
			&req.ID, &req.ServiceID, &req.CanonicalMetricName,
			&req.IsRequired, &req.MinSampleRate, &req.MaxAgeMinutes,
			&req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan requirement: %w", err)
		}
		requirements = append(requirements, req)
	}

	return requirements, nil
}

// GetCoverage checks metric availability for a service
// This is a placeholder - actual implementation would query metrics collector
func (s *MetricProfileStore) GetCoverage(ctx context.Context, serviceID string) ([]MetricCoverage, error) {
	// Get profile and requirements
	profile, err := s.GetProfile(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	if profile == nil {
		return []MetricCoverage{}, nil // No profile configured
	}

	requirements, err := s.ListRequirements(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("list requirements: %w", err)
	}

	// Build coverage map
	reqMap := make(map[string]ServiceMetricRequirement)
	for _, req := range requirements {
		reqMap[req.CanonicalMetricName] = req
	}

	var coverage []MetricCoverage

	// Check required metrics from profile
	for _, metricName := range profile.RequiredMetrics {
		req, hasReq := reqMap[metricName]
		maxAge := 5 // default
		if hasReq {
			maxAge = req.MaxAgeMinutes
		}

		// TODO: Query metrics collector for actual data
		// For now, return placeholder
		coverage = append(coverage, MetricCoverage{
			MetricName:    metricName,
			IsRequired:    true,
			HasData:       false, // TODO: actual check
			LastCollected: nil,   // TODO: actual timestamp
			Status:        CoverageStatusMissing,
			MaxAgeMinutes: maxAge,
		})
	}

	// Check optional metrics
	for _, metricName := range profile.OptionalMetrics {
		req, hasReq := reqMap[metricName]
		maxAge := 5
		if hasReq {
			maxAge = req.MaxAgeMinutes
		}

		coverage = append(coverage, MetricCoverage{
			MetricName:    metricName,
			IsRequired:    false,
			HasData:       false,
			LastCollected: nil,
			Status:        CoverageStatusMissing,
			MaxAgeMinutes: maxAge,
		})
	}

	return coverage, nil
}

// GetTemplates retrieves all pre-defined profile templates
func (s *MetricProfileStore) GetTemplates(ctx context.Context) ([]MetricProfileTemplate, error) {
	query := `
		SELECT id, name, profile_type, description, required_metrics, optional_metrics, created_at
		FROM metric_profile_templates
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get templates: %w", err)
	}
	defer rows.Close()

	var templates []MetricProfileTemplate
	for rows.Next() {
		var t MetricProfileTemplate
		err := rows.Scan(
			&t.ID, &t.Name, &t.ProfileType, &t.Description,
			pq.Array(&t.RequiredMetrics), pq.Array(&t.OptionalMetrics),
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}

	return templates, nil
}
