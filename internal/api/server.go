package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/bwburch/inflight-ui-service/internal/auth"
	"github.com/bwburch/inflight-ui-service/internal/storage/categories"
	"github.com/bwburch/inflight-ui-service/internal/storage/metrics"
	"github.com/bwburch/inflight-ui-service/internal/storage/profiles"
	"github.com/bwburch/inflight-ui-service/internal/storage/rbac"
	"github.com/bwburch/inflight-ui-service/internal/storage/sessions"
	"github.com/bwburch/inflight-ui-service/internal/storage/simulations"
	"github.com/bwburch/inflight-ui-service/internal/storage/templates"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/bwburch/inflight-ui-service/internal/worker"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Server struct {
	echo                   *echo.Echo
	db                     *sql.DB
	redis                  *redis.Client
	templatesHandler       *TemplatesHandler
	usersHandler           *UsersHandler
	authHandler            *AuthHandler
	rbacHandler            *RBACHandler
	simulationQueueHandler *SimulationQueueHandler
	metricsProfilesHandler *MetricsProfilesHandler
	attachmentsHandler     *AttachmentsHandler
	categoriesHandler      *CategoriesHandler
	profilesHandler        *ProfilesHandler
	authMiddleware         *auth.Middleware
	simulationWorker       *worker.SimulationWorker
	logger                 *logrus.Logger
}

func NewServer(db *sql.DB, redisClient *redis.Client, logger *logrus.Logger) *Server {
	e := echo.New()
	e.HideBanner = true

	// Disable validator - we'll do manual validation
	e.Validator = nil

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize stores
	templatesStore := templates.NewStore(db)
	usersStore := users.NewStore(db)
	sessionStore := sessions.NewStore(redisClient)
	roleStore := rbac.NewRoleStore(db)
	permissionStore := rbac.NewPermissionStore(db)
	userRoleStore := rbac.NewUserRoleStore(db)
	jobQueueStore := simulations.NewJobQueueStore(db)
	metricProfileStore := metrics.NewMetricProfileStore(db)
	categoriesStore := categories.NewStore(db)
	profilesStore := profiles.NewStore(db)

	// Initialize S3 attachment store with MinIO configuration
	// TODO: Move to config file
	minioEndpoint := "localhost:9010"         // MinIO API port
	minioAccessKey := "admin"                 // MinIO root user
	minioSecretKey := "admin_password"        // MinIO root password
	minioBucket := "inflight-simulations"     // Bucket name
	minioUseSSL := false                      // Local development = no SSL

	attachmentStore, err := simulations.NewS3AttachmentStore(
		db,
		minioEndpoint,
		minioAccessKey,
		minioSecretKey,
		minioBucket,
		minioUseSSL,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize S3 attachment store: %v", err)
	}

	logger.Info("S3 attachment store initialized with MinIO backend")

	// Initialize handlers
	templatesHandler := NewTemplatesHandler(templatesStore)
	usersHandler := NewUsersHandler(usersStore)
	authHandler := NewAuthHandler(usersStore, sessionStore)
	rbacHandler := NewRBACHandler(roleStore, permissionStore, userRoleStore)
	simulationQueueHandler := NewSimulationQueueHandler(jobQueueStore)
	metricsProfilesHandler := NewMetricsProfilesHandler(metricProfileStore)
	attachmentsHandler := NewAttachmentsHandler(attachmentStore, jobQueueStore)
	categoriesHandler := NewCategoriesHandler(categoriesStore)
	profilesHandler := NewProfilesHandler(profilesStore)

	// Initialize auth middleware
	authMiddleware := auth.NewMiddleware(sessionStore, usersStore)

	// Initialize simulation worker
	advisorURL := "http://localhost:8082" // TODO: Make configurable
	simulationWorker := worker.NewSimulationWorker(jobQueueStore, advisorURL, logger)

	s := &Server{
		echo:                   e,
		db:                     db,
		redis:                  redisClient,
		templatesHandler:       templatesHandler,
		usersHandler:           usersHandler,
		authHandler:            authHandler,
		rbacHandler:            rbacHandler,
		simulationQueueHandler: simulationQueueHandler,
		metricsProfilesHandler: metricsProfilesHandler,
		attachmentsHandler:     attachmentsHandler,
		categoriesHandler:      categoriesHandler,
		profilesHandler:        profilesHandler,
		authMiddleware:         authMiddleware,
		simulationWorker:       simulationWorker,
		logger:                 logger,
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

	// Auth endpoints (no auth required)
	authGroup := v1.Group("/auth")
	authGroup.POST("/login", s.authHandler.Login)
	authGroup.POST("/logout", s.authHandler.Logout)
	authGroup.GET("/me", s.authHandler.Me, s.authMiddleware.RequireAuth)

	// RBAC endpoints (auth required)
	s.rbacHandler.RegisterRoutes(authGroup, s.authMiddleware.RequireAuth)

	// Simulation queue endpoints (auth required)
	simulations := v1.Group("/simulations")
	s.simulationQueueHandler.RegisterRoutes(simulations, s.authMiddleware.RequireAuth)

	// Metrics profiles endpoints (auth required)
	s.metricsProfilesHandler.RegisterRoutes(v1, s.authMiddleware.RequireAuth)

	// Attachments endpoints (auth required)
	s.attachmentsHandler.RegisterRoutes(simulations, s.authMiddleware.RequireAuth)

	// Configuration endpoints
	configGroup := v1.Group("/configuration")
	s.categoriesHandler.RegisterRoutes(configGroup, s.authMiddleware.RequireAuth)  // Auth required for write operations
	s.profilesHandler.RegisterRoutes(configGroup, s.authMiddleware.RequireAuth)    // Auth required for write operations

	// Protected endpoints (auth required)
	// Templates
	templates := v1.Group("/templates", s.authMiddleware.RequireAuth)
	templates.GET("", s.templatesHandler.ListTemplates)
	templates.POST("", s.templatesHandler.CreateTemplate)
	templates.GET("/:id", s.templatesHandler.GetTemplate)
	templates.PUT("/:id", s.templatesHandler.UpdateTemplate)
	templates.DELETE("/:id", s.templatesHandler.DeleteTemplate)

	// Users (admin only - for now just require auth)
	usersGroup := v1.Group("/users", s.authMiddleware.RequireAuth)
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
	// Start simulation worker in background
	go s.simulationWorker.Start(context.Background())

	s.logger.Infof("Starting UI service on %s (with simulation queue worker)", address)
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down server...")
	s.simulationWorker.Stop()
	return s.echo.Shutdown(context.Background())
}
