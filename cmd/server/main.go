package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prosergeant/seriesChecker/internal/config"
	"github.com/prosergeant/seriesChecker/internal/database"
	"github.com/prosergeant/seriesChecker/internal/database/sqlc"
	"github.com/prosergeant/seriesChecker/internal/handler/auth"
	"github.com/prosergeant/seriesChecker/internal/handler/progress"
	"github.com/prosergeant/seriesChecker/internal/handler/series"
	"github.com/prosergeant/seriesChecker/internal/kinopoisk"
	"github.com/prosergeant/seriesChecker/internal/middleware"
	"github.com/prosergeant/seriesChecker/internal/repository"
	"github.com/prosergeant/seriesChecker/internal/service"
)

const nextJSURL = "http://localhost:3000"

func main() {
	ctx := context.Background()

	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?pool_max_conns=%d",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.PoolSize,
	)

	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("failed to create db pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	queries := sqlc.New(dbPool)

	redisClient, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	kinopoiskClient := kinopoisk.NewClient(cfg.Kinopoisk.APIKey)

	userRepo := repository.NewUserRepository(queries)
	seriesRepo := repository.NewSeriesRepository(queries)
	progressRepo := repository.NewUserProgressRepository(queries)
	// работает с редисом
	sessionService := service.NewSessionService(redisClient.Client, cfg.Session.RedisKey, cfg.Session.MaxAge)
	// работает с репозиторием пользователя и сессий
	authService := service.NewAuthService(userRepo, sessionService, cfg.Session)
	// работает с клиентом кинопоиска и репозиторием серий
	seriesService := service.NewSeriesService(seriesRepo, kinopoiskClient)
	// работает с репозиторием прогресса
	progressService := service.NewProgressService(progressRepo, seriesRepo, seriesService)

	authHandler := auth.NewHandler(authService)
	seriesHandler := series.NewHandler(seriesService)
	progressHandler := progress.NewHandler(progressService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)

	protected := middleware.Auth(sessionService, cfg.Session.CookieName)
	mux.Handle("GET /api/auth/me", protected(http.HandlerFunc(authHandler.Me)))

	mux.HandleFunc("GET /api/series/search", seriesHandler.Search)
	mux.HandleFunc("GET /api/series/{id}", seriesHandler.GetByID)

	mux.Handle("GET /api/progress", protected(http.HandlerFunc(progressHandler.GetList)))
	mux.Handle("POST /api/progress", protected(http.HandlerFunc(progressHandler.Update)))
	mux.Handle("DELETE /api/progress/{seriesId}", protected(http.HandlerFunc(progressHandler.Delete)))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.CORS(
		middleware.Recovery(
			middleware.Logger(mux, logger),
			logger,
		),
		cfg.AllowedOrigins,
	)

	nextProxy, err := url.Parse(nextJSURL)
	if err != nil {
		logger.Error("failed to parse next.js url", "error", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(nextProxy)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("server starting", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited")
}
