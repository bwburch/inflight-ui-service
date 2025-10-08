-- Migration: Create simulation attachments
-- Description: File attachments for simulation jobs (screenshots, configs, logs)

-- ============================================================================
-- Simulation Attachments Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS simulation_attachments (
    id SERIAL PRIMARY KEY,
    simulation_job_id INTEGER NOT NULL REFERENCES simulation_jobs(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),

    -- File metadata
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(100), -- MIME type: 'image/png', 'text/plain', 'application/json', etc.
    file_size INTEGER NOT NULL, -- bytes
    storage_path TEXT NOT NULL, -- Relative path from uploads directory or S3 key

    -- Classification
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'other', -- 'screenshot', 'config', 'log', 'documentation', 'other'
    description TEXT,

    -- Timestamps
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_attachment_type CHECK (attachment_type IN ('screenshot', 'config', 'log', 'documentation', 'other')),
    CONSTRAINT valid_file_size CHECK (file_size > 0 AND file_size <= 10485760) -- Max 10MB per file
);

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX idx_simulation_attachments_job_id ON simulation_attachments(simulation_job_id);
CREATE INDEX idx_simulation_attachments_user_id ON simulation_attachments(user_id);
CREATE INDEX idx_simulation_attachments_type ON simulation_attachments(attachment_type);
CREATE INDEX idx_simulation_attachments_uploaded_at ON simulation_attachments(uploaded_at DESC);

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE simulation_attachments IS 'File attachments for simulation jobs (screenshots, configs, logs, documentation)';
COMMENT ON COLUMN simulation_attachments.storage_path IS 'Relative path from uploads directory (local) or S3 key (cloud)';
COMMENT ON COLUMN simulation_attachments.attachment_type IS 'Category of attachment for UI organization';
COMMENT ON COLUMN simulation_attachments.file_size IS 'File size in bytes (max 10MB enforced by constraint)';
