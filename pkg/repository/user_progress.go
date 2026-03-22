package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/pkg/database/sqlc"
)

type UserProgressRepository struct {
	sqlc *sqlc.Queries
}

func NewUserProgressRepository(queries *sqlc.Queries) *UserProgressRepository {
	return &UserProgressRepository{sqlc: queries}
}

type CreateProgressParams struct {
	UserID         pgtype.UUID
	SeriesID       pgtype.UUID
	CurrentSeason  int32
	CurrentEpisode int32
	Status         string
}

type UpdateProgressParams struct {
	UserID         pgtype.UUID
	SeriesID       pgtype.UUID
	CurrentSeason  int32
	CurrentEpisode int32
	Status         string
}

func (r *UserProgressRepository) Get(ctx context.Context, userID, seriesID pgtype.UUID) (sqlc.UserProgress, error) {
	return r.sqlc.GetUserProgress(ctx, sqlc.GetUserProgressParams{
		UserID:   userID,
		SeriesID: seriesID,
	})
}

func (r *UserProgressRepository) GetListByStatus(ctx context.Context, userID pgtype.UUID, status string) ([]sqlc.GetUserProgressListRow, error) {
	return r.sqlc.GetUserProgressList(ctx, sqlc.GetUserProgressListParams{
		UserID: userID,
		Status: status,
	})
}

func (r *UserProgressRepository) GetAll(ctx context.Context, userID pgtype.UUID) ([]sqlc.GetAllUserProgressRow, error) {
	return r.sqlc.GetAllUserProgress(ctx, userID)
}

func (r *UserProgressRepository) Create(ctx context.Context, params CreateProgressParams) (sqlc.UserProgress, error) {
	return r.sqlc.CreateUserProgress(ctx, sqlc.CreateUserProgressParams{
		UserID:         params.UserID,
		SeriesID:       params.SeriesID,
		CurrentSeason:  params.CurrentSeason,
		CurrentEpisode: params.CurrentEpisode,
		Status:         params.Status,
	})
}

func (r *UserProgressRepository) Update(ctx context.Context, params UpdateProgressParams) (sqlc.UserProgress, error) {
	return r.sqlc.UpdateUserProgress(ctx, sqlc.UpdateUserProgressParams{
		UserID:         params.UserID,
		SeriesID:       params.SeriesID,
		CurrentSeason:  params.CurrentSeason,
		CurrentEpisode: params.CurrentEpisode,
		Status:         params.Status,
	})
}

func (r *UserProgressRepository) Delete(ctx context.Context, userID, seriesID pgtype.UUID) error {
	return r.sqlc.DeleteUserProgress(ctx, sqlc.DeleteUserProgressParams{
		UserID:   userID,
		SeriesID: seriesID,
	})
}
