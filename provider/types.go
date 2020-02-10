package provider

type MediaType int

const (
	Show MediaType = iota + 1
	Movie
)

const (
	SearchTypeSchedule string = "schedule"
	SearchTypeNow      string = "now_playing"
	SearchTypeUpcoming string = "upcoming"
	SearchTypePopular  string = "popular"
	SearchTypeTrending string = "trending"
	SearchTypeWatched string = "watched"
)
