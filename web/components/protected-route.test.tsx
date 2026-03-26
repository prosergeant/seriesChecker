import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ProtectedRoute } from './protected-route';

const mockPush = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock('@/components/auth-context', () => ({
  useAuth: vi.fn(),
}));

import { useAuth } from '@/components/auth-context';
const mockUseAuth = useAuth as ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ProtectedRoute', () => {
  it('shows spinner when isLoading = true', () => {
    mockUseAuth.mockReturnValue({ isLoading: true, isAuthenticated: false });

    const { container } = render(
      <ProtectedRoute><span>protected</span></ProtectedRoute>
    );

    // Spinner (Loader2) is rendered, children are not
    expect(screen.queryByText('protected')).toBeNull();
    // Loader2 renders an svg
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it('redirects to /login when not authenticated', () => {
    mockUseAuth.mockReturnValue({ isLoading: false, isAuthenticated: false });

    render(<ProtectedRoute><span>protected</span></ProtectedRoute>);

    expect(mockPush).toHaveBeenCalledWith('/login');
    expect(screen.queryByText('protected')).toBeNull();
  });

  it('renders children when authenticated', () => {
    mockUseAuth.mockReturnValue({ isLoading: false, isAuthenticated: true });

    render(<ProtectedRoute><span>protected</span></ProtectedRoute>);

    expect(screen.getByText('protected')).toBeTruthy();
    expect(mockPush).not.toHaveBeenCalled();
  });
});
