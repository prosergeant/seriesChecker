package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/repository"
)

type ProgressService struct {
	progressRepo  *repository.UserProgressRepository
	seriesRepo    *repository.SeriesRepository
	seriesService *SeriesService
}

func NewProgressService(progressRepo *repository.UserProgressRepository, seriesRepo *repository.SeriesRepository, seriesService *SeriesService) *ProgressService {
	return &ProgressService{
		progressRepo:  progressRepo,
		seriesRepo:    seriesRepo,
		seriesService: seriesService,
	}
}

type ProgressResult struct {
	ID             int    `json:"id"`
	UserID         string `json:"user_id"`
	SeriesID       int    `json:"series_id"`
	KinopoiskID    int    `json:"kinopoisk_id"`
	Title          string `json:"title"`
	PosterURL      string `json:"poster_url,omitempty"`
	CurrentSeason  int    `json:"current_season"`
	CurrentEpisode int    `json:"current_episode"`
	Status         string `json:"status"`
	IsSerial       bool   `json:"is_serial"`
}

func (s *ProgressService) GetListByStatus(ctx context.Context, userID uuid.UUID, status string) ([]ProgressResult, error) {
	var results []ProgressResult

	if status != "" {
		rows, err := s.progressRepo.GetListByStatus(ctx, pgtype.UUID{Bytes: userID, Valid: true}, status)
		if err != nil {
			return nil, err
		}
		results = make([]ProgressResult, len(rows))
		for i, r := range rows {
			results[i] = ProgressResult{
				UserID:         userID.String(),
				SeriesID:       int(r.KinopoiskID),
				KinopoiskID:    int(r.KinopoiskID),
				Title:          r.Title,
				PosterURL:      r.PosterUrl.String,
				CurrentSeason:  int(r.CurrentSeason),
				CurrentEpisode: int(r.CurrentEpisode),
				Status:         r.Status,
				IsSerial:       r.IsSerial.Bool,
			}
		}
	} else {
		rows, err := s.progressRepo.GetAll(ctx, pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			return nil, err
		}
		results = make([]ProgressResult, len(rows))
		for i, r := range rows {
			results[i] = ProgressResult{
				UserID:         userID.String(),
				SeriesID:       int(r.KinopoiskID),
				KinopoiskID:    int(r.KinopoiskID),
				Title:          r.Title,
				PosterURL:      r.PosterUrl.String,
				CurrentSeason:  int(r.CurrentSeason),
				CurrentEpisode: int(r.CurrentEpisode),
				Status:         r.Status,
				IsSerial:       r.IsSerial.Bool,
			}
		}
	}

	return results, nil
}

func (s *ProgressService) UpdateProgress(ctx context.Context, userID uuid.UUID, seriesID, currentSeason, currentEpisode int, status string) (*ProgressResult, error) {
	// Получаем или создаем серию в БД
	series, err := s.seriesRepo.GetByKinopoiskID(ctx, int32(seriesID))
	if err != nil || series.KinopoiskID == 0 {
		// Серии нет в БД - запрашиваем из Kinopoisk и сохраняем
		_, err := s.seriesService.GetByID(ctx, seriesID)
		if err != nil {
			return nil, err
		}
		// GetByID уже сохраняет в БД, получаем снова
		series, err = s.seriesRepo.GetByKinopoiskID(ctx, int32(seriesID))
		if err != nil {
			return nil, err
		}
	}

	existing, err := s.progressRepo.Get(ctx, pgtype.UUID{Bytes: userID, Valid: true}, series.ID)
	if err == nil && existing.ID.Bytes != uuid.Nil {
		updated, err := s.progressRepo.Update(ctx, repository.UpdateProgressParams{
			UserID:         pgtype.UUID{Bytes: userID, Valid: true},
			SeriesID:       series.ID,
			CurrentSeason:  int32(currentSeason),
			CurrentEpisode: int32(currentEpisode),
			Status:         status,
		})
		if err != nil {
			return nil, err
		}
		return &ProgressResult{
			UserID:         userID.String(),
			SeriesID:       seriesID,
			KinopoiskID:    seriesID,
			Title:          series.Title,
			PosterURL:      series.PosterUrl.String,
			CurrentSeason:  int(updated.CurrentSeason),
			CurrentEpisode: int(updated.CurrentEpisode),
			Status:         updated.Status,
		}, nil
	}

	created, err := s.progressRepo.Create(ctx, repository.CreateProgressParams{
		UserID:         pgtype.UUID{Bytes: userID, Valid: true},
		SeriesID:       series.ID,
		CurrentSeason:  int32(currentSeason),
		CurrentEpisode: int32(currentEpisode),
		Status:         status,
	})
	if err != nil {
		return nil, err
	}

	return &ProgressResult{
		UserID:         userID.String(),
		SeriesID:       seriesID,
		KinopoiskID:    seriesID,
		Title:          series.Title,
		PosterURL:      series.PosterUrl.String,
		CurrentSeason:  int(created.CurrentSeason),
		CurrentEpisode: int(created.CurrentEpisode),
		Status:         created.Status,
	}, nil
}

func (s *ProgressService) DeleteProgress(ctx context.Context, userID uuid.UUID, seriesID int) error {
	series, err := s.seriesRepo.GetByKinopoiskID(ctx, int32(seriesID))
	if err != nil {
		return err
	}

	return s.progressRepo.Delete(ctx, pgtype.UUID{Bytes: userID, Valid: true}, series.ID)
}
