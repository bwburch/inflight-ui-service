package api

import (
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/changetypes"
	"github.com/labstack/echo/v4"
)

type ChangeTypesHandler struct {
	store *changetypes.Store
}

func NewChangeTypesHandler(store *changetypes.Store) *ChangeTypesHandler {
	return &ChangeTypesHandler{store: store}
}

// ListChangeTypes returns all active configuration change types
// GET /api/v1/configuration/change-types
func (h *ChangeTypesHandler) ListChangeTypes(c echo.Context) error {
	ctx := c.Request().Context()

	// Check if "all" query param is set to include inactive types
	includeInactive := c.QueryParam("all") == "true"

	var types []changetypes.ChangeType
	var err error

	if includeInactive {
		types, err = h.store.ListAll(ctx)
	} else {
		types, err = h.store.List(ctx)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list change types")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_types": types,
	})
}

// GetChangeType returns a specific change type by code
// GET /api/v1/configuration/change-types/:code
func (h *ChangeTypesHandler) GetChangeType(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.Param("code")

	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code parameter is required")
	}

	changeType, err := h.store.GetByCode(ctx, code)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_type": changeType,
	})
}

// GetChangeTypeByID returns a specific change type by ID
// GET /api/v1/configuration/change-types/id/:id
func (h *ChangeTypesHandler) GetChangeTypeByID(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	changeType, err := h.store.GetByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_type": changeType,
	})
}

// CreateChangeType creates a new change type
// POST /api/v1/configuration/change-types
func (h *ChangeTypesHandler) CreateChangeType(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Code          string   `json:"code"`
		DisplayName   string   `json:"display_name"`
		Description   string   `json:"description"`
		CategoryID    *int     `json:"category_id"`
		AllowedFields []string `json:"allowed_fields"`
		IsActive      bool     `json:"is_active"`
		DisplayOrder  int      `json:"display_order"`
		Icon          string   `json:"icon"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Code == "" || req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code and display_name are required")
	}

	changeType, err := h.store.Create(ctx, changetypes.CreateInput{
		Code:          req.Code,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		AllowedFields: req.AllowedFields,
		IsActive:      req.IsActive,
		DisplayOrder:  req.DisplayOrder,
		Icon:          req.Icon,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create change type")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"change_type": changeType,
	})
}

// UpdateChangeType updates an existing change type
// PUT /api/v1/configuration/change-types/:id
func (h *ChangeTypesHandler) UpdateChangeType(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	var req struct {
		DisplayName   string   `json:"display_name"`
		Description   string   `json:"description"`
		CategoryID    *int     `json:"category_id"`
		AllowedFields []string `json:"allowed_fields"`
		IsActive      bool     `json:"is_active"`
		DisplayOrder  int      `json:"display_order"`
		Icon          string   `json:"icon"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "display_name is required")
	}

	changeType, err := h.store.Update(ctx, id, changetypes.UpdateInput{
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		AllowedFields: req.AllowedFields,
		IsActive:      req.IsActive,
		DisplayOrder:  req.DisplayOrder,
		Icon:          req.Icon,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_type": changeType,
	})
}

// DeleteChangeType deletes a change type
// DELETE /api/v1/configuration/change-types/:id
func (h *ChangeTypesHandler) DeleteChangeType(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	if err := h.store.Delete(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}


// RegisterRoutes registers all change type routes
func (h *ChangeTypesHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	// Public routes (read-only)
	e.GET("/change-types", h.ListChangeTypes)
	e.GET("/change-types/:code", h.GetChangeType)
	e.GET("/change-types/id/:id", h.GetChangeTypeByID)

	// Protected routes (admin only - require auth)
	e.POST("/change-types", h.CreateChangeType, authMiddleware)
	e.PUT("/change-types/:id", h.UpdateChangeType, authMiddleware)
	e.DELETE("/change-types/:id", h.DeleteChangeType, authMiddleware)
}
