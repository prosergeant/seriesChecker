"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import {
  api,
  ProgressItem,
  SeriesSearchResult,
  UpdateProgressRequest,
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Search, Check, Eye, Clock, X, LogOut, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { toast } from "sonner";
import { useAuth } from "@/components/auth-context";
import { ProtectedRoute } from "@/components/protected-route";
import { RelatedMoviesModal } from "@/components/RelatedMoviesModal";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const STATUS_LABELS: Record<string, { label: string; icon: React.ReactNode }> =
  {
    watching: { label: "Смотрю", icon: <Eye className="w-4 h-4" /> },
    completed: { label: "Просмотрено", icon: <Check className="w-4 h-4" /> },
    planned: { label: "Запланировано", icon: <Clock className="w-4 h-4" /> },
    dropped: { label: "Брошено", icon: <X className="w-4 h-4" /> },
    on_hold: { label: "На паузе", icon: <Clock className="w-4 h-4" /> },
  };

function SearchResults({
  query,
  onSelect,
}: {
  query: string;
  onSelect: (series: SeriesSearchResult) => void;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ["search", query],
    queryFn: () => api.series.search(query),
    enabled: query.length >= 2,
  });

  if (!query || query.length < 2) return null;
  if (isLoading) return (
    <Card className="absolute top-full left-0 right-0 z-50 mt-2">
      <CardContent className="p-2 space-y-1">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="flex items-center gap-3 p-2">
            <Skeleton className="w-10 h-14 shrink-0" />
            <div className="space-y-2 flex-1">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-3 w-1/4" />
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
  if (!data?.length)
    return <div className="text-center py-4">Ничего не найдено</div>;

  return (
    <Card className="absolute top-full left-0 right-0 z-50 mt-2 max-h-80 overflow-auto">
      <CardContent className="p-2">
        {data.map((series, index) => (
          <button
            key={`search-${series.id}-${index}`}
            onClick={() => onSelect(series)}
            className="w-full flex items-center gap-3 p-2 hover:bg-muted rounded-lg text-left"
          >
            {series.poster_url && (
              <img
                src={series.poster_url}
                alt={series.title}
                className="w-10 h-14 object-cover rounded"
              />
            )}
            <div>
              <div className="font-medium">{series.title}</div>
              {series.year && (
                <div className="text-sm text-muted-foreground">
                  {series.year}
                </div>
              )}
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
}: {
  item: ProgressItem;
  onUpdate: (data: UpdateProgressRequest) => void;
  onDelete: () => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [season, setSeason] = useState(item.current_season);
  const [episode, setEpisode] = useState(item.current_episode);

  const statusInfo = STATUS_LABELS[item.status] || {
    label: item.status,
    icon: null,
  };

  const handleSave = () => {
    onUpdate({ series_id: item.series_id, current_season: season, current_episode: episode, status: "watching" });
    setIsEditing(false);
  };

  const goToPreview = () => {
    window.open(
      `https://www.sspoisk.ru/${item.is_serial ? "series" : "film"}/${item.kinopoisk_id}`,
      "_blank",
    );
  };

  return (
    <div className="bg-card rounded-2xl flex flex-row overflow-hidden border border-border shadow-sm hover:shadow-md transition-shadow">
      {item.poster_url && (
        <div className="w-[100px] md:w-[120px] flex-shrink-0">
          <img
            src={item.poster_url}
            alt={item.title}
            className="w-full h-full object-cover"
            style={{ minHeight: "100%" }}
          />
        </div>
      )}
      <div className="flex-1 p-4 sm:p-5 flex flex-col gap-3 min-w-0">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <h3 className="text-lg sm:text-xl font-semibold text-card-foreground tracking-tight truncate pr-2">
            {item.title}
          </h3>
          <DropdownMenu>
            <DropdownMenuTrigger className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/20 text-emerald-700 dark:text-emerald-400 border border-emerald-500/30 whitespace-nowrap hover:bg-emerald-500/30 transition-colors cursor-pointer outline-none">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5"></span>
              {statusInfo.label}
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              {Object.entries(STATUS_LABELS).map(([key, { label }]) => (
                <DropdownMenuItem
                  key={key}
                  onClick={() => onUpdate({ series_id: item.series_id, current_season: item.current_season, current_episode: item.current_episode, status: key })}
                >
                  {label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {isEditing ? (
            <>
              <input
                type="number"
                value={season}
                onChange={(e) =>
                  setSeason(Math.max(1, parseInt(e.target.value) || 1))
                }
                className="w-12 h-8 text-center border rounded-lg text-sm"
                min={1}
              />
              <span className="text-sm text-muted-foreground">сезон</span>
              <input
                type="number"
                value={episode}
                onChange={(e) =>
                  setEpisode(Math.max(0, parseInt(e.target.value) || 0))
                }
                className="w-12 h-8 text-center border rounded-lg text-sm"
                min={0}
              />
              <span className="text-sm text-muted-foreground">эпизод</span>
              <button
                onClick={handleSave}
                className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-full"
              >
                Сохранить
              </button>
              <button
                onClick={() => setIsEditing(false)}
                className="px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded-full"
              >
                X
              </button>
            </>
          ) : (
            <>
              {item.is_serial && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="inline-flex items-center gap-1.5 bg-muted hover:bg-muted/80 text-foreground text-sm font-medium px-3 py-1.5 rounded-full transition-colors border border-border"
                >
                  <svg
                    className="w-3 h-3"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                    />
                  </svg>
                  <span>
                    {item.current_season} сезон · {item.current_episode} серия
                  </span>
                </button>
              )}
              <a
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  goToPreview();
                }}
                className="inline-flex items-center gap-1.5 bg-primary/10 hover:bg-primary/20 text-primary text-sm font-medium px-3 py-1.5 rounded-full transition-colors border border-primary/20"
              >
                <svg
                  className="w-3 h-3"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M8 5v14l11-7z" />
                </svg>
                Смотреть
              </a>
            </>
          )}
          <RelatedMoviesModal
            kinopoiskId={item.kinopoisk_id}
            title={item.title}
            posterUrl={item.poster_url}
          />
          <button
            onClick={onDelete}
            className="inline-flex items-center justify-center text-muted-foreground hover:text-destructive bg-transparent hover:bg-destructive/10 p-2 rounded-full transition-colors"
            title="Удалить"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

function HomeContent() {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedSeries, setSelectedSeries] =
    useState<SeriesSearchResult | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const queryClient = useQueryClient();
  const { user, logout } = useAuth();
  const router = useRouter();
  const { theme, setTheme } = useTheme();

  const { data: progress, isLoading } = useQuery({
    queryKey: ["progress", statusFilter],
    queryFn: () => api.progress.getAll(statusFilter || undefined),
  });

  const addMutation = useMutation({
    mutationFn: (series: SeriesSearchResult) =>
      api.progress.update({
        series_id: series.id,
        current_season: 1,
        current_episode: 0,
        status: "watching",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["progress"] });
      toast.success("Добавлено в список");
      setSearchQuery("");
      setSelectedSeries(null);
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: UpdateProgressRequest) => api.progress.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["progress"] });
      toast.success("Обновлено");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seriesId: number) => api.progress.delete(seriesId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["progress"] });
      toast.success("Удалено");
    },
  });

  const handleLogout = async () => {
    await logout();
    router.push("/login");
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
          <div className="flex items-center gap-2 sm:gap-4">
            <span className="hidden sm:block text-sm text-muted-foreground">{user?.email}</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            >
              {theme === "dark" ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
            </Button>
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
            {searchQuery && (
              <SearchResults
                query={searchQuery}
                onSelect={handleSelectSeries}
              />
            )}
          </div>
        </div>

        <div className="mb-6">
          <div className="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0 sm:flex-wrap">
            <Button
              variant={statusFilter === "" ? "default" : "outline"}
              size="sm"
              className="shrink-0"
              onClick={() => setStatusFilter("")}
            >
              Все
            </Button>
            {Object.entries(STATUS_LABELS).map(([key, { label }]) => (
              <Button
                key={key}
                variant={statusFilter === key ? "default" : "outline"}
                size="sm"
                className="shrink-0"
                onClick={() => setStatusFilter(key)}
              >
                {label}
              </Button>
            ))}
          </div>
        </div>

        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2">
          {isLoading ? (
            <>
              {[...Array(4)].map((_, i) => (
                <Card key={i}>
                  <CardContent className="p-4 flex gap-3">
                    <Skeleton className="w-20 h-28 shrink-0" />
                    <div className="flex-1 space-y-2 pt-1">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/3" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </>
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
