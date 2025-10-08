package api

import (
	"net/http"

	"github.com/bwburch/inflight-ui-service/internal/storage/metrics"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

type MetricsProfilesHandler struct {
	profileStore *metrics.MetricProfileStore
}

func NewMetricsProfilesHandler(profileStore *metrics.MetricProfileStore) *MetricsProfilesHandler {
	return &MetricsProfilesHandler{
		profileStore: profileStore,
	}
}

// GetServiceProfile retrieves a service's metric profile
// GET /api/v1/services/:id/metrics/profile
func (h *MetricsProfilesHandler) GetServiceProfile(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	profile, err := h.profileStore.GetProfile(ctx, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get profile")
	}

	if profile == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"profile": nil,
			"message": "no profile configured for this service",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

// UpsertServiceProfile creates or updates a service's metric profile
// POST /api/v1/services/:id/metrics/profile
func (h *MetricsProfilesHandler) UpsertServiceProfile(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	// Get user from context
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	var req struct {
		ProfileType     metrics.ProfileType `json:"profile_type"`
		RequiredMetrics []string            `json:"required_metrics"`
		OptionalMetrics []string            `json:"optional_metrics"`
		SamplingRate    int                 `json:"sampling_rate"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate profile type
	validTypes := []metrics.ProfileType{
		metrics.ProfileTypeBatch,
		metrics.ProfileTypeHighThroughput,
		metrics.ProfileTypeStreaming,
		metrics.ProfileTypeCustom,
	}
	isValid := false
	for _, t := range validTypes {
		if req.ProfileType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid profile type")
	}

	// Default sampling rate
	if req.SamplingRate == 0 {
		req.SamplingRate = 60
	}

	profile, err := h.profileStore.UpsertProfile(ctx, metrics.UpsertProfileInput{
		ServiceID:       serviceID,
		ProfileType:     req.ProfileType,
		RequiredMetrics: req.RequiredMetrics,
		OptionalMetrics: req.OptionalMetrics,
		SamplingRate:    req.SamplingRate,
		UserID:          user.ID,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to upsert profile")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

// DeleteServiceProfile deletes a service's metric profile
// DELETE /api/v1/services/:id/metrics/profile
func (h *MetricsProfilesHandler) DeleteServiceProfile(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	if err := h.profileStore.DeleteProfile(ctx, serviceID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete profile")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "profile deleted",
	})
}

// AddMetricRequirement adds or updates a metric requirement
// POST /api/v1/services/:id/metrics/requirements
func (h *MetricsProfilesHandler) AddMetricRequirement(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	var req struct {
		MetricName    string `json:"metric_name"`
		IsRequired    bool   `json:"is_required"`
		MinSampleRate *int   `json:"min_sample_rate,omitempty"`
		MaxAgeMinutes int    `json:"max_age_minutes"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.MetricName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "metric_name is required")
	}

	// Default max age
	if req.MaxAgeMinutes == 0 {
		req.MaxAgeMinutes = 5
	}

	requirement, err := h.profileStore.AddRequirement(
		ctx, serviceID, req.MetricName, req.IsRequired, req.MinSampleRate, req.MaxAgeMinutes,
	)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add requirement")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"requirement": requirement,
	})
}

// RemoveMetricRequirement removes a metric requirement
// DELETE /api/v1/services/:id/metrics/requirements/:metricName
func (h *MetricsProfilesHandler) RemoveMetricRequirement(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")
	metricName := c.Param("metricName")

	if err := h.profileStore.RemoveRequirement(ctx, serviceID, metricName); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove requirement")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "requirement removed",
	})
}

// ListMetricRequirements lists all metric requirements for a service
// GET /api/v1/services/:id/metrics/requirements
func (h *MetricsProfilesHandler) ListMetricRequirements(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	requirements, err := h.profileStore.ListRequirements(ctx, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list requirements")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"requirements": requirements,
	})
}

