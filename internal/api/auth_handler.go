package api

import (
	"net/http"
	"time"

	"github.com/bwburch/inflight-ui-service/internal/auth"
	"github.com/bwburch/inflight-ui-service/internal/storage/sessions"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userStore    *users.Store
	sessionStore *sessions.Store
}

func NewAuthHandler(userStore *users.Store, sessionStore *sessions.Store) *AuthHandler {
	return &AuthHandler{
		userStore:    userStore,
		sessionStore: sessionStore,
	}
}

// Login authenticates a user and creates a session
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validation
	if input.Username == "" || input.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}

	ctx := c.Request().Context()

	// Find user by username (we need to add this method to users.Store)
	user, err := h.userStore.GetByUsername(ctx, input.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return echo.NewHTTPError(http.StatusUnauthorized, "account is disabled")
	}

	// Verify password
	if user.PasswordHash == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "account not configured")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Create session
	session, err := h.sessionStore.Create(ctx, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create session")
	}

	// Update last login
	h.userStore.UpdateLastLogin(ctx, user.ID)

	// Set session cookie
	cookie := &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    session.SessionID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // Enable in production with HTTPS
	}
	c.SetCookie(cookie)

	// Return user (without password hash)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":       user,
		"session_id": session.SessionID,
		"expires_at": session.ExpiresAt,
	})
}

// Logout destroys the current session
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(auth.SessionCookieName)
	if err != nil {
		// No cookie, nothing to do
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}

	// Delete session from Redis
	if err := h.sessionStore.Delete(c.Request().Context(), cookie.Value); err != nil {
		c.Logger().Warn("failed to delete session:", err)
	}

	// Clear cookie
	cookie = &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// Me returns the current authenticated user
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c echo.Context) error {
	user := auth.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user,
	})
}
