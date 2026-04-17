import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

// Mock RelatedMoviesModal globally to avoid QueryClient dependency in unit tests
vi.mock('@/components/RelatedMoviesModal', () => ({
  RelatedMoviesModal: () => null,
}));

vi.mock('./components/RelatedMoviesModal', () => ({
  RelatedMoviesModal: () => null,
}));
