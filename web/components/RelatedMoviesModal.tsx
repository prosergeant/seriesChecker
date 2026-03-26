"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, SimilarMovie, RelationMovie } from "@/lib/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { PosterImage } from "@/components/ui/PosterImage";
import { Play, Plus, MoreHorizontal, Film, Link2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

interface RelatedMoviesModalProps {
  kinopoiskId: number;
  title: string;
  posterUrl?: string;
}

type TabType = "similar" | "relations";

const RELATION_LABELS: Record<string, { label: string; color: string }> = {
  SEQUEL: { label: "Сиквел", color: "text-green-400" },
  PREQUEL: { label: "Приквел", color: "text-blue-400" },
  SPIN_OFF: { label: "Спин-офф", color: "text-purple-400" },
  SPUN_OFF_FROM: { label: "Отпочковано", color: "text-orange-400" },
  REFERENCES_IN: { label: "Упоминается в", color: "text-yellow-400" },
  EDITED_INTO: { label: "Переработано в", color: "text-pink-400" },
  SIMILAR: { label: "Похожее", color: "text-gray-400" },
};

function getRelationInfo(movie: RelationMovie | SimilarMovie) {
  if ("relationType" in movie) {
    const info = RELATION_LABELS[movie.relationType];
    return info || { label: movie.relationType, color: "text-gray-400" };
  }
  return null;
}

export function RelatedMoviesModal({
  kinopoiskId,
  title,
}: RelatedMoviesModalProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>("similar");
  const queryClient = useQueryClient();

  const { data: similar, isLoading: loadingSimilar } = useQuery({
    queryKey: ["similar", kinopoiskId],
    queryFn: () => api.series.getSimilar(kinopoiskId),
    enabled: isOpen && activeTab === "similar",
  });

  const { data: relations, isLoading: loadingRelations } = useQuery({
    queryKey: ["relations", kinopoiskId],
    queryFn: () => api.series.getRelations(kinopoiskId),
    enabled: isOpen && activeTab === "relations",
  });

  const handleAddToProgress = async (movie: SimilarMovie | RelationMovie) => {
    const id = "filmId" in movie ? movie.filmId : movie.kinopoiskId;
    const movieTitle = movie.nameRu || movie.nameEn || movie.nameOriginal || "Без названия";
    const poster = movie.posterUrl;

    try {
      await api.progress.update({
        series_id: id,
        current_season: 1,
        current_episode: 0,
        status: "planned",
      });
      queryClient.invalidateQueries({ queryKey: ["progress"] });
      toast.success(`${movieTitle} добавлен в список`);
    } catch (error) {
      toast.error("Не удалось добавить");
    }
  };

  const isLoading = activeTab === "similar" ? loadingSimilar : loadingRelations;
  const movies = activeTab === "similar" ? similar : relations;

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger
        render={
          <button
            className="inline-flex items-center justify-center text-gray-400 hover:text-gray-600 bg-transparent hover:bg-gray-100 p-2 rounded-full transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
            aria-label="Похожие и связанные"
            title="Похожие и связанные"
            onClick={() => setIsOpen(true)}
          >
            <MoreHorizontal className="w-4 h-4" />
          </button>
        }
      />
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="text-lg">{title}</DialogTitle>
        </DialogHeader>

        <div className="flex gap-2 border-b pb-2">
          <Button
            variant={activeTab === "similar" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("similar")}
          >
            <Film className="w-4 h-4 mr-2" />
            Похожие
          </Button>
          <Button
            variant={activeTab === "relations" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("relations")}
          >
            <Link2 className="w-4 h-4 mr-2" />
            Связанные
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto mt-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : !movies?.length ? (
            <div className="text-center py-12 text-muted-foreground">
              Ничего не найдено
            </div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              {movies.map((movie, index) => {
                const id = "filmId" in movie ? movie.filmId : movie.kinopoiskId;
                const movieTitle = movie.nameRu || movie.nameEn || movie.nameOriginal || "Без названия";
                const poster = movie.posterUrl;
                const relationInfo = getRelationInfo(movie);
                const uniqueKey = `${id}-${index}`;

                return (
                  <div
                    key={uniqueKey}
                    tabIndex={0}
                    role="button"
                    aria-label={`${movieTitle} — Отслеживать`}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        handleAddToProgress(movie);
                      }
                    }}
                    className={cn("group relative rounded-lg bg-muted outline-none focus-visible:ring-3 focus-visible:ring-ring/50", "hover-hover:overflow-hidden", "hover-none:overflow-visible")}
                  >
                    <PosterImage
                      src={poster}
                      alt={movieTitle}
                      className="w-full aspect-[2/3]"
                      imgClassName="aspect-[2/3]"
                    />
                    <div className={cn("absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent transition-opacity", "hover-hover:opacity-0 hover-hover:group-hover:opacity-100 hover-hover:group-focus-within:opacity-100", "hover-none:hidden")} />
                    <div className={cn("p-2", "hover-hover:absolute hover-hover:bottom-0 hover-hover:left-0 hover-hover:right-0 hover-hover:translate-y-full hover-hover:group-hover:translate-y-0 hover-hover:group-focus-within:translate-y-0 hover-hover:transition-transform", "hover-none:relative hover-none:bg-muted")}>
                      <div className={cn("text-xs font-medium line-clamp-2 mb-1", "hover-hover:text-white", "hover-none:text-foreground")}>
                        {movieTitle}
                      </div>
                      {relationInfo && (
                        <span className={`text-xs ${relationInfo.color}`}>
                          {relationInfo.label}
                        </span>
                      )}
                      <Button
                        size="sm"
                        variant="secondary"
                        className="w-full mt-2 h-8 text-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleAddToProgress(movie);
                        }}
                      >
                        <Plus className="w-3 h-3 mr-1" />
                        Отслеживать
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
