import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ProgressCard } from './ProgressCard';
import { ProgressItem } from '@/lib/api';

const baseItem: ProgressItem = {
  id: 1,
  user_id: 'u1',
  series_id: 42,
  kinopoisk_id: 999,
  title: 'Тест Сериал',
  poster_url: undefined,
  current_season: 2,
  current_episode: 5,
  status: 'watching',
  is_serial: true,
};

const noop = vi.fn();

describe('ProgressCard', () => {
  it('не показывает оверлей загрузки когда isLoading=false', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={false} />
    );
    expect(screen.queryByTestId('progress-card-loading')).toBeNull();
  });

  it('показывает оверлей загрузки когда isLoading=true', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={true} />
    );
    expect(screen.getByTestId('progress-card-loading')).toBeTruthy();
  });

  it('блокирует кнопку удаления когда isLoading=true', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={true} />
    );
    const deleteBtn = screen.getByTitle('Удалить');
    expect(deleteBtn).toBeDisabled();
  });

  it('кнопка удаления активна когда isLoading=false', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={false} />
    );
    const deleteBtn = screen.getByTitle('Удалить');
    expect(deleteBtn).not.toBeDisabled();
  });

  it('блокирует кнопку сохранения когда isLoading=true', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={true} />
    );
    // Open edit mode by clicking the season/episode button
    const editButton = screen.getByText(/сезон/);
    fireEvent.click(editButton);
    const saveBtn = screen.getByText('Сохранить');
    expect(saveBtn).toBeDisabled();
  });

  it('статус-дропдаун задизейблен когда isLoading=true', () => {
    render(
      <ProgressCard item={baseItem} onUpdate={noop} onDelete={noop} isLoading={true} />
    );
    // The status badge/trigger should be disabled
    const statusTrigger = screen.getByText('Смотрю');
    expect(statusTrigger.closest('button')).toBeDisabled();
  });
});
