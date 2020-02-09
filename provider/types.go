package provider

type MediaType int

const (
	Show MediaType = iota + 1
	Movie
)

const (
	SearchTypeSchedule string = "schedule"
	SearchTypeNow      string = "now_playing"
)
