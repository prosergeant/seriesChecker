"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import {
  api,
  SeriesSearchResult,
  UpdateProgressRequest,
} from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Search, LogOut, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { toast } from "sonner";
import { useAuth } from "@/components/auth-context";
import { ProtectedRoute } from "@/components/protected-route";
import { ProgressCard, STATUS_LABELS } from "@/components/ProgressCard";

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

function HomeContent() {
  const [searchQuery, setSearchQuery] = useState("");
  const [, setSelectedSeries] =
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
