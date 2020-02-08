package provider

import "github.com/l3uddz/mediarr/config"

type Interface interface {
	Init(MediaType) error
	GetShows() (map[string]config.MediaItem, error)
}
