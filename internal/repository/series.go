package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/database/db"
)

type SeriesRepository struct {
	db *db.Queries
}

func NewSeriesRepository(queries *db.Queries) *SeriesRepository {
	return &SeriesRepository{db: queries}
}

type CreateSeriesParams struct {
	KinopoiskID   int32
	Title         string
	OriginalTitle pgtype.Text
	PosterUrl     pgtype.Text
	Year          pgtype.Int4
	Description   pgtype.Text
	TotalEpisodes pgtype.Int4
	TotalSeasons  pgtype.Int4
}

func (r *SeriesRepository) GetByKinopoiskID(ctx context.Context, kinopoiskID int32) (db.Series, error) {
	return r.db.GetSeriesByKinopoiskID(ctx, kinopoiskID)
}

func (r *SeriesRepository) GetByID(ctx context.Context, id pgtype.UUID) (db.Series, error) {
	return r.db.GetSeriesByID(ctx, id)
}

func (r *SeriesRepository) Search(ctx context.Context, query string, limit int32) ([]db.Series, error) {
	if limit <= 0 {
		limit = 20
	}
	return r.db.SearchSeries(ctx, db.SearchSeriesParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
	})
}

func (r *SeriesRepository) Create(ctx context.Context, params CreateSeriesParams) (db.Series, error) {
	return r.db.CreateSeries(ctx, db.CreateSeriesParams{
		KinopoiskID:   params.KinopoiskID,
		Title:         params.Title,
		OriginalTitle: params.OriginalTitle,
		PosterUrl:     params.PosterUrl,
		Year:          params.Year,
		Description:   params.Description,
		TotalEpisodes: params.TotalEpisodes,
		TotalSeasons:  params.TotalSeasons,
	})
}
