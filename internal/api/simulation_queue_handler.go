package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bwburch/inflight-ui-service/internal/storage/simulations"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

type SimulationQueueHandler struct {
	queueStore *simulations.JobQueueStore
}

func NewSimulationQueueHandler(queueStore *simulations.JobQueueStore) *SimulationQueueHandler {
	return &SimulationQueueHandler{
		queueStore: queueStore,
	}
}

// EnqueueSimulation creates a new simulation job
// POST /api/v1/simulations/queue
func (h *SimulationQueueHandler) EnqueueSimulation(c echo.Context) error {
	ctx := c.Request().Context()

	// Get user from context
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	var req struct {
		ServiceID       string          `json:"service_id"`
		LLMProvider     *string         `json:"llm_provider,omitempty"`
		PromptVersionID *int            `json:"prompt_version_id,omitempty"`
		CurrentConfig   json.RawMessage `json:"current_config"`
		ProposedConfig  json.RawMessage `json:"proposed_config"`
		Context         json.RawMessage `json:"context,omitempty"`
		Options         json.RawMessage `json:"options,omitempty"`
		Priority        int             `json:"priority"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.ServiceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "service_id is required")
	}

	// Default priority to 50 if not specified
	if req.Priority == 0 {
		req.Priority = 50
	}

	// Log what we received from the request
	c.Logger().Infof("Request ServiceID: %s", req.ServiceID)
	c.Logger().Infof("Request CurrentConfig length: %d bytes", len(req.CurrentConfig))
	c.Logger().Infof("Request ProposedConfig length: %d bytes", len(req.ProposedConfig))
	c.Logger().Infof("Request Context length: %d bytes", len(req.Context))
	c.Logger().Infof("Request Options length: %d bytes", len(req.Options))

	job, err := h.queueStore.Enqueue(ctx, simulations.CreateJobInput{
		UserID:          user.ID,
		ServiceID:       req.ServiceID,
		LLMProvider:     req.LLMProvider,
		PromptVersionID: req.PromptVersionID,
		CurrentConfig:   req.CurrentConfig,
		ProposedConfig:  req.ProposedConfig,
		Context:         req.Context,
		Options:         req.Options,
		Priority:        req.Priority,
	})

	if err != nil {
		// Log the actual error for debugging
		c.Logger().Errorf("Failed to enqueue simulation: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to enqueue simulation: %v", err))
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"job": job,
	})
}

// GetJob retrieves a specific simulation job
// GET /api/v1/simulations/queue/:id
func (h *SimulationQueueHandler) GetJob(c echo.Context) error {
	ctx := c.Request().Context()

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job ID")
	}

	job, err := h.queueStore.GetJob(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve job")
	}

	if job == nil {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"job": job,
	})
}

// ListJobs retrieves simulation jobs for the current user
// GET /api/v1/simulations/queue
func (h *SimulationQueueHandler) ListJobs(c echo.Context) error {
	ctx := c.Request().Context()

	// Get user from context
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	// Parse query parameters
	statusParam := c.QueryParam("status")
	limitParam := c.QueryParam("limit")
	offsetParam := c.QueryParam("offset")

	limit := 20
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	var status *simulations.JobStatus
	if statusParam != "" {
		s := simulations.JobStatus(statusParam)
		status = &s
	}

	jobs, total, err := h.queueStore.ListJobs(ctx, &user.ID, status, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list jobs")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": total,
	})
}

// CancelJob cancels a pending simulation job
// DELETE /api/v1/simulations/queue/:id
func (h *SimulationQueueHandler) CancelJob(c echo.Context) error {
	ctx := c.Request().Context()

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job ID")
	}

	if err := h.queueStore.CancelJob(ctx, jobID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to cancel job")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "job cancelled",
	})
}

// GetQueueStats returns queue statistics
// GET /api/v1/simulations/queue/stats
func (h *SimulationQueueHandler) GetQueueStats(c echo.Context) error {
	ctx := c.Request().Context()

	stats, err := h.queueStore.GetQueueStats(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get queue stats")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stats": stats,
	})
}

// RegisterRoutes registers all simulation queue routes
func (h *SimulationQueueHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	e.POST("/queue", h.EnqueueSimulation, authMiddleware)
	e.GET("/queue", h.ListJobs, authMiddleware)
	e.GET("/queue/stats", h.GetQueueStats, authMiddleware)
	e.GET("/queue/:id", h.GetJob, authMiddleware)
	e.DELETE("/queue/:id", h.CancelJob, authMiddleware)
}
