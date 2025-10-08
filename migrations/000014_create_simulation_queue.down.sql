-- Migration rollback: Remove simulation job queue

DROP INDEX IF EXISTS idx_simulation_jobs_priority_queued;
DROP INDEX IF EXISTS idx_simulation_jobs_queued_at;
DROP INDEX IF EXISTS idx_simulation_jobs_status;
DROP INDEX IF EXISTS idx_simulation_jobs_service_id;
DROP INDEX IF EXISTS idx_simulation_jobs_user_id;

DROP TABLE IF EXISTS simulation_jobs;
