package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwburch/inflight-ui-service/internal/storage/simulations"
	"github.com/sirupsen/logrus"
)

// SimulationWorker processes simulation jobs from the queue
type SimulationWorker struct {
	queueStore   *simulations.JobQueueStore
	advisorURL   string
	pollInterval time.Duration
	logger       *logrus.Logger
	stopChan     chan struct{}
}

// NewSimulationWorker creates a new simulation worker
func NewSimulationWorker(queueStore *simulations.JobQueueStore, advisorURL string, logger *logrus.Logger) *SimulationWorker {
	return &SimulationWorker{
		queueStore:   queueStore,
		advisorURL:   advisorURL,
		pollInterval: 5 * time.Second, // Check for jobs every 5 seconds
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// Start begins processing jobs from the queue
func (w *SimulationWorker) Start(ctx context.Context) {
	w.logger.Info("Starting simulation worker...")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Simulation worker stopped (context cancelled)")
			return
		case <-w.stopChan:
			w.logger.Info("Simulation worker stopped")
			return
		case <-ticker.C:
			w.processNextJob(ctx)
		}
	}
}

// Stop gracefully stops the worker
func (w *SimulationWorker) Stop() {
	close(w.stopChan)
}

// processNextJob picks up and processes the next pending job
func (w *SimulationWorker) processNextJob(ctx context.Context) {
	// Get next job from queue (atomically marks as running)
	job, err := w.queueStore.GetNextPendingJob(ctx)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get next job")
		return
	}

	if job == nil {
		// No pending jobs, that's fine
		return
	}

	w.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"service_id": job.ServiceID,
		"user_id":    job.UserID,
	}).Info("Processing simulation job")

	// Execute the simulation
	if err := w.executeSimulation(ctx, job); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Error("Simulation failed")
		w.queueStore.MarkFailed(ctx, job.ID, err.Error())
		return
	}

	w.logger.WithField("job_id", job.ID).Info("Simulation completed successfully")
}

// executeSimulation calls the Advisor service to run the simulation
func (w *SimulationWorker) executeSimulation(ctx context.Context, job *simulations.SimulationJob) error {
	// Build request payload for Advisor
	payload := map[string]interface{}{
		"service_id":      job.ServiceID,
		"current_config":  json.RawMessage(job.CurrentConfig),
		"proposed_config": json.RawMessage(job.ProposedConfig),
	}

	if job.LLMProvider != nil {
		payload["llm_provider"] = *job.LLMProvider
	}
	if job.PromptVersionID != nil {
		payload["prompt_version_id"] = *job.PromptVersionID
	}
	if job.Context != nil {
		payload["context"] = json.RawMessage(job.Context)
	}
	if job.Options != nil {
		payload["options"] = json.RawMessage(job.Options)
	}

	// Marshal to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Call Advisor /evaluate endpoint
	url := fmt.Sprintf("%s/api/v1/evaluate", w.advisorURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute, // Simulations can take a while
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("advisor returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Store result in job
	if err := w.queueStore.MarkCompleted(ctx, job.ID, responseBody); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	return nil
}
