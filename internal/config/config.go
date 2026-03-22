package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Session     SessionConfig
	Kinopoisk   KinopoiskConfig
	AllowedOrigins []string
	NextPublicAPIURL string
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL      string // from DATABASE_URL (Neon/Vercel)
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	PoolSize int32
}

type RedisConfig struct {
	URL      string // from REDIS_URL (Upstash/Vercel)
	Host     string
	Port     string
	Password string
	DB       int
}

type SessionConfig struct {
	CookieName string
	CookieDomain string
	MaxAge     int
	RedisKey   string
}

type KinopoiskConfig struct {
	APIKey string
}

func Load() *Config {
	godotenv.Load()

	originsStr := getEnv("ALLOWED_ORIGINS", "")
	var origins []string
	if originsStr != "" {
		for _, o := range splitComma(originsStr) {
			origins = append(origins, o)
		}
	}

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "seriestracker"),
			Password: getEnv("DB_PASSWORD", "seriestracker"),
			DBName:   getEnv("DB_NAME", "seriestracker"),
			PoolSize: 20,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Session: SessionConfig{
			CookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
			CookieDomain: getEnv("COOKIE_DOMAIN", ""),
			MaxAge:     60 * 60 * 24 * 7, // 7 days in seconds
			RedisKey:   "session:",
		},
		Kinopoisk: KinopoiskConfig{
			APIKey: getEnv("KINOPOISK_API_KEY", ""),
		},
		AllowedOrigins:    origins,
		NextPublicAPIURL: getEnv("NEXT_PUBLIC_API_URL", ""),
	}
}

func splitComma(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// DatabaseDSN returns DATABASE_URL if set, otherwise builds DSN from individual fields.
func DatabaseDSN(cfg DatabaseConfig) string {
	if cfg.URL != "" {
		return cfg.URL
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?pool_max_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.PoolSize,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
