package simulations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// JobStatus represents the state of a simulation job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// SimulationJob represents a queued simulation job
type SimulationJob struct {
	ID          int             `db:"id" json:"id"`
	UserID      int             `db:"user_id" json:"user_id"`
	ServiceID   string          `db:"service_id" json:"service_id"`

	// Configuration
	LLMProvider     *string         `db:"llm_provider" json:"llm_provider,omitempty"`
	PromptVersionID *int            `db:"prompt_version_id" json:"prompt_version_id,omitempty"`
	CurrentConfig   json.RawMessage `db:"current_config" json:"current_config"`
	ProposedConfig  json.RawMessage `db:"proposed_config" json:"proposed_config"`
	Context         json.RawMessage `db:"context" json:"context,omitempty"`
	Options         json.RawMessage `db:"options" json:"options,omitempty"`

	// Status
	Status   JobStatus `db:"status" json:"status"`
	Priority int       `db:"priority" json:"priority"`

	// Results (nullable)
	Result       *json.RawMessage `db:"result" json:"result,omitempty"`
	ErrorMessage *string          `db:"error_message" json:"error_message,omitempty"`

	// Timing
	QueuedAt    time.Time  `db:"queued_at" json:"queued_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at" json:"updated_at,omitempty"`
}

// CreateJobInput represents input for creating a new simulation job
type CreateJobInput struct {
	UserID          int
	ServiceID       string
	LLMProvider     *string
	PromptVersionID *int
	CurrentConfig   json.RawMessage
	ProposedConfig  json.RawMessage
	Context         json.RawMessage
	Options         json.RawMessage
	Priority        int
}

// JobQueueStore handles database operations for simulation jobs
type JobQueueStore struct {
	db *sql.DB
}

// NewJobQueueStore creates a new job queue store
func NewJobQueueStore(db *sql.DB) *JobQueueStore {
	return &JobQueueStore{db: db}
}

// Enqueue creates a new simulation job in pending status
func (s *JobQueueStore) Enqueue(ctx context.Context, input CreateJobInput) (*SimulationJob, error) {
	// Validate JSON fields are not nil (PostgreSQL JSONB doesn't accept NULL for non-nullable columns)
	if input.CurrentConfig == nil {
		return nil, fmt.Errorf("current_config cannot be nil")
	}
	if input.ProposedConfig == nil {
		return nil, fmt.Errorf("proposed_config cannot be nil")
	}

	// Log the JSON being inserted for debugging
	fmt.Printf("[Enqueue] CurrentConfig: %s\n", string(input.CurrentConfig))
	fmt.Printf("[Enqueue] ProposedConfig: %s\n", string(input.ProposedConfig))
	if input.Context != nil {
		fmt.Printf("[Enqueue] Context: %s\n", string(input.Context))
	}
	if input.Options != nil {
		fmt.Printf("[Enqueue] Options: %s\n", string(input.Options))
	}

	query := `
		INSERT INTO simulation_jobs (
			user_id, service_id, llm_provider, prompt_version_id,
			current_config, proposed_config, context, options, priority, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending')
		RETURNING id, user_id, service_id, llm_provider, prompt_version_id,
		          current_config, proposed_config, context, options, status, priority,
		          result, error_message, queued_at, started_at, completed_at, created_at, updated_at
	`

	var job SimulationJob
	err := s.db.QueryRowContext(ctx, query,
		input.UserID, input.ServiceID, input.LLMProvider, input.PromptVersionID,
		input.CurrentConfig, input.ProposedConfig, input.Context, input.Options, input.Priority,
	).Scan(
		&job.ID, &job.UserID, &job.ServiceID, &job.LLMProvider, &job.PromptVersionID,
		&job.CurrentConfig, &job.ProposedConfig, &job.Context, &job.Options, &job.Status, &job.Priority,
		&job.Result, &job.ErrorMessage, &job.QueuedAt, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	return &job, nil
}

// GetNextPendingJob retrieves the next pending job by priority and queue time
func (s *JobQueueStore) GetNextPendingJob(ctx context.Context) (*SimulationJob, error) {
	query := `
		UPDATE simulation_jobs
		SET status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM simulation_jobs
			WHERE status = 'pending'
			ORDER BY priority DESC, queued_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, user_id, service_id, llm_provider, prompt_version_id,
		          current_config, proposed_config, context, options, status, priority,
		          result, error_message, queued_at, started_at, completed_at, created_at, updated_at
	`

	var job SimulationJob
	err := s.db.QueryRowContext(ctx, query).Scan(
		&job.ID, &job.UserID, &job.ServiceID, &job.LLMProvider, &job.PromptVersionID,
		&job.CurrentConfig, &job.ProposedConfig, &job.Context, &job.Options, &job.Status, &job.Priority,
		&job.Result, &job.ErrorMessage, &job.QueuedAt, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No pending jobs
	}
	if err != nil {
		return nil, fmt.Errorf("get next job: %w", err)
	}

	return &job, nil
}

// MarkCompleted marks a job as completed with results
func (s *JobQueueStore) MarkCompleted(ctx context.Context, jobID int, result json.RawMessage) error {
	query := `
		UPDATE simulation_jobs
		SET status = 'completed', result = $2, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, jobID, result)
	if err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	return nil
}

// MarkFailed marks a job as failed with error message
func (s *JobQueueStore) MarkFailed(ctx context.Context, jobID int, errorMsg string) error {
	query := `
		UPDATE simulation_jobs
		SET status = 'failed', error_message = $2, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, jobID, errorMsg)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	return nil
}

