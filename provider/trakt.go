package provider

import (
	"fmt"
	"github.com/imroc/req"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/lists"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"strconv"
	"time"
)

/* Struct */

type Trakt struct {
	log               *logrus.Entry
	cfg               map[string]string
	fnAcceptMediaItem func(*config.MediaItem) bool

	apiUrl     string
	apiHeaders req.Header

	reqRatelimit *ratelimit.Limiter
	reqRetry     web.Retry

	genres map[int]string

	supportedShowsSearchTypes  []string
	supportedMoviesSearchTypes []string
}

type TraktMovie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Ids   struct {
		Trakt int    `json:"trakt"`
		Slug  string `json:"slug"`
		Imdb  string `json:"imdb"`
		Tmdb  int    `json:"tmdb"`
	} `json:"ids"`
	Tagline               string   `json:"tagline"`
	Overview              string   `json:"overview"`
	Released              string   `json:"released"`
	Runtime               int      `json:"runtime"`
	Country               string   `json:"country"`
	Trailer               string   `json:"trailer"`
	Homepage              string   `json:"homepage"`
	Status                string   `json:"status"`
	Rating                float64  `json:"rating"`
	Votes                 int      `json:"votes"`
	CommentCount          int      `json:"comment_count"`
	Language              string   `json:"language"`
	AvailableTranslations []string `json:"available_translations"`
	Genres                []string `json:"genres"`
	Certification         string   `json:"certification"`
}

type TraktMoviesResponse []struct {
	Watchers int        `json:"watchers"`
	Movie    TraktMovie `json:"movie"`
}

/* Initializer */

func NewTrakt() *Trakt {
	return &Trakt{
		log:               logger.GetLogger("trakt"),
		cfg:               nil,
		fnAcceptMediaItem: nil,

		apiUrl:     "https://api.trakt.tv",
		apiHeaders: make(req.Header, 0),

		genres: make(map[int]string, 0),

		supportedShowsSearchTypes: []string{},
		supportedMoviesSearchTypes: []string{
			SearchTypeTrending,
		},
	}
}

/* Interface Implements */

func (p *Trakt) Init(mediaType MediaType, cfg map[string]string) error {
	// validate we support this media type
	switch mediaType {
	case Movie:
		break
	default:
		return errors.New("unsupported media type")
	}

	// set provider config
	p.cfg = cfg

	// validate client_id set
	if p.cfg == nil {
		return errors.New("provider has no configuration data set")
	} else if v, err := config.GetProviderSetting(cfg, "client_id"); err != nil {
		return errors.New("provider requires an client_id to be configured")
	} else {
		p.apiHeaders["trakt-api-key"] = *v
	}

	// set ratelimiter
	p.reqRatelimit = web.GetRateLimiter("trakt", 3)

	// set default retry
	p.reqRetry = providerDefaultRetry

	return nil
}

func (p *Trakt) SetAcceptMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnAcceptMediaItem = fn
}

func (p *Trakt) GetShowsSearchTypes() []string {
	return p.supportedShowsSearchTypes
}

func (p *Trakt) SupportsShowsSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedShowsSearchTypes, searchType, false)
}

func (p *Trakt) GetMoviesSearchTypes() []string {
	return p.supportedMoviesSearchTypes
}

func (p *Trakt) SupportsMoviesSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedMoviesSearchTypes, searchType, false)
}

func (p *Trakt) GetShows(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
	return nil, errors.New("unsupported media type")
}

func (p *Trakt) GetMovies(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {

	switch searchType {
	case SearchTypeTrending:
		return p.getMovies("/movies/trending", logic, params)
	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

/* Private - Sub-Implements */

func (p *Trakt) getRequestParams(params map[string]string) req.Param {
	// set request params
	reqParams := req.Param{
		"extended": "full",
	}

	for k, v := range params {
		// skip empty params
		if v == "" {
			continue
		}

		switch k {
		case "country":
			reqParams["region"] = v
		case "language":
			reqParams["language"] = v

		default:
			break
		}
	}

	return reqParams
}

func (p *Trakt) getMovies(endpoint string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
	// set request params
	reqParams := p.getRequestParams(params)

	p.log.Tracef("Request params: %+v", params)

	// parse logic params
	limit := 0
	limitReached := false

	if v := getLogicParam(logic, "limit"); v != nil {
		limit = v.(int)
	}

	// fetch all page results
	mediaItems := make(map[string]config.MediaItem, 0)
	mediaItemsSize := 0
	ignoredItemsSize := 0

	page := 1

	for {
		// set params
		reqParams["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, endpoint), providerDefaultTimeout, p.apiHeaders,
			reqParams, &p.reqRetry, p.reqRatelimit)
		if err != nil {
			return nil, errors.WithMessage(err, "failed retrieving movies api response")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid movies api response: %s", resp.Response().Status)
		}

		// decode response
		var s TraktMoviesResponse
		if err := resp.ToJSON(&s); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding movies api response")
		}

		_ = resp.Response().Body.Close()

		// process response
		for _, item := range s {
			// set movie item
			var movieItem TraktMovie = item.Movie

			// skip this item?
			if movieItem.Ids.Slug == "" {
				continue
			} else if movieItem.Runtime == 0 {
				continue
			} else if movieItem.Released == "" {
				continue
			}

			// does item already exist?
			itemId := strconv.Itoa(movieItem.Ids.Tmdb)
			if _, exists := mediaItems[itemId]; exists {
				continue
			} else if _, exists := mediaItems[movieItem.Ids.Imdb]; exists {
				continue
			}

			// parse item date
			date, err := time.Parse("2006-01-02", movieItem.Released)
			if err != nil {
				p.log.WithError(err).Tracef("Failed parsing release date for item: %+v", item)
				continue
			}

			// init media item
			mediaItem := config.MediaItem{
				Provider:  "trakt",
				TvdbId:    "",
				TmdbId:    itemId,
				ImdbId:    movieItem.Ids.Imdb,
				Title:     movieItem.Title,
				Network:   "",
				Date:      date,
				Year:      date.Year(),
				Runtime:   movieItem.Runtime,
				Genres:    movieItem.Genres,
				Languages: []string{movieItem.Language},
			}

			// media item wanted?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Tracef("Ignoring: %+v", mediaItem)
				ignoredItemsSize += 1
				continue
			} else {
				p.log.Debugf("Accepted: %+v", mediaItem)
			}

			// set media item
			mediaItems[itemId] = mediaItem
			mediaItemsSize += 1

			// stop when limit reached
			if limit > 0 && mediaItemsSize >= limit {
				// limit was supplied via cli and we have reached this limit
				limitReached = true
				break
			}
		}

		// parse pages information
		totalPages := 0
		tmp := resp.Response().Header.Get("X-Pagination-Page-Count")
		if v, err := strconv.Atoi(tmp); err == nil {
			totalPages = v
		}

		p.log.WithFields(logrus.Fields{
			"page":     page,
			"pages":    totalPages,
			"accepted": mediaItemsSize,
			"ignored":  ignoredItemsSize,
		}).Info("Retrieved")

		// loop logic
		if limitReached {
			// the limit has been reached for accepted items
			break
		}

		if page >= totalPages {
			break
		} else {
			page += 1
		}
	}

	p.log.WithField("accepted_items", mediaItemsSize).Info("Retrieved media items")
	return mediaItems, nil
}
