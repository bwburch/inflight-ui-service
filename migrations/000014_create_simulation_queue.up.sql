-- Migration: Create simulation job queue
-- Description: Adds tables for queueing and tracking simulation jobs

-- ============================================================================
-- Simulation Jobs Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS simulation_jobs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    service_id VARCHAR(255) NOT NULL,

    -- Job Configuration
    llm_provider VARCHAR(100),
    prompt_version_id INTEGER,
    current_config JSONB NOT NULL,
    proposed_config JSONB NOT NULL,
    context JSONB,
    options JSONB,

    -- Job Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    priority INTEGER DEFAULT 50, -- 0-100, higher = more urgent

    -- Results
    result JSONB, -- Stores the complete advisor response
    error_message TEXT,

    -- Timing
    queued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,

    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT valid_priority CHECK (priority BETWEEN 0 AND 100)
);

-- Indexes for performance
CREATE INDEX idx_simulation_jobs_user_id ON simulation_jobs(user_id);
CREATE INDEX idx_simulation_jobs_service_id ON simulation_jobs(service_id);
CREATE INDEX idx_simulation_jobs_status ON simulation_jobs(status);
CREATE INDEX idx_simulation_jobs_queued_at ON simulation_jobs(queued_at DESC);
CREATE INDEX idx_simulation_jobs_priority_queued ON simulation_jobs(priority DESC, queued_at ASC) WHERE status = 'pending';

-- Comments
COMMENT ON TABLE simulation_jobs IS 'Queue for JVM configuration simulation jobs';
COMMENT ON COLUMN simulation_jobs.status IS 'Job lifecycle: pending -> running -> completed/failed/cancelled';
COMMENT ON COLUMN simulation_jobs.priority IS 'Job priority (0-100), higher values processed first';
