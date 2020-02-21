package provider

import (
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/l3uddz/mediarr/utils/web"
	"strings"
	"time"
)

var (
	providerDefaultTimeout = 30
	providerDefaultRetry   = web.Retry{
		MaxAttempts:          6,
		RetryableStatusCodes: []int{},
		Backoff: backoff.Backoff{
			Jitter: true,
			Min:    500 * time.Millisecond,
			Max:    10 * time.Second,
		},
	}
)

/* Public */

func Get(providerType string) (Interface, error) {
	switch strings.ToLower(providerType) {
	case "tvmaze":
		return NewTvMaze(), nil
	case "tmdb":
		return NewTmdb(), nil
	case "trakt":
		return NewTrakt(), nil
	default:
		break
	}

	return nil, fmt.Errorf("unsupported media provider provided: %q", providerType)
}
