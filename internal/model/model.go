package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Series struct {
	ID             uuid.UUID `json:"id"`
	KinopoiskID    int       `json:"kinopoisk_id"`
	Title          string    `json:"title"`
	OriginalTitle  *string   `json:"original_title,omitempty"`
	PosterURL      *string   `json:"poster_url,omitempty"`
	Year           *int      `json:"year,omitempty"`
	Description    *string   `json:"description,omitempty"`
	TotalEpisodes  *int      `json:"total_episodes,omitempty"`
	TotalSeasons   *int      `json:"total_seasons,omitempty"`
	IsSerial       bool      `json:"is_serial"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserProgress struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	SeriesID       uuid.UUID `json:"series_id"`
	CurrentSeason  int       `json:"current_season"`
	CurrentEpisode int       `json:"current_episode"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProgressStatus string

const (
	StatusWatching ProgressStatus = "watching"
	StatusCompleted ProgressStatus = "completed"
	StatusPlanned  ProgressStatus = "planned"
	StatusDropped  ProgressStatus = "dropped"
	StatusOnHold   ProgressStatus = "on_hold"
)

func (p ProgressStatus) String() string {
	return string(p)
}

var ValidStatuses = []ProgressStatus{
	StatusWatching,
	StatusCompleted,
	StatusPlanned,
	StatusDropped,
	StatusOnHold,
}
