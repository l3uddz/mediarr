package provider

import "github.com/l3uddz/mediarr/config"

type Interface interface {
	Init(MediaType, *config.Provider) error

	GetShowsSearchTypes() []string
	GetMoviesSearchTypes() []string
	SupportsShowsSearchType(string) bool
	SupportsMoviesSearchType(string) bool

	GetShows() (map[string]config.MediaItem, error)
	GetMovies() (map[string]config.MediaItem, error)
}
