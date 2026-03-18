'use server';

import { revalidatePath } from 'next/cache';
import { getUserId } from '@/lib/auth/server';
import { getProgressList, createProgress, updateProgress, deleteProgress, getSeriesByKinopoiskId, createSeries } from '@/lib/db/queries';
import { ProgressStatus } from '@/lib/db/types';

export async function getProgress(status?: ProgressStatus) {
  const userId = await getUserId();
  if (!userId) throw new Error('Unauthorized');
  return getProgressList(userId, status);
}

export async function addProgress(
  kinopoiskId: number,
  title: string,
  posterUrl: string | null,
  isSerial: boolean,
  currentSeason: number,
  currentEpisode: number,
  status: ProgressStatus
) {
  const userId = await getUserId();
  if (!userId) throw new Error('Unauthorized');
  
  // Check if series exists in DB, if not create it
  let series = await getSeriesByKinopoiskId(kinopoiskId);
  if (!series) {
    series = await createSeries(kinopoiskId, title, null, posterUrl, null, null, null, isSerial);
  }
  
  // Create progress record
  await createProgress(userId, series.id, currentSeason, currentEpisode, status);
  revalidatePath('/');
}

export async function updateUserProgress(
  seriesId: string,
  currentSeason: number,
  currentEpisode: number,
  status: ProgressStatus
) {
  const userId = await getUserId();
  if (!userId) throw new Error('Unauthorized');
  
  await updateProgress(userId, seriesId, currentSeason, currentEpisode, status);
  revalidatePath('/');
}

export async function removeProgress(seriesId: string) {
  const userId = await getUserId();
  if (!userId) throw new Error('Unauthorized');
  
  await deleteProgress(userId, seriesId);
  revalidatePath('/');
}
