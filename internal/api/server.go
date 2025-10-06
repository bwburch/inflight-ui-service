package api

import (
	"database/sql"
	"net/http"

	"github.com/bwburch/inflight-ui-service/internal/storage/templates"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

type Server struct {
	echo             *echo.Echo
	db               *sql.DB
	templatesHandler *TemplatesHandler
	usersHandler     *UsersHandler
	logger           *logrus.Logger
}

func NewServer(db *sql.DB, logger *logrus.Logger) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize handlers
	templatesStore := templates.NewStore(db)
	templatesHandler := NewTemplatesHandler(templatesStore)

	usersStore := users.NewStore(db)
	usersHandler := NewUsersHandler(usersStore)

	s := &Server{
		echo:             e,
		db:               db,
		templatesHandler: templatesHandler,
		usersHandler:     usersHandler,
		logger:           logger,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Health endpoints
	s.echo.GET("/health", s.handleHealth)
	s.echo.GET("/ready", s.handleReady)

	// API v1
	v1 := s.echo.Group("/api/v1")

	// Templates
	templates := v1.Group("/templates")
	templates.GET("", s.templatesHandler.ListTemplates)
	templates.POST("", s.templatesHandler.CreateTemplate)
	templates.GET("/:id", s.templatesHandler.GetTemplate)
	templates.PUT("/:id", s.templatesHandler.UpdateTemplate)
	templates.DELETE("/:id", s.templatesHandler.DeleteTemplate)

	// Users
	usersGroup := v1.Group("/users")
	usersGroup.GET("", s.usersHandler.ListUsers)
	usersGroup.POST("", s.usersHandler.CreateUser)
	usersGroup.GET("/:id", s.usersHandler.GetUser)
	usersGroup.PUT("/:id", s.usersHandler.UpdateUser)
	usersGroup.DELETE("/:id", s.usersHandler.DeleteUser)
	usersGroup.PUT("/:id/password", s.usersHandler.UpdatePassword)
}

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *Server) handleReady(c echo.Context) error {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"reason": "database unavailable",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (s *Server) Start(address string) error {
	s.logger.Infof("Starting UI service on %s", address)
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	return s.echo.Shutdown(nil)
}
