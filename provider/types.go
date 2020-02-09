package provider

type MediaType int

const (
	SHOW MediaType = iota + 1
	MOVIE
)

const (
	SEARCH_TYPE_SCHEDULE string = "schedule"
	SEARCH_TYPE_NOW      string = "now_playing"
)
