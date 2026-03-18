package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/database/sqlc"
)

type UserRepository struct {
	sqlc *sqlc.Queries
}

func NewUserRepository(queries *sqlc.Queries) *UserRepository {
	return &UserRepository{sqlc: queries}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.sqlc.GetUserByEmail(ctx, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id pgtype.UUID) (sqlc.User, error) {
	return r.sqlc.GetUserByID(ctx, id)
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (sqlc.User, error) {
	return r.sqlc.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
}

func (r *UserRepository) Update(ctx context.Context, id pgtype.UUID, email string) (sqlc.User, error) {
	return r.sqlc.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:    id,
		Email: email,
	})
}
