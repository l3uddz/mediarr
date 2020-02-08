package config

import "time"

type MediaItem struct {
	Provider string
	Id       string
	Name     string
	Network  string
	Date     time.Time
	Genre    []string
	Language []string
}
