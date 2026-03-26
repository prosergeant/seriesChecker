import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PosterImage } from './PosterImage';

describe('PosterImage', () => {
  it('shows Film icon placeholder when src is absent', () => {
    render(<PosterImage src={null} alt="test" />);
    // Film icon from lucide-react renders as svg with a specific aria role or test-id
    // No img element should be present
    expect(screen.queryByRole('img')).toBeNull();
    // The placeholder container is visible
    expect(document.querySelector('svg')).toBeTruthy(); // Film icon
  });

  it('shows skeleton while image is loading', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    // Skeleton should be visible before onLoad fires
    // Skeleton renders as a div with animate-pulse class
    const skeleton = document.querySelector('[class*="animate-pulse"]');
    expect(skeleton).toBeTruthy();
  });

  it('hides skeleton after image loads', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.load(img);
    // After load, no skeleton
    const skeleton = document.querySelector('[class*="animate-pulse"]');
    expect(skeleton).toBeNull();
  });

  it('renders img with correct src and alt', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="My Poster" />);
    const img = screen.getByRole('img');
    expect(img).toHaveAttribute('src', 'http://example.com/poster.jpg');
    expect(img).toHaveAttribute('alt', 'My Poster');
  });
});
