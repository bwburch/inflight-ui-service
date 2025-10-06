package auth

import (
	"net/http"

	"github.com/bwburch/inflight-ui-service/internal/storage/sessions"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

const (
	SessionCookieName = "session_id"
	UserContextKey    = "user"
)

// Middleware handles session authentication
type Middleware struct {
	sessionStore *sessions.Store
	userStore    *users.Store
}

// NewMiddleware creates authentication middleware
func NewMiddleware(sessionStore *sessions.Store, userStore *users.Store) *Middleware {
	return &Middleware{
		sessionStore: sessionStore,
		userStore:    userStore,
	}
}

// RequireAuth validates session and injects user into context
func (m *Middleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get session cookie
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
		}

		sessionID := cookie.Value
		if sessionID == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
		}

		// Get session from Redis
		session, err := m.sessionStore.Get(c.Request().Context(), sessionID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "session validation failed")
		}

		if session == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired session")
		}

		// Get user from database
		user, err := m.userStore.Get(c.Request().Context(), session.UserID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "user lookup failed")
		}

		if user == nil || !user.IsActive {
			return echo.NewHTTPError(http.StatusUnauthorized, "user not found or inactive")
		}

		// Update session activity (sliding window)
		if err := m.sessionStore.UpdateActivity(c.Request().Context(), sessionID); err != nil {
			// Log but don't fail the request
			c.Logger().Warn("failed to update session activity:", err)
		}

		// Inject user into context
		c.Set(UserContextKey, user)

		return next(c)
	}
}

// OptionalAuth checks for auth but doesn't require it
func (m *Middleware) OptionalAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(SessionCookieName)
		if err == nil && cookie.Value != "" {
			session, err := m.sessionStore.Get(c.Request().Context(), cookie.Value)
			if err == nil && session != nil {
				user, err := m.userStore.Get(c.Request().Context(), session.UserID)
				if err == nil && user != nil && user.IsActive {
					c.Set(UserContextKey, user)
					m.sessionStore.UpdateActivity(c.Request().Context(), cookie.Value)
				}
			}
		}
		return next(c)
	}
}

// GetUserFromContext extracts the authenticated user from context
func GetUserFromContext(c echo.Context) *users.User {
	user, ok := c.Get(UserContextKey).(*users.User)
	if !ok {
		return nil
	}
	return user
}
