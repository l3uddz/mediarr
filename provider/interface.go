package provider

import "github.com/l3uddz/mediarr/config"

type Interface interface {
	Init(MediaType, *config.Provider) error

	GetShowsSearchTypes() []string
	GetMoviesSearchTypes() []string
	SupportsShowsSearchType(string) bool
	SupportsMoviesSearchType(string) bool

	GetShows(string, map[string]interface{}, map[string]string) (map[string]config.MediaItem, error)
	GetMovies(string, map[string]interface{}, map[string]string) (map[string]config.MediaItem, error)
}
