package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionService struct {
	redis   *redis.Client
	redisKey string
	maxAge  time.Duration
}

func NewSessionService(redisClient *redis.Client, redisKey string, maxAgeSeconds int) *SessionService {
	return &SessionService{
		redis:    redisClient,
		redisKey: redisKey,
		maxAge:  time.Duration(maxAgeSeconds) * time.Second,
	}
}

func (s *SessionService) Create(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	sessionID := uuid.New()
	key := fmt.Sprintf("%s%s", s.redisKey, sessionID.String())

	if err := s.redis.Set(ctx, key, userID.String(), s.maxAge).Err(); err != nil {
		return uuid.Nil, err
	}

	return sessionID, nil
}

func (s *SessionService) Get(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	key := fmt.Sprintf("%s%s", s.redisKey, sessionID.String())

	userIDStr, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return uuid.Nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	if err := s.redis.Expire(ctx, key, s.maxAge).Err(); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func (s *SessionService) Delete(ctx context.Context, sessionID uuid.UUID) error {
	key := fmt.Sprintf("%s%s", s.redisKey, sessionID.String())
	return s.redis.Del(ctx, key).Err()
}

func (s *SessionService) Refresh(ctx context.Context, sessionID uuid.UUID) error {
	key := fmt.Sprintf("%s%s", s.redisKey, sessionID.String())
	return s.redis.Expire(ctx, key, s.maxAge).Err()
}
