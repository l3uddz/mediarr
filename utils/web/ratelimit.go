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
		rateLimiters = make(map[string]ratelimit.Limiter)
		log.Trace("Initialized rateLimiters map")
	}

	// retrieve or create new ratelimit
	var rl ratelimit.Limiter
	ok := false
	lowerName := strings.ToLower(name)

	rl, ok = rateLimiters[lowerName]
	if !ok {
		rl = ratelimit.New(newRateLimit, ratelimit.WithoutSlack)
		rateLimiters[lowerName] = rl

		log.WithFields(logrus.Fields{
			"name":  name,
			"limit": newRateLimit,
		}).Trace("Created new ratelimit")
	}

	return &rl
}
