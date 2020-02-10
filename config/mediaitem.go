package config

import (
	"fmt"
	"time"
)

type MediaItem struct {
	Provider  string
	TvdbId    string
	TmdbId    string
	ImdbId    string
	Slug      string
	Title     string
	Country   string
	Network   string
	Date      time.Time
	Year      int
	Runtime   int
	Genres    []string
	Languages []string
}

type ExprEnv struct {
	MediaItem
	Now func() time.Time
}

/* Public */
func GetExprEnv(media *MediaItem) *ExprEnv {
	return &ExprEnv{
		MediaItem: *media,
		Now:       func() time.Time { return time.Now().UTC() },
	}
}

func (m *MediaItem) String() string {
	return fmt.Sprintf("%s (%d)", m.Title, m.Year)
}
