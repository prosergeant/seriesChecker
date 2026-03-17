-- name: GetSeriesByKinopoiskID :one
SELECT id, kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons, created_at, updated_at
FROM series
WHERE kinopoisk_id = $1;

-- name: GetSeriesByID :one
SELECT id, kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons, created_at, updated_at
FROM series
WHERE id = $1;

-- name: CreateSeries :one
INSERT INTO series (kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons, created_at, updated_at;

-- name: SearchSeries :many
SELECT id, kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons, created_at, updated_at
FROM series
WHERE title ILIKE '%' || $1 || '%'
ORDER BY title
LIMIT $2;
