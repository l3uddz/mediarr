package provider

import "github.com/l3uddz/mediarr/config"

type Interface interface {
	Init(MediaType, *config.Provider) error
	GetShows() (map[string]config.MediaItem, error)
}
