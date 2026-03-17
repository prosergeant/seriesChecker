package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/database/db"
)

type UserProgressRepository struct {
	db *db.Queries
}

func NewUserProgressRepository(queries *db.Queries) *UserProgressRepository {
	return &UserProgressRepository{db: queries}
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

func (r *UserProgressRepository) Get(ctx context.Context, userID, seriesID pgtype.UUID) (db.UserProgress, error) {
	return r.db.GetUserProgress(ctx, db.GetUserProgressParams{
		UserID:   userID,
		SeriesID: seriesID,
	})
}

func (r *UserProgressRepository) GetListByStatus(ctx context.Context, userID pgtype.UUID, status string) ([]db.GetUserProgressListRow, error) {
	return r.db.GetUserProgressList(ctx, db.GetUserProgressListParams{
		UserID: userID,
		Status: status,
	})
}

func (r *UserProgressRepository) GetAll(ctx context.Context, userID pgtype.UUID) ([]db.GetAllUserProgressRow, error) {
	return r.db.GetAllUserProgress(ctx, userID)
}

func (r *UserProgressRepository) Create(ctx context.Context, params CreateProgressParams) (db.UserProgress, error) {
	return r.db.CreateUserProgress(ctx, db.CreateUserProgressParams{
		UserID:         params.UserID,
		SeriesID:       params.SeriesID,
		CurrentSeason:  params.CurrentSeason,
		CurrentEpisode: params.CurrentEpisode,
		Status:         params.Status,
	})
}

func (r *UserProgressRepository) Update(ctx context.Context, params UpdateProgressParams) (db.UserProgress, error) {
	return r.db.UpdateUserProgress(ctx, db.UpdateUserProgressParams{
		UserID:         params.UserID,
		SeriesID:       params.SeriesID,
		CurrentSeason:  params.CurrentSeason,
		CurrentEpisode: params.CurrentEpisode,
		Status:         params.Status,
	})
}

func (r *UserProgressRepository) Delete(ctx context.Context, userID, seriesID pgtype.UUID) error {
	return r.db.DeleteUserProgress(ctx, db.DeleteUserProgressParams{
		UserID:   userID,
		SeriesID: seriesID,
	})
}
