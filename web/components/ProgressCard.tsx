"use client";

import { useState } from "react";
import { Eye, Check, Clock, X } from "lucide-react";
import { RelatedMoviesModal } from "@/components/RelatedMoviesModal";
import { PosterImage } from "@/components/ui/PosterImage";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ProgressItem, UpdateProgressRequest } from "@/lib/api";

export const STATUS_LABELS: Record<
  string,
  { label: string; icon: React.ReactNode }
> = {
  watching: { label: "Смотрю", icon: <Eye className="w-4 h-4" /> },
  completed: { label: "Просмотрено", icon: <Check className="w-4 h-4" /> },
  planned: { label: "Запланировано", icon: <Clock className="w-4 h-4" /> },
  dropped: { label: "Брошено", icon: <X className="w-4 h-4" /> },
  on_hold: { label: "На паузе", icon: <Clock className="w-4 h-4" /> },
};

export function ProgressCard({
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
      <PosterImage
        src={item.poster_url}
        alt={item.title}
        className="w-[100px] md:w-[120px] flex-shrink-0 min-h-[140px]"
      />
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
