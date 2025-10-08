# Database Migrations - Inflight UI Service

## Overview

This directory contains database migrations for the Inflight UI Service. Migrations are managed using [golang-migrate](https://github.com/golang-migrate/migrate).

## Migration Files

### 000014 - Simulation Queue (Phase 1) ✅
**Files:** `000014_create_simulation_queue.{up,down}.sql`

**Purpose:** Add job queue system for asynchronous simulation processing

**Tables Created:**
- `simulation_jobs` - Queue for JVM configuration simulation jobs

**Features:**
- Job lifecycle: pending → running → completed/failed/cancelled
- Priority queue (0-100, higher = more urgent)
- Atomic job claiming with `FOR UPDATE SKIP LOCKED`
- Result storage in JSONB
- Full timing tracking (queued_at, started_at, completed_at)

**Indexes:**
- `idx_simulation_jobs_user_id` - Filter by user
- `idx_simulation_jobs_service_id` - Filter by service
- `idx_simulation_jobs_status` - Filter by status
- `idx_simulation_jobs_priority_queued` - Worker query optimization

---

### 000015 - Service Metrics Profiles (Phase 2) ✅
**Files:** `000015_create_service_metrics_profiles.{up,down}.sql`

**Purpose:** Service-specific metric requirements and validation

**Tables Created:**
- `service_metric_profiles` - Service-level metric profiles (batch, high-throughput, streaming, custom)
- `service_metric_requirements` - Granular metric requirements per service
- `metric_profile_templates` - Pre-defined profile templates for reference

**Features:**
- Profile types: batch, high_throughput, streaming, custom
- Required vs optional metrics
- Sampling rate configuration
- Max age tracking for staleness detection
- Pre-seeded with 3 profile templates

**Indexes:**
- `idx_service_metric_profiles_service_id`
- `idx_service_metric_profiles_type`
- `idx_service_metric_requirements_service_id`
- `idx_service_metric_requirements_metric_name`

---

### 000016 - Simulation Attachments (Phase 3) ✅
**Files:** `000016_create_simulation_attachments.{up,down}.sql`

**Purpose:** File attachments for simulation jobs (screenshots, configs, logs)

**Tables Created:**
- `simulation_attachments` - File metadata and storage paths

**Features:**
- Attachment types: screenshot, config, log, documentation, other
- MIME type tracking
- File size validation (max 10MB per file via constraint)
- Cascade delete when job is deleted
- Storage path for local filesystem or S3

**Indexes:**
- `idx_simulation_attachments_job_id` - List attachments by job
- `idx_simulation_attachments_user_id` - Filter by user
- `idx_simulation_attachments_type` - Filter by type
- `idx_simulation_attachments_uploaded_at` - Sort by upload time

**Constraints:**
- `valid_attachment_type` - Ensures type is one of 5 allowed values
- `valid_file_size` - Max 10 MB per file (10485760 bytes)

---

## Running Migrations

### Automatic (Recommended)
Migrations run automatically on service startup if configured in `config/service.yaml`:

```yaml
migrations:
  auto_run: true
  path: ./migrations
```

### Manual
```bash
# Up (apply all pending)
migrate -path ./migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" up

# Down (rollback last migration)
migrate -path ./migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" down 1

# Force version (if migrations are out of sync)
migrate -path ./migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" force <version>
```

### Using setup_database.sh
```bash
./setup_database.sh
```

---

## Migration Naming Convention

Format: `<version>_<description>.{up,down}.sql`

Examples:
- `000014_create_simulation_queue.up.sql`
- `000014_create_simulation_queue.down.sql`

**Rules:**
- Use 6-digit zero-padded version numbers
- Use snake_case for descriptions
- Always create both `.up.sql` and `.down.sql`
- Test rollback before committing

---

## Database Schema Summary

### Core Tables
- `users` - User accounts (from previous migrations)
- `sessions` - Redis-backed sessions (from previous migrations)
- `roles`, `permissions`, `user_roles` - RBAC system

### Workbench V2 Tables (NEW)
- `simulation_jobs` - Job queue
- `simulation_attachments` - File uploads
- `service_metric_profiles` - Metric profiles
- `service_metric_requirements` - Metric requirements
- `metric_profile_templates` - Pre-defined templates

### Relationships
```
users (1) ────< (N) simulation_jobs (1) ────< (N) simulation_attachments
                          │
                          └─ service_id → services (in metrics-collector)

users (1) ────< (N) service_metric_profiles
                          │
                          └─ service_id → services

service_metric_requirements ──┐
                              ├─ service_id → services
                              └─ canonical_metric_name → canonical_metrics (in metrics-collector)
```

---

## Storage Requirements

### Disk Space Estimates

**simulation_jobs table:**
- ~2 KB per job (config + result JSONB)
- 1,000 jobs = ~2 MB
- 10,000 jobs = ~20 MB
- 100,000 jobs = ~200 MB

**simulation_attachments table:**
- Metadata: ~500 bytes per attachment
- Files: Average ~2 MB per attachment
- 1,000 attachments = ~2 GB (disk)

**Recommendations:**
- Archive jobs > 30 days to `simulation_jobs_archive` table
- Delete attachments when archiving jobs
- Monitor `/var/inflight/uploads` disk usage
- Set up log rotation for uploads directory

---

## Indexes and Performance

### Critical Indexes for Worker Performance

**Most Important:**
```sql
CREATE INDEX idx_simulation_jobs_priority_queued
ON simulation_jobs(priority DESC, queued_at ASC)
WHERE status = 'pending';
```

This partial index is used by the worker to efficiently find the next job:
```sql
SELECT id FROM simulation_jobs
WHERE status = 'pending'
ORDER BY priority DESC, queued_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
```

**Query Performance:**
- Without index: O(n) table scan
- With index: O(1) index lookup
- ~10,000 jobs in queue: < 1ms query time

### Secondary Indexes

For UI filtering and user queries:
- `idx_simulation_jobs_user_id` - User's job list
- `idx_simulation_jobs_service_id` - Service-specific jobs
- `idx_simulation_jobs_status` - Filter by status

---

## Data Retention Policy

### Recommended:
```sql
-- Archive completed jobs older than 30 days
INSERT INTO simulation_jobs_archive
SELECT * FROM simulation_jobs
WHERE status IN ('completed', 'failed', 'cancelled')
  AND completed_at < NOW() - INTERVAL '30 days';

DELETE FROM simulation_jobs
WHERE status IN ('completed', 'failed', 'cancelled')
  AND completed_at < NOW() - INTERVAL '30 days';

-- Delete orphaned attachments (if any)
DELETE FROM simulation_attachments
WHERE simulation_job_id NOT IN (SELECT id FROM simulation_jobs);
```

Run monthly via cron job.

---

## Troubleshooting

### Migration Failed
**Symptoms:** Service won't start, migration error in logs

**Solutions:**
1. Check PostgreSQL connection
2. Check migration version:
   ```sql
   SELECT * FROM schema_migrations;
   ```
3. Force to correct version if out of sync:
   ```bash
   migrate -path ./migrations -database "$DSN" force 16
   ```

### Duplicate Key Violations
**Symptoms:** `ERROR: duplicate key value violates unique constraint`

**Causes:**
- Re-running migration that already succeeded
- Data conflicts

**Solutions:**
1. Check if table already exists:
   ```sql
   \dt simulation_jobs
   ```
2. If exists, force migration version to match
3. If data conflict, resolve manually then retry

### Index Build Timeout
**Symptoms:** CREATE INDEX takes > 5 minutes

**Causes:**
- Large existing dataset
- Locks on table

**Solutions:**
1. Use `CREATE INDEX CONCURRENTLY`:
   ```sql
   CREATE INDEX CONCURRENTLY idx_name ON table(column);
   ```
2. Run during low-traffic window
3. Check for blocking queries:
   ```sql
   SELECT * FROM pg_stat_activity WHERE state = 'active';
   ```

---

## Future Migrations

### Planned (Phase 8+):
- **000017**: Simulation comparison tables
- **000018**: Scheduled simulation jobs
- **000019**: Simulation templates (full config save)
- **000020**: Job sharing and collaboration

---

## Best Practices

### When Creating Migrations:

1. **Test locally first**
   ```bash
   # Test up
   migrate -path ./migrations -database "postgresql://simulator:simulator@localhost:5432/ui_service?sslmode=disable" up

   # Test down (rollback)
   migrate -path ./migrations -database "postgresql://simulator:simulator@localhost:5432/ui_service?sslmode=disable" down 1

   # Test up again
   migrate -path ./migrations -database "postgresql://simulator:simulator@localhost:5432/ui_service?sslmode=disable" up
   ```

2. **Always create .down.sql**
   - Must be reversible
   - Drop objects in reverse order
   - Handle data migration in rollback

3. **Use IF NOT EXISTS**
   - Idempotent migrations are safer
   - Allows re-running if partially applied

4. **Add comments**
   - Use `COMMENT ON TABLE` and `COMMENT ON COLUMN`
   - Helps future developers understand schema

5. **Consider data migration**
   - If altering existing tables with data
   - Write data migration logic
   - Test with production-sized dataset

6. **Check dependencies**
   - Foreign keys must reference existing tables
   - Ensure migration order is correct

---

## Monitoring

### Query Performance
```sql
-- Check slow queries on simulation_jobs
SELECT
  query,
  calls,
  mean_exec_time,
  max_exec_time
FROM pg_stat_statements
WHERE query LIKE '%simulation_jobs%'
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Table Sizes
```sql
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'simulation%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Index Usage
```sql
SELECT
  schemaname,
  tablename,
  indexname,
  idx_scan,
  idx_tup_read,
  idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename LIKE 'simulation%'
ORDER BY idx_scan DESC;
```

---

## Contact

For migration issues or questions:
- Check logs: `docker logs inflight-ui-service`
- Review migration code in this directory
- Check database status: `\l` and `\d simulation_jobs` in psql
