-- migrations/001_initial_schema.sql

-- users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(60) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- series table (cached data from Kinopoisk)
CREATE TABLE IF NOT EXISTS series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kinopoisk_id INTEGER UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    original_title VARCHAR(500),
    poster_url TEXT,
    year INTEGER,
    description TEXT,
    total_episodes INTEGER,
    total_seasons INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_series_kinopoisk_id ON series(kinopoisk_id);
CREATE INDEX IF NOT EXISTS idx_series_title ON series(title);

-- user_progress table
CREATE TABLE IF NOT EXISTS user_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    current_season INTEGER NOT NULL DEFAULT 1,
    current_episode INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'watching',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_series UNIQUE (user_id, series_id)
);

CREATE INDEX IF NOT EXISTS idx_user_progress_user_status ON user_progress(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_progress_user ON user_progress(user_id);
