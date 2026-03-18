'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { api, ProgressItem, SeriesSearchResult, UpdateProgressRequest } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Search, Play, Plus, Trash2, Check, Eye, Clock, X, LogOut } from 'lucide-react';
import { toast } from 'sonner';
import { useAuth } from '@/components/auth-context';
import { ProtectedRoute } from '@/components/protected-route';

const STATUS_LABELS: Record<string, { label: string; icon: React.ReactNode }> = {
  watching: { label: 'Смотрю', icon: <Eye className="w-4 h-4" /> },
  completed: { label: 'Просмотрено', icon: <Check className="w-4 h-4" /> },
  planned: { label: 'Запланировано', icon: <Clock className="w-4 h-4" /> },
  dropped: { label: 'Брошено', icon: <X className="w-4 h-4" /> },
  on_hold: { label: 'На паузе', icon: <Clock className="w-4 h-4" /> },
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

function ProgressCard({ item, onUpdate, onDelete }: { 
  item: ProgressItem; 
  onUpdate: (data: UpdateProgressRequest) => void;
  onDelete: () => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [season, setSeason] = useState(item.current_season);
  const [episode, setEpisode] = useState(item.current_episode);

  const statusInfo = STATUS_LABELS[item.status] || { label: item.status, icon: null };

  const handleSave = () => {
    onUpdate({ ...item, current_season: season, current_episode: episode });
    setIsEditing(false);
  };

  const goToPreview = () => {
    window.open(`https://www.sspoisk.ru/${item.is_serial ? 'series' : 'film'}/${item.kinopoisk_id}`, '_blank')
  }

  return (
    <Card className="flex flex-row overflow-hidden">
      {item.poster_url && (
        <img src={item.poster_url} alt={item.title} className="w-24 h-36 object-cover" />
      )}
      <div className="flex-1 p-4">
        <div className="flex justify-between items-start">
          <div>
            <h3 className="font-semibold">{item.title}</h3>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm text-muted-foreground flex items-center gap-1">
                {statusInfo.icon}
                {statusInfo.label}
              </span>
            </div>
          </div>
          <Button variant="ghost" size="icon" onClick={onDelete}>
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>

        {isEditing ? (
          <div className="mt-3 flex gap-2 items-center">
            <Input
              type="number"
              value={season}
              onChange={(e) => setSeason(parseInt(e.target.value) || 1)}
              className="w-16"
              min={1}
            />
            <span>сезон</span>
            <Input
              type="number"
              value={episode}
              onChange={(e) => setEpisode(parseInt(e.target.value) || 0)}
              className="w-16"
              min={0}
            />
            <span>эпизод</span>
            <Button size="sm" onClick={handleSave}>Сохранить</Button>
          </div>
        ) : (
          <div className="mt-3 flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
              {item.current_season} сезон, {item.current_episode} эпизод
            </Button>
          </div>
        )}

        <Button className="mt-3" variant="outline" size="sm" onClick={goToPreview}>
          Просмотр
        </Button>
      </div>
    </Card>
  );
}

function HomeContent() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedSeries, setSelectedSeries] = useState<SeriesSearchResult | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const queryClient = useQueryClient();
  const { user, logout } = useAuth();
  const router = useRouter();

  const { data: progress, isLoading } = useQuery({
    queryKey: ['progress', statusFilter],
    queryFn: () => api.progress.getAll(statusFilter || undefined),
  });

  const addMutation = useMutation({
    mutationFn: (series: SeriesSearchResult) =>
      api.progress.update({
        series_id: series.id,
        current_season: 1,
        current_episode: 0,
        status: 'watching',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] });
      toast.success('Добавлено в список');
      setSearchQuery('');
      setSelectedSeries(null);
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: UpdateProgressRequest) => api.progress.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] });
      toast.success('Обновлено');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seriesId: number) => api.progress.delete(seriesId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] });
      toast.success('Удалено');
    },
  });

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  const handleSelectSeries = (series: SeriesSearchResult) => {
    setSelectedSeries(series);
    addMutation.mutate(series);
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

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
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
                onUpdate={(data) => updateMutation.mutate(data)}
                onDelete={() => deleteMutation.mutate(item.series_id)}
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
