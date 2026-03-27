'use client';
import { useState } from 'react';
import { Film } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Skeleton } from './skeleton';

interface PosterImageProps {
  src?: string | null;
  alt: string;
  className?: string;
  imgClassName?: string;
}

export function PosterImage({ src, alt, className, imgClassName }: PosterImageProps) {
  const [loaded, setLoaded] = useState(false);

  if (!src) {
    return (
      <div data-testid="poster-placeholder" className={cn('flex items-center justify-center bg-muted', className)}>
        <Film className="w-8 h-8 text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className={cn('relative', className)}>
      {!loaded && <Skeleton data-testid="poster-skeleton" className="absolute inset-0 rounded-none" />}
      <img
        src={src}
        alt={alt}
        className={cn('w-full h-full object-cover', imgClassName)}
        onLoad={() => setLoaded(true)}
        onError={() => setLoaded(true)}
      />
    </div>
  );
}
