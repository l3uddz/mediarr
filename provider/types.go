package provider

type MediaType int

const (
	Show MediaType = iota + 1
	Movie
)

const (
	SearchTypeSchedule    string = "schedule"
	SearchTypeNow                = "now_playing"
	SearchTypeUpcoming           = "upcoming"
	SearchTypeAnticipated        = "anticipated"
	SearchTypePopular            = "popular"
	SearchTypeTrending           = "trending"
	SearchTypeWatched            = "watched"
	SearchTypePlayed             = "played"
	SearchTypeCollected          = "collected"
	SearchTypePerson             = "person"
	SearchTypeQuery              = "query"
)
