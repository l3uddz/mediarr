package pvr

import (
	"github.com/l3uddz/mediarr/config"
)

type Interface interface {
	Init(MediaType) error

	GetQualityProfileId(string) (int, error)
	GetExistingMedia() (map[string]config.MediaItem, error)
}
