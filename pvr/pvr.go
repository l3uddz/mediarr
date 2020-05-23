package pvr

import (
	"fmt"
	"strings"
	"time"

	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/utils/web"

	"github.com/jpillora/backoff"
)

var (
	pvrDefaultTimeout = 120
	pvrDefaultRetry   = web.Retry{
		MaxAttempts: 6,
		RetryableStatusCodes: []int{
			504,
		},
		Backoff: backoff.Backoff{
			Jitter: true,
			Min:    500 * time.Millisecond,
			Max:    10 * time.Second,
		},
	}
)

/* Public */

func Get(pvrName string, pvrType string, pvrConfig *config.Pvr) (Interface, error) {
	switch strings.ToLower(pvrType) {
	case "sonarr":
		return NewSonarr(pvrName, pvrConfig), nil
	case "radarr":
		return NewRadarr(pvrName, pvrConfig), nil
	default:
		break
	}

	return nil, fmt.Errorf("unsupported pvr type provided: %q", pvrType)
}
