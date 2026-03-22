package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prosergeant/seriesChecker/internal/config"
	"github.com/prosergeant/seriesChecker/internal/database"
	"github.com/prosergeant/seriesChecker/internal/database/sqlc"
	authhandler "github.com/prosergeant/seriesChecker/internal/handler/auth"
	progresshandler "github.com/prosergeant/seriesChecker/internal/handler/progress"
	serieshandler "github.com/prosergeant/seriesChecker/internal/handler/series"
	"github.com/prosergeant/seriesChecker/internal/kinopoisk"
	"github.com/prosergeant/seriesChecker/internal/middleware"
	"github.com/prosergeant/seriesChecker/internal/repository"
	"github.com/prosergeant/seriesChecker/internal/service"
)

var (
	once       sync.Once
	muxHandler http.Handler
)

// Handler is the Vercel Go serverless function entrypoint.
// sync.Once ensures dependencies are initialized exactly once per warm instance.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() { muxHandler = buildHandler() })
	muxHandler.ServeHTTP(w, r)
}

func buildHandler() http.Handler {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cfg := config.Load()

	dbPool, err := pgxpool.New(ctx, config.DatabaseDSN(cfg.Database))
	if err != nil {
		panic("failed to create db pool: " + err.Error())
	}
	if err := dbPool.Ping(ctx); err != nil {
		panic("failed to ping database: " + err.Error())
	}

	queries := sqlc.New(dbPool)

	redisClient, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		panic("failed to connect to redis: " + err.Error())
	}

	kinopoiskClient := kinopoisk.NewClient(cfg.Kinopoisk.APIKey)

	userRepo     := repository.NewUserRepository(queries)
	seriesRepo   := repository.NewSeriesRepository(queries)
	progressRepo := repository.NewUserProgressRepository(queries)

	sessionService  := service.NewSessionService(redisClient.Client, cfg.Session.RedisKey, cfg.Session.MaxAge)
	authService     := service.NewAuthService(userRepo, sessionService, cfg.Session)
	seriesService   := service.NewSeriesService(seriesRepo, kinopoiskClient)
	progressService := service.NewProgressService(progressRepo, seriesRepo, seriesService)

	authH     := authhandler.NewHandler(authService)
	seriesH   := serieshandler.NewHandler(seriesService)
	progressH := progresshandler.NewHandler(progressService)

	protected := middleware.Auth(sessionService, cfg.Session.CookieName)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register",       authH.Register)
	mux.HandleFunc("POST /api/auth/login",           authH.Login)
	mux.HandleFunc("POST /api/auth/logout",          authH.Logout)
	mux.Handle("GET /api/auth/me",                   protected(http.HandlerFunc(authH.Me)))
	mux.HandleFunc("GET /api/series/search",         seriesH.Search)
	mux.HandleFunc("GET /api/series/{id}",           seriesH.GetByID)
	mux.HandleFunc("GET /api/series/{id}/similar",   seriesH.GetSimilar)
	mux.HandleFunc("GET /api/series/{id}/relations", seriesH.GetRelations)
	mux.Handle("GET /api/progress",                  protected(http.HandlerFunc(progressH.GetList)))
	mux.Handle("POST /api/progress",                 protected(http.HandlerFunc(progressH.Update)))
	mux.Handle("DELETE /api/progress/{seriesId}",    protected(http.HandlerFunc(progressH.Delete)))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return middleware.CORS(
		middleware.Recovery(
			middleware.Logger(mux, logger),
			logger,
		),
		cfg.AllowedOrigins,
	)
}