// GetMetricCoverage checks which required metrics have data
// GET /api/v1/services/:id/metrics/coverage
func (h *MetricsProfilesHandler) GetMetricCoverage(c echo.Context) error {
	ctx := c.Request().Context()
	serviceID := c.Param("id")

	coverage, err := h.profileStore.GetCoverage(ctx, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get coverage")
	}

	// Calculate summary stats
	total := len(coverage)
	requiredCount := 0
	availableCount := 0
	staleCount := 0
	missingCount := 0

	for _, cov := range coverage {
		if cov.IsRequired {
			requiredCount++
		}
		switch cov.Status {
		case metrics.CoverageStatusOK:
			availableCount++
		case metrics.CoverageStatusStale:
			staleCount++
		case metrics.CoverageStatusMissing:
			missingCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"coverage": coverage,
		"summary": map[string]int{
			"total":     total,
			"required":  requiredCount,
			"available": availableCount,
			"stale":     staleCount,
			"missing":   missingCount,
		},
	})
}

// GetProfileTemplates retrieves all pre-defined profile templates
// GET /api/v1/metrics/templates
func (h *MetricsProfilesHandler) GetProfileTemplates(c echo.Context) error {
	ctx := c.Request().Context()

	templates, err := h.profileStore.GetTemplates(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get templates")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}

// CreateProfileTemplate creates a new profile template
// POST /api/v1/metrics/templates
func (h *MetricsProfilesHandler) CreateProfileTemplate(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Name            string   `json:"name"`
		ProfileType     string   `json:"profile_type"`
		Description     string   `json:"description"`
		RequiredMetrics []string `json:"required_metrics"`
		OptionalMetrics []string `json:"optional_metrics"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.ProfileType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and profile_type are required")
	}

	template, err := h.profileStore.CreateTemplate(ctx, req.Name, req.ProfileType, req.Description, req.RequiredMetrics, req.OptionalMetrics)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create template")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"template": template,
	})
}

// UpdateProfileTemplate updates an existing profile template
// PUT /api/v1/metrics/templates/:id
func (h *MetricsProfilesHandler) UpdateProfileTemplate(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var req struct {
		Name            string   `json:"name"`
		Description     string   `json:"description"`
		RequiredMetrics []string `json:"required_metrics"`
		OptionalMetrics []string `json:"optional_metrics"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	template, err := h.profileStore.UpdateTemplate(ctx, id, req.Name, req.Description, req.RequiredMetrics, req.OptionalMetrics)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update template")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"template": template,
	})
}

// DeleteProfileTemplate deletes a profile template
// DELETE /api/v1/metrics/templates/:id
func (h *MetricsProfilesHandler) DeleteProfileTemplate(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	if err := h.profileStore.DeleteTemplate(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete template")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "template deleted",
	})
}

// RegisterRoutes registers all metrics profile routes
func (h *MetricsProfilesHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	// Service-specific routes
	e.GET("/services/:id/metrics/profile", h.GetServiceProfile, authMiddleware)
	e.POST("/services/:id/metrics/profile", h.UpsertServiceProfile, authMiddleware)
	e.DELETE("/services/:id/metrics/profile", h.DeleteServiceProfile, authMiddleware)
	e.GET("/services/:id/metrics/requirements", h.ListMetricRequirements, authMiddleware)
	e.POST("/services/:id/metrics/requirements", h.AddMetricRequirement, authMiddleware)
	e.DELETE("/services/:id/metrics/requirements/:metricName", h.RemoveMetricRequirement, authMiddleware)
	e.GET("/services/:id/metrics/coverage", h.GetMetricCoverage, authMiddleware)

	// Global routes
	e.GET("/metrics/templates", h.GetProfileTemplates, authMiddleware)
	e.POST("/metrics/templates", h.CreateProfileTemplate, authMiddleware)
	e.PUT("/metrics/templates/:id", h.UpdateProfileTemplate, authMiddleware)
	e.DELETE("/metrics/templates/:id", h.DeleteProfileTemplate, authMiddleware)
}
