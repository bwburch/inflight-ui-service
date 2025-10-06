#!/bin/bash

# Database setup script for inflight-metrics-collector
# This script creates the database and runs migrations

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-simulator}"
DB_PASSWORD="${DB_PASSWORD:-simulator}"
DB_NAME="${DB_NAME:-ui_service}"
MIGRATE_PATH="${MIGRATE_PATH:-./migrations}"

echo "🔧 Setting up database for inflight-metrics-collector..."

# Export password for psql
export PGPASSWORD=$DB_PASSWORD

# Check if database exists
DB_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'")

if [ "$DB_EXISTS" != "1" ]; then
    echo "📦 Creating database: $DB_NAME"
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
    echo "✅ Database created successfully"
else
    echo "ℹ️  Database $DB_NAME already exists"
fi

# Check if migrate is installed
if ! command -v migrate &> /dev/null; then
    echo "❌ migrate command not found. Please install golang-migrate:"
    echo "   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0"
    exit 1
fi

# Run migrations
echo "🔄 Running migrations..."
DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

migrate -path $MIGRATE_PATH -database "$DATABASE_URL" up

echo "✅ Database setup complete!"
echo ""
echo "Connection details:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"