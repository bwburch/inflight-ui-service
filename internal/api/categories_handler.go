package api

import (
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/categories"
	"github.com/labstack/echo/v4"
)

type CategoriesHandler struct {
	store *categories.Store
}

func NewCategoriesHandler(store *categories.Store) *CategoriesHandler {
	return &CategoriesHandler{store: store}
}

// ListCategories returns all active categories
// GET /api/v1/configuration/categories
func (h *CategoriesHandler) ListCategories(c echo.Context) error {
	ctx := c.Request().Context()

	// Check if "all" query param is set to include inactive
	includeInactive := c.QueryParam("all") == "true"

	var cats []categories.Category
	var err error

	if includeInactive {
		cats, err = h.store.ListAll(ctx)
	} else {
		cats, err = h.store.List(ctx)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list categories")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": cats,
	})
}

// GetCategory returns a specific category by ID
// GET /api/v1/configuration/categories/:id
func (h *CategoriesHandler) GetCategory(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	category, err := h.store.GetByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"category": category,
	})
}

// CreateCategory creates a new category
// POST /api/v1/configuration/categories
func (h *CategoriesHandler) CreateCategory(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Name         string `json:"name"`
		DisplayName  string `json:"display_name"`
		Description  string `json:"description"`
		Color        string `json:"color"`
		Icon         string `json:"icon"`
		DisplayOrder int    `json:"display_order"`
		IsActive     bool   `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and display_name are required")
	}

	category, err := h.store.Create(ctx, categories.CreateInput{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Color:        req.Color,
		Icon:         req.Icon,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create category")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"category": category,
	})
}

// UpdateCategory updates an existing category
// PUT /api/v1/configuration/categories/:id
func (h *CategoriesHandler) UpdateCategory(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	var req struct {
		DisplayName  string `json:"display_name"`
		Description  string `json:"description"`
		Color        string `json:"color"`
		Icon         string `json:"icon"`
		DisplayOrder int    `json:"display_order"`
		IsActive     bool   `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "display_name is required")
	}

	category, err := h.store.Update(ctx, id, categories.UpdateInput{
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Color:        req.Color,
		Icon:         req.Icon,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"category": category,
	})
}

// DeleteCategory deletes a category
// DELETE /api/v1/configuration/categories/:id
func (h *CategoriesHandler) DeleteCategory(c echo.Context) error {
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

// RegisterRoutes registers all category routes
func (h *CategoriesHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	// Public routes (read-only)
	e.GET("/categories", h.ListCategories)
	e.GET("/categories/:id", h.GetCategory)

	// Protected routes (admin only - require auth)
	e.POST("/categories", h.CreateCategory, authMiddleware)
	e.PUT("/categories/:id", h.UpdateCategory, authMiddleware)
	e.DELETE("/categories/:id", h.DeleteCategory, authMiddleware)
}
