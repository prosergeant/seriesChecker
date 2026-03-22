package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/pkg/database/sqlc"
)

type SeriesRepository struct {
	sqlc *sqlc.Queries
}

func NewSeriesRepository(queries *sqlc.Queries) *SeriesRepository {
	return &SeriesRepository{sqlc: queries}
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
	IsSerial      pgtype.Bool
}

func (r *SeriesRepository) GetByKinopoiskID(ctx context.Context, kinopoiskID int32) (sqlc.GetSeriesByKinopoiskIDRow, error) {
	return r.sqlc.GetSeriesByKinopoiskID(ctx, kinopoiskID)
}

func (r *SeriesRepository) GetByID(ctx context.Context, id pgtype.UUID) (sqlc.GetSeriesByIDRow, error) {
	return r.sqlc.GetSeriesByID(ctx, id)
}

func (r *SeriesRepository) Search(ctx context.Context, query string, limit int32) ([]sqlc.SearchSeriesRow, error) {
	if limit <= 0 {
		limit = 20
	}
	return r.sqlc.SearchSeries(ctx, sqlc.SearchSeriesParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
	})
}

func (r *SeriesRepository) Create(ctx context.Context, params CreateSeriesParams) (sqlc.CreateSeriesRow, error) {
	return r.sqlc.CreateSeries(ctx, sqlc.CreateSeriesParams{
		KinopoiskID:   params.KinopoiskID,
		Title:         params.Title,
		OriginalTitle: params.OriginalTitle,
		PosterUrl:     params.PosterUrl,
		Year:          params.Year,
		Description:   params.Description,
		TotalEpisodes: params.TotalEpisodes,
		TotalSeasons:  params.TotalSeasons,
		IsSerial:      params.IsSerial,
	})
}
