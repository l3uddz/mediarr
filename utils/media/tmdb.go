package media

import (
	"github.com/l3uddz/mediarr/database"
	"github.com/l3uddz/mediarr/utils/web"
)

func ValidateTmdbId(idType string, tmdbId string) bool {
	// check cache to determine if this item has been validated before
	if database.ExistsValidatedProviderItem("tmdb", tmdbId) {
		return true
	}

	// get ratelimit
	rl := web.GetRateLimiter("tmdb", 3)

	// send request
	resp, err := web.GetResponse(web.GET, "https://www.themoviedb.org/"+idType+"/"+tmdbId, 30, rl)
	if err != nil {
		log.WithError(err).Tracef("Failed retrieving tmdb details for: %q", tmdbId)
		return false
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		log.Tracef("failed retrieving valid tmdb details for %q: %s", tmdbId, resp.Response().Status)
		return false
	}

	// cache that this item is valid
	if err := database.AddValidatedProviderItem("tmdb", tmdbId); err != nil {
		log.WithError(err).Error("Failed storing valid provider item id in database...")
	}

	return true
}
