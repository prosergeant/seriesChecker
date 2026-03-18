const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = `${API_URL}${endpoint}`;
  
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(response.status, error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  auth: {
    register: (email: string, password: string) =>
      request<{ id: string; email: string }>('/api/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    
    login: (email: string, password: string) =>
      request<{ message: string; session_id: string }>('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    
    logout: () =>
      request('/api/auth/logout', { method: 'POST' }),
    
    me: () => request<{ id: string; email: string }>('/api/auth/me'),
  },

  series: {
    search: (query: string) =>
      request<SeriesSearchResult[]>(`/api/series/search?q=${encodeURIComponent(query)}`),
    
    getById: (id: number) =>
      request<SeriesDetails>(`/api/series/${id}`),
  },

  progress: {
    getAll: (status?: string) =>
      request<ProgressItem[]>(
        `/api/progress${status ? `?status=${status}` : ''}`
      ),
    
    update: (data: UpdateProgressRequest) =>
      request<ProgressItem>('/api/progress', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    delete: (seriesId: number) =>
      request<{ message: string }>(`/api/progress/${seriesId}`, {
        method: 'DELETE',
      }),
  },
};

export interface SeriesSearchResult {
  id: number;
  kinopoisk_id: number;
  title: string;
  original_title?: string;
  poster_url?: string;
  year?: number;
}

export interface SeriesDetails extends SeriesSearchResult {
  description?: string;
  total_episodes?: number;
  total_seasons?: number;
}

export interface ProgressItem {
  id: number;
  user_id: string;
  series_id: number;
  kinopoisk_id: number;
  title: string;
  poster_url?: string;
  current_season: number;
  current_episode: number;
  status: string;
  is_serial: boolean
}

export interface UpdateProgressRequest {
  series_id: number;
  current_season: number;
  current_episode: number;
  status: string;
}
