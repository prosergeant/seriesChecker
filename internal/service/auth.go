package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prosergeant/seriesChecker/internal/config"
	"github.com/prosergeant/seriesChecker/internal/model"
	"github.com/prosergeant/seriesChecker/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound   = errors.New("session not found")
)

type AuthService struct {
	userRepo  *repository.UserRepository
	session   *SessionService
	cfg       config.SessionConfig
}

func NewAuthService(userRepo *repository.UserRepository, session *SessionService, cfg config.SessionConfig) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		session:   session,
		cfg:       cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing.Email != "" {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, email, string(hash))
	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           user.ID.Bytes,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (uuid.UUID, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}

	sessionID, err := s.session.Create(ctx, user.ID.Bytes)
	if err != nil {
		return uuid.Nil, err
	}

	return sessionID, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.session.Delete(ctx, sessionID)
}

func (s *AuthService) GetUserBySession(ctx context.Context, sessionID uuid.UUID) (*model.User, error) {
	userID, err := s.session.Get(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	user, err := s.userRepo.GetByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &model.User{
		ID:           user.ID.Bytes,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}, nil
}

func (s *AuthService) GetConfig() config.SessionConfig {
	return s.cfg
}
