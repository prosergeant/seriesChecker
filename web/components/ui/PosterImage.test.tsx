import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PosterImage } from './PosterImage';

describe('PosterImage', () => {
  it('показывает плейсхолдер Film icon когда src равен null', () => {
    render(<PosterImage src={null} alt="test" />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('показывает плейсхолдер Film icon когда src не передан', () => {
    render(<PosterImage alt="test" />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('показывает spinner пока изображение загружается', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    expect(screen.queryByTestId('poster-skeleton')).toBeTruthy();
  });

  it('скрывает spinner после загрузки изображения', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.load(img);
    expect(screen.queryByTestId('poster-skeleton')).toBeNull();
  });

  it('скрывает spinner при ошибке загрузки изображения', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.error(img);
    expect(screen.queryByTestId('poster-skeleton')).toBeNull();
  });

  it('показывает плейсхолдер Film icon при ошибке загрузки изображения (404)', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="test" />);
    const img = screen.getByRole('img');
    fireEvent.error(img);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('рендерит img с корректными src и alt', () => {
    render(<PosterImage src="http://example.com/poster.jpg" alt="My Poster" />);
    const img = screen.getByRole('img');
    expect(img).toHaveAttribute('src', 'http://example.com/poster.jpg');
    expect(img).toHaveAttribute('alt', 'My Poster');
  });
});
