package pvr

import (
	"github.com/l3uddz/mediarr/config"
)

type Interface interface {
	Init(MediaType) error
	ShouldIgnore(*config.MediaItem) (bool, error)

	GetQualityProfileId(string) (int, error)
	GetExistingMedia() (map[string]config.MediaItem, error)
}
