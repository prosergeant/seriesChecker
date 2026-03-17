package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/database/db"
)

type UserRepository struct {
	db *db.Queries
}

func NewUserRepository(queries *db.Queries) *UserRepository {
	return &UserRepository{db: queries}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (db.User, error) {
	return r.db.GetUserByEmail(ctx, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return r.db.GetUserByID(ctx, id)
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (db.User, error) {
	return r.db.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
}

func (r *UserRepository) Update(ctx context.Context, id pgtype.UUID, email string) (db.User, error) {
	return r.db.UpdateUser(ctx, db.UpdateUserParams{
		ID:    id,
		Email: email,
	})
}
