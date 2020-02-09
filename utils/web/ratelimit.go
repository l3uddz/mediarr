package web

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"strings"
	"sync"
)

var (
	rateLimiters map[string]ratelimit.Limiter
	mtx          sync.Mutex
)

func GetRateLimiter(name string, newRateLimit int) *ratelimit.Limiter {
	// acquire lock
	mtx.Lock()
	defer mtx.Unlock()

	// init map
	if rateLimiters == nil {
		rateLimiters = make(map[string]ratelimit.Limiter, 0)
		log.Trace("Initialized rateLimiters map")
	}

	// retrieve or create new ratelimit
	var rl ratelimit.Limiter
	ok := false

	rl, ok = rateLimiters[strings.ToLower(name)]
	if !ok {
		rl = ratelimit.New(newRateLimit)

		log.WithFields(logrus.Fields{
			"name":  name,
			"limit": newRateLimit,
		}).Trace("Created new ratelimit")
	}

	return &rl
}
