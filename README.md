# Inflight UI Service

Backend service for UI-specific features including quick templates, saved queries, user preferences, and future authentication.

## Overview

The UI service provides a dedicated backend for user-facing features that don't belong in domain services (advisor, simulator, metrics-collector).

## Features

- **Quick Templates**: Save and reuse configuration change presets for the Evaluation Workbench
- **Saved Queries**: Store frequently-used metric queries
- **User Preferences**: Theme, default service, UI customization
- **Future**: Authentication, team collaboration, sharing

## Quick Start

### Database Setup

Create database:
```bash
createdb ui_service
```

Or with custom user:
```bash
PGPASSWORD=yourpassword createdb -U postgres ui_service
```

### Build and Run

```bash
# Build
go build -o bin/ui-service ./cmd/ui-service

# Run with default settings (localhost:5432, dbname=ui_service)
./bin/ui-service

# Run with custom database
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=yourpass DB_NAME=ui_service ./bin/ui-service

# Run on custom port
SERVER_PORT=8083 ./bin/ui-service
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| DB_HOST | localhost | PostgreSQL host |
| DB_PORT | 5432 | PostgreSQL port |
| DB_USER | postgres | Database user |
| DB_PASSWORD | (empty) | Database password |
| DB_NAME | ui_service | Database name |
| DB_SSLMODE | disable | SSL mode (disable/require) |
| SERVER_PORT | 8083 | HTTP server port |

## API Endpoints

### Quick Templates

```http
GET    /api/v1/templates       # List templates
POST   /api/v1/templates       # Create template
GET    /api/v1/templates/:id   # Get template
PUT    /api/v1/templates/:id   # Update template
DELETE /api/v1/templates/:id   # Delete template
```

### Health

```http
GET /health  # Health check
GET /ready   # Readiness check
```

## Database Schema

### quick_templates

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL | Primary key |
| user_id | INTEGER | Owner (FK to users) |
| name | VARCHAR(255) | Template name |
| description | TEXT | Template description |
| template_data | JSONB | Proposed changes JSON |
| is_shared | BOOLEAN | Team-wide vs personal |
| created_at | TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | Last update timestamp |

## Development

### Run Migrations

Migrations run automatically on startup. To run manually:

```bash
migrate -path migrations -database "postgres://user:pass@localhost:5432/ui_service?sslmode=disable" up
```

### Run Tests

```bash
go test ./...
```

## Integration with inflight-ui

Update `next.config.js`:

```javascript
async rewrites() {
  return [
    // ... existing proxies
    {
      source: '/proxy/ui-service/:path*',
      destination: 'http://localhost:8083/:path*'
    }
  ]
}
```

## Architecture

The UI service follows the same patterns as other Inflight services:
- Echo for HTTP routing
- PostgreSQL for persistence
- golang-migrate for schema versioning
- Structured logging with logrus
- Health/readiness endpoints

## Future Enhancements

- Authentication (JWT tokens)
- User management (registration, profiles)
- Saved queries
- Custom dashboards
- Annotations (comments on simulations)
- Sharing (share results via link)
- Team permissions
