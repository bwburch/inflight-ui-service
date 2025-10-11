package api

import (
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/profiles"
	"github.com/labstack/echo/v4"
)

type ProfilesHandler struct {
	store *profiles.Store
}

func NewProfilesHandler(store *profiles.Store) *ProfilesHandler {
	return &ProfilesHandler{store: store}
}

// ListProfiles returns all active profiles
// GET /api/v1/configuration/profiles
func (h *ProfilesHandler) ListProfiles(c echo.Context) error {
	ctx := c.Request().Context()
	includeInactive := c.QueryParam("all") == "true"

	var profileList []profiles.ServiceProfile
	var err error

	if includeInactive {
		profileList, err = h.store.ListAll(ctx)
	} else {
		profileList, err = h.store.List(ctx)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list profiles")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"profiles": profileList,
	})
}


// GetProfile returns a specific profile
// GET /api/v1/configuration/profiles/:id
func (h *ProfilesHandler) GetProfile(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid profile ID")
	}

	profile, err := h.store.GetByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

// CreateProfile creates a new profile
// POST /api/v1/configuration/profiles
func (h *ProfilesHandler) CreateProfile(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"display_name"`
		Description                string   `json:"description"`
		RequiredMetrics            []string `json:"required_metrics"`
		RecommendedMetrics         []string `json:"recommended_metrics"`
		AllowedConfigurationFields []string `json:"allowed_configuration_fields"`
		Icon                       string   `json:"icon"`
		Color                      string   `json:"color"`
		DisplayOrder               int      `json:"display_order"`
		IsActive                   bool     `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and display_name are required")
	}

	profile, err := h.store.Create(ctx, profiles.CreateInput{
		Name:                       req.Name,
		DisplayName:                req.DisplayName,
		Description:                req.Description,
		RequiredMetrics:            req.RequiredMetrics,
		RecommendedMetrics:         req.RecommendedMetrics,
		AllowedConfigurationFields: req.AllowedConfigurationFields,
		Icon:                       req.Icon,
		Color:                      req.Color,
		DisplayOrder:               req.DisplayOrder,
		IsActive:                   req.IsActive,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create profile")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"profile": profile,
	})
}

// UpdateProfile updates an existing profile
// PUT /api/v1/configuration/profiles/:id
func (h *ProfilesHandler) UpdateProfile(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid profile ID")
	}

	var req struct {
		DisplayName                string   `json:"display_name"`
		Description                string   `json:"description"`
		RequiredMetrics            []string `json:"required_metrics"`
		RecommendedMetrics         []string `json:"recommended_metrics"`
		AllowedConfigurationFields []string `json:"allowed_configuration_fields"`
		Icon                       string   `json:"icon"`
		Color                      string   `json:"color"`
		DisplayOrder               int      `json:"display_order"`
		IsActive                   bool     `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	profile, err := h.store.Update(ctx, id, profiles.UpdateInput{
		DisplayName:                req.DisplayName,
		Description:                req.Description,
		RequiredMetrics:            req.RequiredMetrics,
		RecommendedMetrics:         req.RecommendedMetrics,
		AllowedConfigurationFields: req.AllowedConfigurationFields,
		Icon:                       req.Icon,
		Color:                      req.Color,
		DisplayOrder:               req.DisplayOrder,
		IsActive:                   req.IsActive,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

// DeleteProfile deletes a profile
// DELETE /api/v1/configuration/profiles/:id
func (h *ProfilesHandler) DeleteProfile(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid profile ID")
	}

	if err := h.store.Delete(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// RegisterRoutes registers all profile routes
func (h *ProfilesHandler) RegisterRoutes(configGroup *echo.Group, authMiddleware echo.MiddlewareFunc) {
	// Public routes (read-only)
	configGroup.GET("/profiles", h.ListProfiles)
	configGroup.GET("/profiles/:id", h.GetProfile)

	// Protected routes (admin only - require auth)
	configGroup.POST("/profiles", h.CreateProfile, authMiddleware)
	configGroup.PUT("/profiles/:id", h.UpdateProfile, authMiddleware)
	configGroup.DELETE("/profiles/:id", h.DeleteProfile, authMiddleware)
}
