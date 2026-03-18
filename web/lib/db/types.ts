export interface Series {
  id: string;
  kinopoisk_id: number;
  title: string;
  original_title: string | null;
  poster_url: string | null;
  year: number | null;
  description: string | null;
  total_episodes: number | null;
  total_seasons: number | null;
  is_serial: boolean;
  created_at: Date;
  updated_at: Date;
}

export interface UserProgress {
  id: string;
  user_id: string;
  series_id: string;
  current_season: number;
  current_episode: number;
  status: 'watching' | 'completed' | 'planned' | 'dropped' | 'on_hold';
  created_at: Date;
  updated_at: Date;
}

export interface ProgressWithSeries extends UserProgress {
  kinopoisk_id: number;
  series_title: string;
  poster_url: string | null;
  is_serial: boolean;
}

export type ProgressStatus = 'watching' | 'completed' | 'planned' | 'dropped' | 'on_hold';

export interface ProgressResult {
  id: number;
  user_id: string;
  series_id: number;
  kinopoisk_id: number;
  title: string;
  poster_url: string | null;
  current_season: number;
  current_episode: number;
  status: string;
  is_serial: boolean;
}
