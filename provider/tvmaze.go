package provider

import (
	"github.com/pkg/errors"
)

/* Struct */

type TvMaze struct {
	apiUrl string
	apiKey string
}

/* Initializer */

func NewTvMaze() *TvMaze {
	return &TvMaze{
		apiUrl: "http://api.tvmaze.com",
		apiKey: "",
	}
}

/* Interface Implements */

func (p *TvMaze) Init(mediaType MediaType) error {
	// validate we support this media type
	switch mediaType {
	case SHOW:
		break
	default:
		return errors.New("unsupported media type")
	}

	return nil
}
