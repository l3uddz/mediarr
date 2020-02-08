package pvr

import "github.com/l3uddz/mediarr/provider"

type Interface interface {
	Init(MediaType) error

	GetQualityProfileId(string) (int, error)
	GetExistingMedia() (map[string]provider.MediaItem, error)
}
