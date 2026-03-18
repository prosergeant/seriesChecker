#!/bin/sh
set -e

echo "Waiting for PostgreSQL..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
        echo "PostgreSQL is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "Attempt $attempt/$max_attempts - waiting..."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "Failed to connect to PostgreSQL"
    exit 1
fi

echo "Running migrations..."

run_migration() {
    migration_file="$1"
    migration_name=$(basename "$migration_file")
    
    applied=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM schema_migrations WHERE name = '$migration_name';" 2>/dev/null | tr -d ' ')
    
    if [ "$applied" = "0" ]; then
        echo "Applying migration: $migration_name"
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration_file"
        echo "Migration $migration_name applied successfully"
    else
        echo "Migration $migration_name already applied, skipping"
    fi
}

for migration in /app/migrations/*.sql; do
    if [ -f "$migration" ]; then
        run_migration "$migration"
    fi
done

echo "Starting server..."
exec /app/server
