package config

import "time"

type MediaItem struct {
	Id       string
	Name     string
	Network  string
	Date     time.Time
	Genre    []string
	Language []string
}
