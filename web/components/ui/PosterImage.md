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