// CancelJob cancels a pending job
func (s *JobQueueStore) CancelJob(ctx context.Context, jobID int) error {
	query := `
		UPDATE simulation_jobs
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`

	result, err := s.db.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job not found or not in pending status")
	}

	return nil
}

// GetJob retrieves a specific job by ID
func (s *JobQueueStore) GetJob(ctx context.Context, jobID int) (*SimulationJob, error) {
	query := `
		SELECT id, user_id, service_id, llm_provider, prompt_version_id,
		       current_config, proposed_config, context, options, status, priority,
		       result, error_message, queued_at, started_at, completed_at, created_at, updated_at
		FROM simulation_jobs
		WHERE id = $1
	`

	var job SimulationJob
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.UserID, &job.ServiceID, &job.LLMProvider, &job.PromptVersionID,
		&job.CurrentConfig, &job.ProposedConfig, &job.Context, &job.Options, &job.Status, &job.Priority,
		&job.Result, &job.ErrorMessage, &job.QueuedAt, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	return &job, nil
}

// ListJobs retrieves jobs with optional filters
func (s *JobQueueStore) ListJobs(ctx context.Context, userID *int, status *JobStatus, limit, offset int) ([]SimulationJob, int, error) {
	// Build query with filters
	query := `
		SELECT id, user_id, service_id, llm_provider, prompt_version_id,
		       current_config, proposed_config, context, options, status, priority,
		       result, error_message, queued_at, started_at, completed_at, created_at, updated_at
		FROM simulation_jobs
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM simulation_jobs WHERE 1=1`
	args := []interface{}{}
	countArgs := []interface{}{}
	argNum := 1

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argNum)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argNum)
		args = append(args, *userID)
		countArgs = append(countArgs, *userID)
		argNum++
	}

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		countQuery += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *status)
		countArgs = append(countArgs, *status)
		argNum++
	}

	// Order by priority (high first) then queue time (old first)
	query += " ORDER BY priority DESC, queued_at ASC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	// Get total count
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// Get jobs
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SimulationJob
	for rows.Next() {
		var job SimulationJob
		err := rows.Scan(
			&job.ID, &job.UserID, &job.ServiceID, &job.LLMProvider, &job.PromptVersionID,
			&job.CurrentConfig, &job.ProposedConfig, &job.Context, &job.Options, &job.Status, &job.Priority,
			&job.Result, &job.ErrorMessage, &job.QueuedAt, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// GetQueueStats returns statistics about the job queue
func (s *JobQueueStore) GetQueueStats(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM simulation_jobs
		GROUP BY status
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get queue stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}
