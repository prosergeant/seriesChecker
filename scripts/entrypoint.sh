#!/bin/sh
set -e

echo "=== Container starting ==="
echo "Current directory: $(pwd)"
echo "Files in /app: $(ls -la /app 2>/dev/null || echo 'not found')"

shutdown() {
    echo "=== Received shutdown signal ==="
    if [ -n "$NEXT_PID" ]; then
        echo "Stopping Next.js (PID: $NEXT_PID)..."
        kill -TERM $NEXT_PID 2>/dev/null || true
    fi
    if [ -n "$SERVER_PID" ]; then
        echo "Stopping Go server (PID: $SERVER_PID)..."
        kill -TERM $SERVER_PID 2>/dev/null || true
    fi
    echo "=== Shutdown complete ==="
    exit 0
}

trap shutdown TERM INT

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
for migration in /app/migrations/*.sql; do
    if [ -f "$migration" ]; then
        migration_name=$(basename "$migration")
        applied=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM schema_migrations WHERE name = '$migration_name';" 2>/dev/null | tr -d ' ')
        if [ "$applied" = "0" ]; then
            echo "Applying migration: $migration_name"
            PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration"
        else
            echo "Migration $migration_name already applied"
        fi
    fi
done

echo "Checking Next.js standalone files..."
if [ -f "server.js" ]; then
    echo "server.js found at /app/server.js"
else
    echo "ERROR: server.js NOT found!"
    ls -la /app/
    exit 1
fi

echo "Starting Next.js server on port 3000..."
PORT=3000 HOSTNAME=localhost node server.js > /tmp/nextjs.log 2>&1 &
NEXT_PID=$!
echo "Next.js started with PID: $NEXT_PID"

echo "Waiting for Next.js to be ready..."
for i in $(seq 1 30); do
    if grep -q "Ready" /tmp/nextjs.log 2>/dev/null; then
        echo "Next.js is ready!"
        break
    fi
    echo "Waiting... ($i/30)"
    sleep 1
done

if ! kill -0 $NEXT_PID 2>/dev/null; then
    echo "ERROR: Next.js failed to start!"
    echo "Next.js log:"
    cat /tmp/nextjs.log
    exit 1
fi

echo "Starting Go server on port 8080..."
/app/server > /tmp/go.log 2>&1 &
SERVER_PID=$!
echo "Go server started with PID: $SERVER_PID"

sleep 3

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "ERROR: Go server failed to start!"
    echo "Go log:"
    cat /tmp/go.log
    kill $NEXT_PID 2>/dev/null || true
    exit 1
fi

echo "=== All services started ==="
echo "Next.js: PID $NEXT_PID, Log: /tmp/nextjs.log"
echo "Go: PID $SERVER_PID, Log: /tmp/go.log"

# Monitor processes
while true; do
    sleep 10
    if ! kill -0 $NEXT_PID 2>/dev/null; then
        echo "WARNING: Next.js died!"
        cat /tmp/nextjs.log
    fi
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "WARNING: Go server died!"
        cat /tmp/go.log
        kill $NEXT_PID 2>/dev/null || true
        exit 1
    fi
done
