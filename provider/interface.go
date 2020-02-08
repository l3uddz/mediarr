package provider

type Interface interface {
	Init(MediaType) error
	GetShows() error
}
