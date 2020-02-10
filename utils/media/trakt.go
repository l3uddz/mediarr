package media

import (
	"fmt"
	"github.com/imroc/req"
	"github.com/jpillora/backoff"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"time"
)

type TraktSearchResponse []struct {
	Type  string `json:"type"`
	Score int    `json:"score"`
	Movie *struct {
		Title string `json:"title"`
		Year  int    `json:"year"`
		Ids   struct {
			Trakt int    `json:"trakt"`
			Slug  string `json:"slug"`
			Imdb  string `json:"imdb"`
			Tmdb  int    `json:"tmdb"`
		} `json:"ids"`
	} `json:"movie"`
	Show *struct {
		Title string `json:"title"`
		Year  int    `json:"year"`
		Ids   struct {
			Trakt  int         `json:"trakt"`
			Slug   string      `json:"slug"`
			Tvdb   int         `json:"tvdb"`
			Imdb   string      `json:"imdb"`
			Tmdb   int         `json:"tmdb"`
			Tvrage interface{} `json:"tvrage"`
		} `json:"ids"`
	} `json:"show"`
}

type TraktSearchType string

const (
	TraktClientId string = "7eb1023eff72ac4d130e1fb46ae2741fe0f5fd39c367b74c8f37285e09aff23d"

	Tmdb TraktSearchType = "tmdb"
	Tvdb                 = "tvdb"
)

func LookupTraktId(mediaType string, providerType TraktSearchType, searchId string) (int, error) {
	// set request details
	reqLimit := web.GetRateLimiter("trakt", 3)
	reqHeader := req.Header{
		"trakt-api-key": TraktClientId,
	}
	reqRetry := web.Retry{
		MaxAttempts:          5,
		RetryableStatusCodes: []int{},
		Backoff: backoff.Backoff{
			Jitter: true,
			Min:    1 * time.Second,
			Max:    5 * time.Second,
		},
	}

	searchUrl := fmt.Sprintf("https://api.trakt.tv/search/%s/%s?type=%s", providerType, searchId, mediaType)

	// send request
	resp, err := web.GetResponse(web.GET, searchUrl, 15, reqHeader, &reqRetry, reqLimit)
	if err != nil {
		return 0, errors.WithMessagef(err, "failed retrieving trakt %s search response for: %q", mediaType, searchId)
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return 0, errors.WithMessagef(err, "failed validating trakt %s search response for %q: %s",
			mediaType, searchId, resp.Response().Status)
	}

	// decode response
	var s TraktSearchResponse
	if err := resp.ToJSON(&s); err != nil {
		return 0, errors.WithMessagef(err, "failed decoding trakt %s search response for: %q", mediaType, searchId)
	}

	if len(s) == 0 {
		return 0, fmt.Errorf("failed finding trakt %s item with %s id: %q", mediaType, providerType, searchId)
	}

	// validate trakt id retrieved
	traktId := 0
	switch mediaType {
	case "movie":
		if s[0].Movie == nil {
			return 0, errors.New("failed parsing trakt movie search result")
		}
		traktId = s[0].Movie.Ids.Trakt
	case "show":
		if s[0].Show == nil {
			return 0, errors.New("failed parsing trakt show search result")
		}
		traktId = s[0].Show.Ids.Trakt
	default:
		return 0, errors.New("unknown media type")
	}

	return traktId, nil
}
