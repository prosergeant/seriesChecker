"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

interface PlayerModalProps {
  kinopoiskId: number;
  isSerial: boolean;
  title: string;
}

export function PlayerModal({ kinopoiskId, isSerial, title }: PlayerModalProps) {
  const [isOpen, setIsOpen] = useState(false);

  const playerUrl = `https://fbfree.lat/${isSerial ? "series" : "film"}/${kinopoiskId}`;

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger>
        <a
          href="#"
          onClick={(e) => e.preventDefault()}
          className="inline-flex items-center gap-1.5 bg-indigo-50 hover:bg-indigo-100 text-indigo-700 text-sm font-medium px-3 py-1.5 rounded-full transition-colors border border-indigo-200"
        >
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
            <path d="M8 5v14l11-7z" />
          </svg>
          Смотреть
        </a>
      </DialogTrigger>
      <DialogContent className="max-w-5xl w-full p-0 overflow-hidden">
        <DialogHeader className="px-4 py-3 border-b">
          <DialogTitle className="text-base">{title}</DialogTitle>
        </DialogHeader>
        <div className="w-full" style={{ height: "70vh" }}>
          {isOpen && (
            <iframe
              src={playerUrl}
              className="w-full h-full border-0"
              allowFullScreen
              allow="autoplay; fullscreen"
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
