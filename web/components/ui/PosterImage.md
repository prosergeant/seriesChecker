# PosterImage

## What the file does

`PosterImage.tsx` is a reusable React component that renders a series/movie poster image with three distinct states:

1. **No src (absent or null):** renders a centered `Film` icon (lucide-react) on a `bg-muted` background as a placeholder.
2. **src present, loading:** renders the `<img>` element alongside an absolutely-positioned `Skeleton` (animate-pulse) overlay until the image fires its `onLoad` event.
3. **src present, loaded:** the skeleton disappears and the fully-loaded `<img>` is shown.

## Props

| Prop | Type | Description |
|------|------|-------------|
| `src` | `string \| null \| undefined` | URL of the poster image |
| `alt` | `string` | Alt text for the img element |
| `className` | `string?` | Extra classes applied to the outer wrapper |
| `imgClassName` | `string?` | Extra classes applied to the `<img>` tag |

## What was changed and why

**Initial creation (TDD):** This file was created as part of a Test-Driven Development exercise. The test file `PosterImage.test.tsx` was written first and confirmed to fail, then this implementation was written to make all 4 tests pass. The component extracts a common poster-display pattern used across the app (search results, progress list) into a single reusable unit with proper loading UX.

**Code review fixes:**

- **Added `onError` handler** (`onError={() => setLoaded(true)}`): previously a 404 or network-failed image URL would leave `loaded` as `false` indefinitely, causing the skeleton to animate forever. The error handler dismisses the skeleton so the browser's native broken-image indicator is visible instead.
- **Added `data-testid="poster-skeleton"`** to `<Skeleton>` and **`data-testid="poster-placeholder"`** to the placeholder `<div>`: allows tests to query elements by stable test IDs rather than fragile class-name selectors (e.g. `[class*="animate-pulse"]`).
- Updated `skeleton.tsx` to accept and spread `React.HTMLAttributes<HTMLDivElement>` so forwarded props like `data-testid` reach the underlying `<div>`.
