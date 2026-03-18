-- name: GetUserProgress :one
SELECT id, user_id, series_id, current_season, current_episode, status, created_at, updated_at
FROM user_progress
WHERE user_id = $1 AND series_id = $2;

-- name: GetUserProgressList :many
SELECT up.id, up.user_id, up.series_id, up.current_season, up.current_episode, up.status, up.created_at, up.updated_at,
       s.id as s_id, s.kinopoisk_id, s.title, s.original_title, s.poster_url, s.year, s.description, s.total_episodes, s.total_seasons, s.is_serial
FROM user_progress up
JOIN series s ON up.series_id = s.id
WHERE up.user_id = $1 AND up.status = $2
ORDER BY up.updated_at DESC;

-- name: GetAllUserProgress :many
SELECT up.id, up.user_id, up.series_id, up.current_season, up.current_episode, up.status, up.created_at, up.updated_at,
       s.id as s_id, s.kinopoisk_id, s.title, s.original_title, s.poster_url, s.year, s.description, s.total_episodes, s.total_seasons, s.is_serial
FROM user_progress up
JOIN series s ON up.series_id = s.id
WHERE up.user_id = $1
ORDER BY up.updated_at DESC;

-- name: CreateUserProgress :one
INSERT INTO user_progress (user_id, series_id, current_season, current_episode, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, series_id, current_season, current_episode, status, created_at, updated_at;

-- name: UpdateUserProgress :one
UPDATE user_progress
SET current_season = $3, current_episode = $4, status = $5, updated_at = NOW()
WHERE user_id = $1 AND series_id = $2
RETURNING id, user_id, series_id, current_season, current_episode, status, created_at, updated_at;

-- name: DeleteUserProgress :exec
DELETE FROM user_progress
WHERE user_id = $1 AND series_id = $2;
