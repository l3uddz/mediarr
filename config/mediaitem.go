package config

import "time"

type MediaItem struct {
	Provider  string
	TvdbId    string
	TmdbId    string
	ImdbId    string
	Slug      string
	Title     string
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

func GetExprEnv(media *MediaItem) *ExprEnv {
	return &ExprEnv{
		MediaItem: *media,
		Now:       func() time.Time { return time.Now().UTC() },
	}
}
