package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/kinopoisk"
	"github.com/prosergeant/seriesChecker/internal/repository"
)

var ErrSeriesNotFound = errors.New("series not found")

type SeriesService struct {
	seriesRepo *repository.SeriesRepository
	kinopoisk  *kinopoisk.Client
}

func NewSeriesService(seriesRepo *repository.SeriesRepository, kinopoiskClient *kinopoisk.Client) *SeriesService {
	return &SeriesService{
		seriesRepo: seriesRepo,
		kinopoisk:  kinopoiskClient,
	}
}

type SeriesSearchResult struct {
	ID            int    `json:"id"`
	KinopoiskID   int    `json:"kinopoisk_id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title,omitempty"`
	PosterURL     string `json:"poster_url,omitempty"`
	Year          int    `json:"year,omitempty"`
	Description   string `json:"description,omitempty"`
	TotalEpisodes int    `json:"total_episodes,omitempty"`
	TotalSeasons  int    `json:"total_seasons,omitempty"`
	IsSerial      bool   `json:"is_serial"`
}

func (s *SeriesService) Search(ctx context.Context, query string) ([]SeriesSearchResult, error) {
	films, err := s.kinopoisk.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]SeriesSearchResult, 0, len(films))
	for _, f := range films {
		results = append(results, SeriesSearchResult{
			ID:            f.GetKinopoiskID(),
			KinopoiskID:   f.GetKinopoiskID(),
			Title:         f.NameRU,
			OriginalTitle: f.NameEN,
			PosterURL:     f.PosterURLPreview,
			Year:          parseYear(f.Year),
		})
	}

	return results, nil
}

func (s *SeriesService) GetByID(ctx context.Context, kinopoiskID int) (*SeriesSearchResult, error) {
	// 1. Сначала проверяем БД (кеш)
	existing, err := s.seriesRepo.GetByKinopoiskID(ctx, int32(kinopoiskID))
	if err == nil && existing.KinopoiskID > 0 {
		return &SeriesSearchResult{
			ID:            int(existing.KinopoiskID),
			KinopoiskID:   int(existing.KinopoiskID),
			Title:         existing.Title,
			OriginalTitle: existing.OriginalTitle.String,
			PosterURL:     existing.PosterUrl.String,
			Year:          int(existing.Year.Int32),
			Description:   existing.Description.String,
			TotalSeasons:  int(existing.TotalSeasons.Int32),
			IsSerial:      existing.IsSerial.Bool,
		}, nil
	}

	// 2. Если нет в БД - запрашиваем Kinopoisk
	film, err := s.kinopoisk.GetFilm(ctx, kinopoiskID)
	if err != nil {
		return nil, err
	}

	result := SeriesSearchResult{
		ID:            film.KinopoiskID,
		KinopoiskID:   film.KinopoiskID,
		Title:         film.NameRU,
		OriginalTitle: film.NameEN,
		PosterURL:     film.PosterURL,
		Year:          film.Year,
		Description:   film.Description,
		TotalSeasons:  film.TotalSeries,
		IsSerial:      film.Series || film.IsSerial,
	}

	// Сохраняем в БД для будущих запросов
	s.SaveToDB(ctx, result)

	return &result, nil
}

func (s *SeriesService) SaveToDB(ctx context.Context, params SeriesSearchResult) error {
	_, err := s.seriesRepo.Create(ctx, repository.CreateSeriesParams{
		KinopoiskID:   int32(params.KinopoiskID),
		Title:         params.Title,
		OriginalTitle: pgtype.Text{String: params.OriginalTitle, Valid: params.OriginalTitle != ""},
		PosterUrl:     pgtype.Text{String: params.PosterURL, Valid: params.PosterURL != ""},
		Year:          pgtype.Int4{Int32: int32(params.Year), Valid: params.Year > 0},
		Description:   pgtype.Text{String: params.Description, Valid: params.Description != ""},
		TotalEpisodes: pgtype.Int4{Int32: 0, Valid: false},
		TotalSeasons:  pgtype.Int4{Int32: int32(params.TotalSeasons), Valid: params.TotalSeasons > 0},
		IsSerial:      pgtype.Bool{Bool: params.IsSerial, Valid: true},
	})
	return err
}

func parseYear(s string) int {
	var year int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			year = year*10 + int(c-'0')
			if year > 1000 && year < 10000 {
				return year
			}
		}
	}
	return 0
}
