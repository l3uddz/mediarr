package provider

type MediaType int

const (
	Show MediaType = iota + 1
	Movie
)

const (
	SearchTypeSchedule    string = "schedule"
	SearchTypeNow         string = "now_playing"
	SearchTypeUpcoming    string = "upcoming"
	SearchTypeAnticipated string = "anticipated"
	SearchTypePopular     string = "popular"
	SearchTypeTrending    string = "trending"
	SearchTypeWatched     string = "watched"
	SearchTypePlayed      string = "played"
	SearchTypeCollected   string = "collected"
	SearchTypePerson      string = "person"
	SearchTypeQuery       string = "query"
	SearchTypeList        string = "list"
)
