package api

import (
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

type UsersHandler struct {
	store *users.Store
}

func NewUsersHandler(store *users.Store) *UsersHandler {
	return &UsersHandler{store: store}
}

// ListUsers returns all users with pagination
// GET /api/v1/users?role=admin&is_active=true&limit=20&offset=0
func (h *UsersHandler) ListUsers(c echo.Context) error {
	role := c.QueryParam("role")
	if role == "" {
		role = "all"
	}

	var isActive *bool
	if isActiveStr := c.QueryParam("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		isActive = &val
	}

	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	usersList, total, err := h.store.List(c.Request().Context(), role, isActive, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  usersList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUser returns a specific user
// GET /api/v1/users/:id
func (h *UsersHandler) GetUser(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	user, err := h.store.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	return c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
// POST /api/v1/users
func (h *UsersHandler) CreateUser(c echo.Context) error {
	var input struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		FullName string `json:"full_name"`
		Password string `json:"password" validate:"required,min=8"`
		Role     string `json:"role"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Default role to 'user' if not specified
	if input.Role == "" {
		input.Role = "user"
	}

	// Validate role
	if input.Role != "admin" && input.Role != "user" && input.Role != "viewer" {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be admin, user, or viewer")
	}

	user, err := h.store.Create(c.Request().Context(), users.CreateUserInput{
		Username: input.Username,
		Email:    input.Email,
		FullName: input.FullName,
		Password: input.Password,
		Role:     input.Role,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, user)
}

// UpdateUser updates a user
// PUT /api/v1/users/:id
func (h *UsersHandler) UpdateUser(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	var input struct {
		Email    *string `json:"email,omitempty"`
		FullName *string `json:"full_name,omitempty"`
		Role     *string `json:"role,omitempty"`
		IsActive *bool   `json:"is_active,omitempty"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate role if provided
	if input.Role != nil {
		role := *input.Role
		if role != "admin" && role != "user" && role != "viewer" {
			return echo.NewHTTPError(http.StatusBadRequest, "role must be admin, user, or viewer")
		}
	}

	user, err := h.store.Update(c.Request().Context(), id, users.UpdateUserInput{
		Email:    input.Email,
		FullName: input.FullName,
		Role:     input.Role,
		IsActive: input.IsActive,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user
// DELETE /api/v1/users/:id
func (h *UsersHandler) DeleteUser(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	// Prevent deleting user ID 1 (default admin)
	if id == 1 {
		return echo.NewHTTPError(http.StatusForbidden, "cannot delete default admin user")
	}

	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// UpdatePassword changes a user's password
// PUT /api/v1/users/:id/password
func (h *UsersHandler) UpdatePassword(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	var input struct {
		Password string `json:"password" validate:"required,min=8"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.store.UpdatePassword(c.Request().Context(), id, input.Password); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
