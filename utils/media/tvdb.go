package media

import (
	"github.com/l3uddz/mediarr/database"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/web"
)

var (
	log = logger.GetLogger("media_utils")
)

func ValidateTvdbId(tvdbId string) bool {
	// check cache to determine if this item has been validated before
	if database.ExistsValidatedProviderItem("tvdb", tvdbId) {
		return true
	}

	// get ratelimit
	rl := web.GetRateLimiter("tvdb", 3)

	// send request
	resp, err := web.GetResponse(web.GET, "https://www.thetvdb.com/dereferrer/series/"+tvdbId, 30, rl)
	if err != nil {
		log.WithError(err).Tracef("Failed retrieving tvdb details for: %q", tvdbId)
		return false
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		log.Tracef("failed retrieving valid tvdb details for %q: %s", tvdbId, resp.Response().Status)
		return false
	}

	// cache that this item is valid
	if err := database.AddValidatedProviderItem("tvdb", tvdbId); err != nil {
		log.WithError(err).Error("Failed storing valid provider item id in database...")
	}

	return true
}
