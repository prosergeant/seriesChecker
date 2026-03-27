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
  it('показывает спиннер когда isLoading = true', () => {
    mockUseAuth.mockReturnValue({ isLoading: true, isAuthenticated: false });

    const { container } = render(
      <ProtectedRoute><span>protected</span></ProtectedRoute>
    );

    // Спиннер (Loader2) отрендерен, children не показаны
    expect(screen.queryByText('protected')).toBeNull();
    // Loader2 рендерит svg
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it('редиректит на /login если не авторизован', () => {
    mockUseAuth.mockReturnValue({ isLoading: false, isAuthenticated: false });

    render(<ProtectedRoute><span>protected</span></ProtectedRoute>);

    expect(mockPush).toHaveBeenCalledWith('/login');
    expect(screen.queryByText('protected')).toBeNull();
  });

  it('рендерит children если авторизован', () => {
    mockUseAuth.mockReturnValue({ isLoading: false, isAuthenticated: true });

    render(<ProtectedRoute><span>protected</span></ProtectedRoute>);

    expect(screen.getByText('protected')).toBeTruthy();
    expect(mockPush).not.toHaveBeenCalled();
  });
});
