import { query, queryOne } from './client';
import { ProgressResult, ProgressStatus, Series } from './types';

export async function getSeriesByKinopoiskId(kinopoiskId: number): Promise<Series | null> {
  const sql = `
    SELECT id, kinopoisk_id, title, original_title, poster_url, year, 
           description, total_episodes, total_seasons, is_serial,
           created_at, updated_at
    FROM series
    WHERE kinopoisk_id = $1
  `;
  return queryOne<Series>(sql, [kinopoiskId]);
}

export async function createSeries(
  kinopoiskId: number,
  title: string,
  originalTitle: string | null,
  posterUrl: string | null,
  year: number | null,
  description: string | null,
  totalSeasons: number | null,
  isSerial: boolean
): Promise<Series> {
  const sql = `
    INSERT INTO series (kinopoisk_id, title, original_title, poster_url, year, description, total_seasons, is_serial)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, kinopoisk_id, title, original_title, poster_url, year, description, total_episodes, total_seasons, is_serial, created_at, updated_at
  `;
  const result = await queryOne<Series>(sql, [
    kinopoiskId,
    title,
    originalTitle,
    posterUrl,
    year,
    description,
    totalSeasons,
    isSerial,
  ]);
  if (!result) throw new Error('Failed to create series');
  return result;
}

export async function getProgressList(userId: string, status?: ProgressStatus): Promise<ProgressResult[]> {
  let sql = `
    SELECT 
      up.id,
      up.user_id,
      up.series_id,
      s.kinopoisk_id,
      s.title,
      s.poster_url,
      up.current_season,
      up.current_episode,
      up.status,
      s.is_serial
    FROM user_progress up
    JOIN series s ON up.series_id = s.id
    WHERE up.user_id = $1
  `;
  
  const params: unknown[] = [userId];
  
  if (status) {
    sql += ` AND up.status = $2`;
    params.push(status);
  }
  
  sql += ` ORDER BY up.updated_at DESC`;
  
  return query<ProgressResult>(sql, params);
}

export async function getProgressBySeries(userId: string, seriesId: string) {
  const sql = `
    SELECT id, user_id, series_id, current_season, current_episode, status
    FROM user_progress
    WHERE user_id = $1 AND series_id = $2
  `;
  return queryOne(sql, [userId, seriesId]);
}

export async function createProgress(
  userId: string,
  seriesId: string,
  currentSeason: number,
  currentEpisode: number,
  status: ProgressStatus
) {
  const sql = `
    INSERT INTO user_progress (user_id, series_id, current_season, current_episode, status)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, user_id, series_id, current_season, current_episode, status
  `;
  return queryOne(sql, [userId, seriesId, currentSeason, currentEpisode, status]);
}

export async function updateProgress(
  userId: string,
  seriesId: string,
  currentSeason: number,
  currentEpisode: number,
  status: ProgressStatus
) {
  const sql = `
    UPDATE user_progress
    SET current_season = $3, current_episode = $4, status = $5, updated_at = NOW()
    WHERE user_id = $1 AND series_id = $2
    RETURNING id, user_id, series_id, current_season, current_episode, status
  `;
  return queryOne(sql, [userId, seriesId, currentSeason, currentEpisode, status]);
}

export async function deleteProgress(userId: string, seriesId: string) {
  const sql = `
    DELETE FROM user_progress
    WHERE user_id = $1 AND series_id = $2
  `;
  await query(sql, [userId, seriesId]);
}
