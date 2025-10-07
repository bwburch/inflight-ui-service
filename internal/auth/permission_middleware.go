package auth

import (
	"net/http"

	"github.com/bwburch/inflight-ui-service/internal/storage/rbac"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

// PermissionMiddleware creates middleware that checks if the user has required permission(s)
func PermissionMiddleware(userRoleStore *rbac.UserRoleStore, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user from context (set by AuthMiddleware)
			user, ok := c.Get("user").(*users.User)
			if !ok || user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			ctx := c.Request().Context()

			// Check if user is admin (bypass permission check)
			isAdmin, err := userRoleStore.IsAdmin(ctx, user.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "permission check failed")
			}

			if isAdmin {
				// Admins have all permissions
				return next(c)
			}

			// Check if user has the required permission
			hasPermission, err := userRoleStore.CheckPermission(ctx, user.ID, permission)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "permission check failed")
			}

			if !hasPermission {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}

			return next(c)
		}
	}
}

// AnyPermissionMiddleware checks if user has ANY of the specified permissions
func AnyPermissionMiddleware(userRoleStore *rbac.UserRoleStore, permissions []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*users.User)
			if !ok || user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			ctx := c.Request().Context()

			// Check if admin
			isAdmin, err := userRoleStore.IsAdmin(ctx, user.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "permission check failed")
			}

			if isAdmin {
				return next(c)
			}

			// Check if user has any of the required permissions
			hasAny, err := userRoleStore.CheckAnyPermission(ctx, user.ID, permissions)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "permission check failed")
			}

			if !hasAny {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}

			return next(c)
		}
	}
}

// RequirePermission is a helper function to create permission middleware
func RequirePermission(userRoleStore *rbac.UserRoleStore, permission string) echo.MiddlewareFunc {
	return PermissionMiddleware(userRoleStore, permission)
}

// RequireAnyPermission is a helper function to create "any permission" middleware
func RequireAnyPermission(userRoleStore *rbac.UserRoleStore, permissions ...string) echo.MiddlewareFunc {
	return AnyPermissionMiddleware(userRoleStore, permissions)
}
