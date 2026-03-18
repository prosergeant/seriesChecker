-- migrations/000_create_schema_migrations.sql

CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Mark this migration as applied
INSERT INTO schema_migrations (name) VALUES ('000_create_schema_migrations.sql')
ON CONFLICT (name) DO NOTHING;
