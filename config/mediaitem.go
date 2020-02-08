package config

import "time"

type MediaItem struct {
	Provider  string
	TvdbId    string
	TmdbId    string
	ImdbId    string
	Title     string
	Network   string
	Date      time.Time
	Year      int
	Runtime   int
	Genres    []string
	Languages []string
}
