package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/bwburch/inflight-ui-service/internal/storage/rbac"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

type RBACHandler struct {
	roleStore       *rbac.RoleStore
	permissionStore *rbac.PermissionStore
	userRoleStore   *rbac.UserRoleStore
}

func NewRBACHandler(roleStore *rbac.RoleStore, permissionStore *rbac.PermissionStore, userRoleStore *rbac.UserRoleStore) *RBACHandler {
	return &RBACHandler{
		roleStore:       roleStore,
		permissionStore: permissionStore,
		userRoleStore:   userRoleStore,
	}
}

// ============================================================================
// Permission Endpoints
// ============================================================================

// ListPermissions retrieves all available permissions
// GET /api/v1/auth/permissions
func (h *RBACHandler) ListPermissions(c echo.Context) error {
	ctx := c.Request().Context()

	permissions, err := h.permissionStore.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch permissions")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"permissions": permissions,
		"total":       len(permissions),
	})
}

// ============================================================================
// Role Endpoints
// ============================================================================

// ListRoles retrieves all roles
// GET /api/v1/auth/roles
func (h *RBACHandler) ListRoles(c echo.Context) error {
	ctx := c.Request().Context()

	roles, err := h.roleStore.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch roles")
	}

	// Enhance with permission and user counts
	rolesWithDetails := make([]map[string]interface{}, 0, len(roles))
	for _, role := range roles {
		permissions, _ := h.roleStore.GetPermissions(ctx, role.ID)
		userCount, _ := h.roleStore.GetUserCount(ctx, role.ID)

		rolesWithDetails = append(rolesWithDetails, map[string]interface{}{
			"id":               role.ID,
			"name":             role.Name,
			"description":      role.Description,
			"is_system":        role.IsSystem,
			"permission_count": len(permissions),
			"user_count":       userCount,
			"created_at":       role.CreatedAt,
			"updated_at":       role.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"roles": rolesWithDetails,
		"total": len(rolesWithDetails),
	})
}

// GetRole retrieves a role by ID with its permissions
// GET /api/v1/auth/roles/:id
func (h *RBACHandler) GetRole(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	role, err := h.roleStore.GetByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "role not found")
	}

	permissions, err := h.roleStore.GetPermissions(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch permissions")
	}

	userCount, _ := h.roleStore.GetUserCount(ctx, id)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"role":             role,
		"permissions":      permissions,
		"permission_count": len(permissions),
		"user_count":       userCount,
	})
}

// CreateRole creates a new custom role
// POST /api/v1/auth/roles
func (h *RBACHandler) CreateRole(c echo.Context) error {
	ctx := c.Request().Context()

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "role name is required")
	}

	role, err := h.roleStore.Create(ctx, input.Name, input.Description)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create role")
	}

	return c.JSON(http.StatusCreated, role)
}

// UpdateRole updates a role's name and description
// PUT /api/v1/auth/roles/:id
func (h *RBACHandler) UpdateRole(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.roleStore.Update(ctx, id, input.Name, input.Description); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update role")
	}

	role, _ := h.roleStore.GetByID(ctx, id)
	return c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a custom role
// DELETE /api/v1/auth/roles/:id
func (h *RBACHandler) DeleteRole(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	if err := h.roleStore.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusForbidden, "cannot delete system role")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete role")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role deleted"})
}

