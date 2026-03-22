#!/usr/bin/env bash
set -euo pipefail

LOCAL_DSN="postgresql://seriestracker:seriestracker@localhost:5432/seriestracker"
BACKUP_FILE="neon_backup_$(date +%Y%m%d_%H%M%S).sql"

echo "→ Pulling DATABASE_URL from Vercel..."
vercel env pull .env.neon --yes 2>/dev/null
NEON_URL=$(grep '^DATABASE_URL=' .env.neon | cut -d'=' -f2- | tr -d '"')
rm .env.neon

if [ -z "$NEON_URL" ]; then
  echo "✗ DATABASE_URL not found. Run 'vercel link' first."
  exit 1
fi

echo "→ Dumping from Neon → $BACKUP_FILE..."
docker run --rm postgres:17 pg_dump "$NEON_URL" --no-owner --no-acl -F p > "$BACKUP_FILE"

echo "→ Ensuring local postgres is running..."
docker compose up -d postgres
sleep 2

echo "→ Recreating local database..."
docker compose exec -T postgres psql -U seriestracker postgres \
  -c "DROP DATABASE IF EXISTS seriestracker;" \
  -c "CREATE DATABASE seriestracker;"

echo "→ Restoring to local postgres..."
psql "$LOCAL_DSN" -f "$BACKUP_FILE"

echo "✓ Done. Backup saved to $BACKUP_FILE"
