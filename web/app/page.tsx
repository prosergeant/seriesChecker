'use client';

import { useState, useTransition } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { api, ProgressItem, SeriesSearchResult, UpdateProgressRequest } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Search, Plus, LogOut } from 'lucide-react';
import { toast } from 'sonner';
import { useAuth } from '@/components/auth-context';
import { ProtectedRoute } from '@/components/protected-route';
import { addProgress, updateUserProgress, removeProgress } from '@/lib/actions/progress';

const STATUS_LABELS: Record<string, { label: string }> = {
  watching: { label: 'Смотрю' },
  completed: { label: 'Просмотрено' },
  planned: { label: 'Запланировано' },
  dropped: { label: 'Брошено' },
  on_hold: { label: 'На паузе' },
};

function SearchResults({ query, onSelect }: { query: string; onSelect: (series: SeriesSearchResult) => void }) {
  const { data, isLoading } = useQuery({
    queryKey: ['search', query],
    queryFn: () => api.series.search(query),
    enabled: query.length >= 2,
  });

  if (!query || query.length < 2) return null;
  if (isLoading) return <div className="text-center py-4">Загрузка...</div>;
  if (!data?.length) return <div className="text-center py-4">Ничего не найдено</div>;

  return (
    <Card className="absolute top-full left-0 right-0 z-50 mt-2 max-h-80 overflow-auto">
      <CardContent className="p-2">
        {data.map((series) => (
          <button
            key={series.id}
            onClick={() => onSelect(series)}
            className="w-full flex items-center gap-3 p-2 hover:bg-muted rounded-lg text-left"
          >
            {series.poster_url && (
              <img src={series.poster_url} alt={series.title} className="w-10 h-14 object-cover rounded" />
            )}
            <div>
              <div className="font-medium">{series.title}</div>
              {series.year && <div className="text-sm text-muted-foreground">{series.year}</div>}
            </div>
          </button>
        ))}
      </CardContent>
    </Card>
  );
}

