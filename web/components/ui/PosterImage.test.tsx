import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PosterImage } from './PosterImage';

describe('PosterImage', () => {
  it('shows Film icon placeholder when src is null', () => {
    render(<PosterImage src={null} alt="test" />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('shows Film icon placeholder when src is undefined', () => {
    render(<PosterImage alt="test" />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('shows skeleton while image is loading', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    expect(screen.queryByTestId('poster-skeleton')).toBeTruthy();
  });

  it('hides skeleton after image loads', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.load(img);
    expect(screen.queryByTestId('poster-skeleton')).toBeNull();
  });

  it('hides skeleton after image load error', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.error(img);
    expect(screen.queryByTestId('poster-skeleton')).toBeNull();
  });

  it('renders img with correct src and alt', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="My Poster" />);
    const img = screen.getByRole('img');
    expect(img).toHaveAttribute('src', 'http://example.com/poster.jpg');
    expect(img).toHaveAttribute('alt', 'My Poster');
  });
});
