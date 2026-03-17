package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Session    SessionConfig
	Kinopoisk  KinopoiskConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	PoolSize int32
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type SessionConfig struct {
	CookieName string
	MaxAge     int
	RedisKey   string
}

type KinopoiskConfig struct {
	APIKey string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "seriestracker"),
			Password: getEnv("DB_PASSWORD", "seriestracker"),
			DBName:   getEnv("DB_NAME", "seriestracker"),
			PoolSize: 20,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Session: SessionConfig{
			CookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
			MaxAge:     60 * 60 * 24 * 7, // 7 days in seconds
			RedisKey:   "session:",
		},
		Kinopoisk: KinopoiskConfig{
			APIKey: getEnv("KINOPOISK_API_KEY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