function ProgressCard({ 
  item, 
  onUpdate, 
  onDelete,
  isPending 
}: { 
  item: ProgressItem; 
  onUpdate: (data: UpdateProgressRequest) => void;
  onDelete: () => void;
  isPending?: boolean;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [season, setSeason] = useState(item.current_season);
  const [episode, setEpisode] = useState(item.current_episode);

  const statusInfo = STATUS_LABELS[item.status] || { label: item.status, icon: null };

  const handleSave = () => {
    onUpdate({ series_id: item.series_id, current_season: season, current_episode: episode, status: item.status });
    setIsEditing(false);
  };

  const goToPreview = () => {
    window.open(`https://www.sspoisk.ru/${item.is_serial ? 'series' : 'film'}/${item.kinopoisk_id}`, '_blank')
  }

  return (
    <div className={`bg-white rounded-2xl flex flex-row overflow-hidden border border-gray-200 shadow-sm hover:shadow-md transition-shadow ${isPending ? 'opacity-50' : ''}`}>
      {item.poster_url && (
        <div className="w-[100px] md:w-[120px] flex-shrink-0">
          <img 
            src={item.poster_url} 
            alt={item.title} 
            className="w-full h-full object-cover"
            style={{ minHeight: '100%' }}
          />
        </div>
      )}
      <div className="flex-1 p-4 sm:p-5 flex flex-col gap-3 min-w-0">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <h3 className="text-lg sm:text-xl font-semibold text-gray-800 tracking-tight truncate pr-2">
            {item.title}
          </h3>
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800 border border-emerald-200 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5"></span>
            {statusInfo.label}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {isEditing ? (
            <>
              <input
                type="number"
                value={season}
                onChange={(e) => setSeason(Math.max(1, parseInt(e.target.value) || 1))}
                className="w-12 h-8 text-center border rounded-lg text-sm"
                min={1}
              />
              <span className="text-sm text-gray-600">сезон</span>
              <input
                type="number"
                value={episode}
                onChange={(e) => setEpisode(Math.max(0, parseInt(e.target.value) || 0))}
                className="w-12 h-8 text-center border rounded-lg text-sm"
                min={0}
              />
              <span className="text-sm text-gray-600">эпизод</span>
              <button
                onClick={handleSave}
                className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-full"
              >
                Сохранить
              </button>
              <button
                onClick={() => setIsEditing(false)}
                className="px-3 py-1.5 bg-gray-200 hover:bg-gray-300 text-gray-600 text-sm font-medium rounded-full"
              >
                ✕
              </button>
            </>
          ) : (
            <>
              <button 
                onClick={() => setIsEditing(true)}
                className="inline-flex items-center gap-1.5 bg-gray-100 hover:bg-gray-200 text-gray-800 text-sm font-medium px-3 py-1.5 rounded-full transition-colors border border-gray-200"
              >
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
                <span>{item.current_season} сезон · {item.current_episode} серия</span>
              </button>
              <a 
                href="#" 
                onClick={(e) => { e.preventDefault(); goToPreview(); }}
                className="inline-flex items-center gap-1.5 bg-indigo-50 hover:bg-indigo-100 text-indigo-700 text-sm font-medium px-3 py-1.5 rounded-full transition-colors border border-indigo-200"
              >
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z"/>
                </svg>
                Смотреть
              </a>
            </>
          )}
          <button 
            onClick={onDelete}
            className="ml-auto inline-flex items-center justify-center text-gray-400 hover:text-red-500 bg-transparent hover:bg-red-50 p-2 rounded-full transition-colors"
            title="Удалить"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

function HomeContent() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedSeries, setSelectedSeries] = useState<SeriesSearchResult | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [isPending, startTransition] = useTransition();
  const [pendingId, setPendingId] = useState<number | null>(null);
  const queryClient = useQueryClient();
  const { user, logout } = useAuth();
  const router = useRouter();

  const { data: progress, isLoading } = useQuery({
    queryKey: ['progress', statusFilter],
    queryFn: () => api.progress.getAll(statusFilter || undefined),
  });

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  const handleAddSeries = (series: SeriesSearchResult) => {
    setSelectedSeries(series);
    startTransition(async () => {
      try {
        await addProgress(
          series.id,
          series.title,
          series.poster_url || null,
          false, // is_serial - default to false for search results
          1,
          0,
          'watching'
        );
        toast.success('Добавлено в список');
        queryClient.invalidateQueries({ queryKey: ['progress'] });
        setSearchQuery('');
        setSelectedSeries(null);
      } catch (err) {
        toast.error('Ошибка при добавлении');
      }
    });
  };

  const handleUpdate = (data: UpdateProgressRequest) => {
    const item = progress?.find(p => p.series_id === data.series_id);
    if (!item) return;
    
    setPendingId(data.series_id);
    startTransition(async () => {
      try {
        // We need series_id as string (UUID) for Server Action, but we have kinopoisk_id
        // For now, we'll use the API route for updates since we need to get the series UUID
        await api.progress.update(data);
        queryClient.invalidateQueries({ queryKey: ['progress'] });
        toast.success('Обновлено');
      } catch (err) {
        toast.error('Ошибка при обновлении');
      } finally {
        setPendingId(null);
      }
    });
  };

  const handleDelete = (seriesId: number) => {
    setPendingId(seriesId);
    startTransition(async () => {
      try {
        await api.progress.delete(seriesId);
        queryClient.invalidateQueries({ queryKey: ['progress'] });
        toast.success('Удалено');
      } catch (err) {
        toast.error('Ошибка при удалении');
      } finally {
        setPendingId(null);
      }
    });
  };

  const handleSelectSeries = (series: SeriesSearchResult) => {
    handleAddSeries(series);
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-2xl font-bold">SeriesTracker</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-muted-foreground">{user?.email}</span>
            <Button variant="outline" size="sm" onClick={handleLogout}>
              <LogOut className="w-4 h-4 mr-2" />
              Выйти
            </Button>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-6">
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3">Добавить сериал</h2>
          <div className="relative">
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Поиск сериалов..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
              <Button>
                <Search className="w-4 h-4" />
              </Button>
            </div>
            {searchQuery && <SearchResults query={searchQuery} onSelect={handleSelectSeries} />}
          </div>
        </div>

        <div className="mb-6">
          <div className="flex gap-2 flex-wrap">
            <Button
              variant={statusFilter === '' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setStatusFilter('')}
            >
              Все
            </Button>
            {Object.entries(STATUS_LABELS).map(([key, { label }]) => (
              <Button
                key={key}
                variant={statusFilter === key ? 'default' : 'outline'}
                size="sm"
                onClick={() => setStatusFilter(key)}
              >
                {label}
              </Button>
            ))}
          </div>
        </div>

        <div className="grid gap-4 grid-cols-2">
          {isLoading ? (
            <div className="text-center py-8">Загрузка...</div>
          ) : !Array.isArray(progress) ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              Ошибка загрузки данных
            </div>
          ) : progress.length === 0 ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              У вас пока нет сериалов в списке
            </div>
          ) : (
            progress.map((item) => (
              <ProgressCard
                key={item.series_id}
                item={item}
                onUpdate={handleUpdate}
                onDelete={() => handleDelete(item.series_id)}
                isPending={pendingId === item.series_id}
              />
            ))
          )}
        </div>
      </main>
    </div>
  );
}

export default function Home() {
  return (
    <ProtectedRoute>
      <HomeContent />
    </ProtectedRoute>
  );
}
