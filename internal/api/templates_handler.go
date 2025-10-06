package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/templates"
	"github.com/labstack/echo/v4"
)

type TemplatesHandler struct {
	store *templates.Store
}

func NewTemplatesHandler(store *templates.Store) *TemplatesHandler {
	return &TemplatesHandler{store: store}
}

// ListTemplates returns all templates for the current user
func (h *TemplatesHandler) ListTemplates(c echo.Context) error {
	// TODO: Get user ID from auth context
	userID := 1 // Placeholder - will come from JWT/session

	templates, err := h.store.List(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to list templates",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}

// GetTemplate returns a specific template
func (h *TemplatesHandler) GetTemplate(c echo.Context) error {
	userID := 1 // Placeholder

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid template ID",
		})
	}

	template, err := h.store.Get(c.Request().Context(), id, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"template": template,
	})
}

// CreateTemplate creates a new template
func (h *TemplatesHandler) CreateTemplate(c echo.Context) error {
	userID := 1 // Placeholder

	var req struct {
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		ConfigurationData json.RawMessage `json:"configuration_data"`
		IsShared     bool            `json:"is_shared"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Name is required",
		})
	}

	template, err := h.store.Create(c.Request().Context(), templates.CreateTemplateInput{
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Description,
		ConfigurationData: req.ConfigurationData,
		IsShared:     req.IsShared,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to create template: %v", err),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"template": template,
	})
}

// UpdateTemplate updates a template
func (h *TemplatesHandler) UpdateTemplate(c echo.Context) error {
	userID := 1 // Placeholder

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid template ID",
		})
	}

	var req struct {
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		ConfigurationData json.RawMessage `json:"configuration_data"`
		IsShared     bool            `json:"is_shared"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
		})
	}

	template, err := h.store.Update(c.Request().Context(), id, userID, templates.UpdateTemplateInput{
		Name:         req.Name,
		Description:  req.Description,
		ConfigurationData: req.ConfigurationData,
		IsShared:     req.IsShared,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"template": template,
	})
}

// DeleteTemplate deletes a template
func (h *TemplatesHandler) DeleteTemplate(c echo.Context) error {
	userID := 1 // Placeholder

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid template ID",
		})
	}

	if err := h.store.Delete(c.Request().Context(), id, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

type ErrorResponse struct {
	Error string `json:"error"`
}
