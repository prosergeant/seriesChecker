# ProgressCard Component

## Overview
ProgressCard displays a user's series/movie progress in a card layout. It shows the poster image (with skeleton loading and placeholder support), series title, watch status, current season/episode, and action buttons for editing progress, watching on Kinopoisk, viewing related content, and deleting from watchlist.

## Architecture
- Renders a flex row layout with poster on the left and content on the right
- Status is managed via dropdown menu with predefined status labels
- Inline editing of season/episode is supported
- Integrates with RelatedMoviesModal for browsing related content
- Uses the PosterImage component for consistent image handling and loading states

## Recent Changes (feat: use PosterImage in ProgressCard for skeleton loading and placeholder)

### What Changed
- **Replaced raw `<img>` element** (lines 57-66): Removed conditional `{item.poster_url && (...)}` wrapper
- **Added PosterImage import**: `import { PosterImage } from '@/components/ui/PosterImage'`
- **Updated poster rendering** (lines 58-62):
  - Old: conditionally rendered div with raw img tag only when `poster_url` exists
  - New: Always renders PosterImage component which handles:
    - Placeholder state: Shows Film icon when poster_url is null/undefined
    - Skeleton state: Shows loading skeleton while image loads
    - Loaded state: Shows actual image with proper object-fit
  - Added `min-h-[140px]` to prevent layout shift during skeleton loading phase
  - Removed inline `style={{ minHeight: "100%" }}`

### Why This Change
- **Better UX**: Container always renders, preventing layout shift when poster loads or is missing
- **Consistent design**: Uses centralized PosterImage component for all poster rendering
- **Skeleton loading**: Visual feedback while image is loading
- **Graceful degradation**: Film icon placeholder when poster_url is unavailable
- **Type-safe**: PosterImage handles null/undefined src properly

### Testing
- Build: Compiled successfully with no TypeScript errors
- Unit tests: All 22 tests pass