// GrantPermissionToRole grants a permission to a role
// POST /api/v1/auth/roles/:id/permissions
func (h *RBACHandler) GrantPermissionToRole(c echo.Context) error {
	ctx := c.Request().Context()

	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	var input struct {
		PermissionID int `json:"permission_id"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get current user ID from context (set by auth middleware)
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	if err := h.roleStore.GrantPermission(ctx, roleID, input.PermissionID, user.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to grant permission")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "permission granted"})
}

// RevokePermissionFromRole revokes a permission from a role
// DELETE /api/v1/auth/roles/:id/permissions/:permissionId
func (h *RBACHandler) RevokePermissionFromRole(c echo.Context) error {
	ctx := c.Request().Context()

	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	permissionID, err := strconv.Atoi(c.Param("permissionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid permission ID")
	}

	if err := h.roleStore.RevokePermission(ctx, roleID, permissionID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to revoke permission")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "permission revoked"})
}

// ============================================================================
// User Role Endpoints
// ============================================================================

// GetUserRoles retrieves all roles for a user
// GET /api/v1/auth/users/:id/roles
func (h *RBACHandler) GetUserRoles(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	roles, err := h.userRoleStore.GetUserRoles(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user roles")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"roles":   roles,
		"total":   len(roles),
	})
}

// GetUserPermissions retrieves all effective permissions for a user
// GET /api/v1/auth/users/:id/permissions
func (h *RBACHandler) GetUserPermissions(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	permissions, err := h.userRoleStore.GetUserPermissions(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user permissions")
	}

	return c.JSON(http.StatusOK, permissions)
}

// GetMyPermissions retrieves the current user's permissions
// GET /api/v1/auth/me/permissions
func (h *RBACHandler) GetMyPermissions(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user ID from context (set by auth middleware)
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	permissions, err := h.userRoleStore.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch permissions")
	}

	return c.JSON(http.StatusOK, permissions)
}

// AssignRoleToUser assigns a role to a user
// POST /api/v1/auth/users/:id/roles
func (h *RBACHandler) AssignRoleToUser(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	var input struct {
		RoleID    int        `json:"role_id"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get current user ID (who is assigning the role)
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}
	assignedBy := user.ID

	if err := h.userRoleStore.AssignRole(ctx, userID, input.RoleID, assignedBy, input.ExpiresAt); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to assign role")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role assigned"})
}

// RemoveRoleFromUser removes a role from a user
// DELETE /api/v1/auth/users/:id/roles/:roleId
func (h *RBACHandler) RemoveRoleFromUser(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	roleID, err := strconv.Atoi(c.Param("roleId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role ID")
	}

	if err := h.userRoleStore.RemoveRole(ctx, userID, roleID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove role")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role removed"})
}

// CheckPermission checks if the current user has a specific permission
// POST /api/v1/auth/check
func (h *RBACHandler) CheckPermission(c echo.Context) error {
	ctx := c.Request().Context()

	var input struct {
		Permission string `json:"permission"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	// Check if admin (bypass permission check)
	isAdmin, _ := h.userRoleStore.IsAdmin(ctx, user.ID)
	if isAdmin {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"permission": input.Permission,
			"granted":    true,
			"reason":     "admin",
		})
	}

	hasPermission, err := h.userRoleStore.CheckPermission(ctx, user.ID, input.Permission)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check permission")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"permission": input.Permission,
		"granted":    hasPermission,
	})
}

// ============================================================================
// Route Registration
// ============================================================================

func (h *RBACHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	// All RBAC endpoints require authentication
	// Permission endpoints
	e.GET("/permissions", h.ListPermissions, authMiddleware)

	// Role endpoints
	e.GET("/roles", h.ListRoles, authMiddleware)
	e.GET("/roles/:id", h.GetRole, authMiddleware)
	e.POST("/roles", h.CreateRole, authMiddleware)           // Requires 'roles.create'
	e.PUT("/roles/:id", h.UpdateRole, authMiddleware)        // Requires 'roles.edit'
	e.DELETE("/roles/:id", h.DeleteRole, authMiddleware)     // Requires 'roles.delete'

	// Role permission management
	e.POST("/roles/:id/permissions", h.GrantPermissionToRole, authMiddleware)            // Requires 'roles.edit'
	e.DELETE("/roles/:id/permissions/:permissionId", h.RevokePermissionFromRole, authMiddleware) // Requires 'roles.edit'

	// User role management
	e.GET("/users/:id/roles", h.GetUserRoles, authMiddleware)              // Requires 'users.view'
	e.GET("/users/:id/permissions", h.GetUserPermissions, authMiddleware)  // Requires 'users.view'
	e.POST("/users/:id/roles", h.AssignRoleToUser, authMiddleware)         // Requires 'users.manage_roles'
	e.DELETE("/users/:id/roles/:roleId", h.RemoveRoleFromUser, authMiddleware) // Requires 'users.manage_roles'

	// Current user permissions
	e.GET("/me/permissions", h.GetMyPermissions, authMiddleware) // Always allowed for authenticated users
	e.POST("/check", h.CheckPermission, authMiddleware)          // Always allowed for authenticated users
}
