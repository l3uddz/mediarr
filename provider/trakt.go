package provider

import (
	"fmt"
	"github.com/imroc/req"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/lists"
	"github.com/l3uddz/mediarr/utils/media"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"strconv"
	"time"
)

/* Const */
const (
	TraktRateLimit int = 3
)

/* Struct */

type Trakt struct {
	log                       *logrus.Entry
	cfg                       map[string]string
	fnIgnoreExistingMediaItem func(*config.MediaItem) bool
	fnAcceptMediaItem         func(*config.MediaItem) bool

	apiUrl     string
	apiHeaders req.Header

	reqRatelimit *ratelimit.Limiter
	reqRetry     web.Retry

	genres map[int]string

	supportedShowsSearchTypes  []string
	supportedMoviesSearchTypes []string
}

type TraktMovieIds struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	Imdb  string `json:"imdb"`
	Tmdb  int    `json:"tmdb"`
}

type TraktMovie struct {
	Title                 string        `json:"title"`
	Year                  int           `json:"year"`
	Ids                   TraktMovieIds `json:"ids"`
	Tagline               string        `json:"tagline"`
	Overview              string        `json:"overview"`
	Released              string        `json:"released"`
	Runtime               int           `json:"runtime"`
	Country               string        `json:"country"`
	Trailer               string        `json:"trailer"`
	Homepage              string        `json:"homepage"`
	Status                string        `json:"status"`
	Rating                float64       `json:"rating"`
	Votes                 int           `json:"votes"`
	CommentCount          int           `json:"comment_count"`
	Language              string        `json:"language"`
	AvailableTranslations []string      `json:"available_translations"`
	Genres                []string      `json:"genres"`
	Certification         string        `json:"certification"`
}

type TraktMoviesResponse struct {
	TraktMovie
	Movie *TraktMovie `json:"movie"`
}

type TraktShowIds struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	Tvdb  int    `json:"tvdb"`
	Imdb  string `json:"imdb"`
	Tmdb  int    `json:"tmdb"`
}

type TraktShow struct {
	Title                 string       `json:"title"`
	Year                  int          `json:"year"`
	Ids                   TraktShowIds `json:"ids"`
	Overview              string       `json:"overview"`
	FirstAired            time.Time    `json:"first_aired"`
	Runtime               int          `json:"runtime"`
	Certification         string       `json:"certification"`
	Network               string       `json:"network"`
	Country               string       `json:"country"`
	Trailer               string       `json:"trailer"`
	Homepage              string       `json:"homepage"`
	Status                string       `json:"status"`
	Rating                float64      `json:"rating"`
	Votes                 int          `json:"votes"`
	CommentCount          int          `json:"comment_count"`
	Language              string       `json:"language"`
	AvailableTranslations []string     `json:"available_translations"`
	Genres                []string     `json:"genres"`
	AiredEpisodes         int          `json:"aired_episodes"`
}

type TraktShowsResponse struct {
	TraktShow
	Show *TraktShow `json:"show"`
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

		supportedShowsSearchTypes: []string{
			SearchTypePopular,
			SearchTypeTrending,
			SearchTypeUpcoming,
			SearchTypeWatched,
		},
		supportedMoviesSearchTypes: []string{
			SearchTypeTrending,
			SearchTypeUpcoming,
			SearchTypePopular,
			SearchTypeNow,
			SearchTypeWatched,
		},
	}
}

/* Interface Implements */

func (p *Trakt) Init(mediaType MediaType, cfg map[string]string) error {
	// validate we support this media type
	switch mediaType {
	case Movie, Show:
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
	p.reqRatelimit = web.GetRateLimiter("trakt", TraktRateLimit)

	// set default retry
	p.reqRetry = providerDefaultRetry

	return nil
}

func (p *Trakt) SetIgnoreExistingMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnIgnoreExistingMediaItem = fn
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

	switch searchType {
	case SearchTypePopular:
		return p.getShows("/shows/popular", logic, params)
	case SearchTypeTrending:
		return p.getShows("/shows/trending", logic, params)
	case SearchTypeUpcoming:
		return p.getShows("/shows/anticipated", logic, params)
	case SearchTypeWatched:
		return p.getShows("/shows/watched", logic, params)
	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

func (p *Trakt) GetMovies(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {

	switch searchType {
	case SearchTypePopular:
		return p.getMovies("/movies/popular", logic, params)
	case SearchTypeUpcoming:
		return p.getMovies("/movies/anticipated", logic, params)
	case SearchTypeTrending:
		return p.getMovies("/movies/trending", logic, params)
	case SearchTypeNow:
		return p.getMovies("/movies/boxoffice", logic, params)
	case SearchTypeWatched:
		return p.getMovies("/movies/watched", logic, params)
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
		"limit":    100,
	}

	for k, v := range params {
		// skip empty params
		if v == "" {
			continue
		}

		switch k {
		case "country":
			reqParams["countries"] = v
		case "language":
			reqParams["languages"] = v
		case "genre":
			reqParams["genres"] = v
		case "year":
			reqParams["years"] = v
		case "rating":
			reqParams["ratings"] = v
		case "network":
			reqParams["networks"] = v
		case "status":
			reqParams["status"] = v
		default:
			break
		}
	}

	return reqParams
}

func (p *Trakt) translateMovie(response TraktMoviesResponse) *TraktMovie {
	if response.Movie != nil {
		return response.Movie
	}

	return &TraktMovie{
		Title: response.Title,
		Year:  response.Year,
		Ids: TraktMovieIds{
			Trakt: response.Ids.Trakt,
			Slug:  response.Ids.Slug,
			Imdb:  response.Ids.Imdb,
			Tmdb:  response.Ids.Tmdb,
		},
		Tagline:               response.Tagline,
		Overview:              response.Overview,
		Released:              response.Released,
		Runtime:               response.Runtime,
		Country:               response.Country,
		Trailer:               response.Trailer,
		Homepage:              response.Homepage,
		Status:                response.Status,
		Rating:                response.Rating,
		Votes:                 response.Votes,
		CommentCount:          response.CommentCount,
		Language:              response.Language,
		AvailableTranslations: response.AvailableTranslations,
		Genres:                response.Genres,
		Certification:         response.Certification,
	}
}

func (p *Trakt) translateShow(response TraktShowsResponse) *TraktShow {
	if response.Show != nil {
		return response.Show
	}

	return &TraktShow{
		Title: response.Title,
		Year:  response.Year,
		Ids: TraktShowIds{
			Trakt: response.Ids.Trakt,
			Slug:  response.Ids.Slug,
			Tvdb:  response.Ids.Tvdb,
			Imdb:  response.Ids.Imdb,
			Tmdb:  response.Ids.Tmdb,
		},
		Overview:              response.Overview,
		FirstAired:            response.FirstAired,
		Runtime:               response.Runtime,
		Certification:         response.Certification,
		Network:               response.Network,
		Country:               response.Country,
		Trailer:               response.Trailer,
		Homepage:              response.Homepage,
		Status:                response.Status,
		Rating:                response.Rating,
		Votes:                 response.Votes,
		CommentCount:          response.CommentCount,
		Language:              response.Language,
		AvailableTranslations: response.AvailableTranslations,
		Genres:                response.Genres,
		AiredEpisodes:         response.AiredEpisodes,
	}
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
		var s []TraktMoviesResponse
		if err := resp.ToJSON(&s); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding movies api response")
		}

		_ = resp.Response().Body.Close()

		// process response
		for _, item := range s {
			// set movie item
			var movieItem *TraktMovie = p.translateMovie(item)
			if movieItem == nil {
				p.log.Tracef("Failed translating trakt movie: %#v", item)
				continue
			}

			// skip this item?
			if movieItem.Ids.Slug == "" {
				continue
			} else if movieItem.Ids.Tmdb == 0 {
				continue
			} else if movieItem.Runtime == 0 {
				continue
			} else if movieItem.Released == "" {
				continue
			} else if lists.StringListContains([]string{
				"canceled",
				"rumored",
				"planned",
				"in production",}, movieItem.Status, true) {
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
				Slug:      movieItem.Ids.Slug,
				Title:     movieItem.Title,
				Country:   movieItem.Country,
				Network:   "",
				Date:      date,
				Year:      date.Year(),
				Runtime:   movieItem.Runtime,
				Status:    movieItem.Status,
				Genres:    movieItem.Genres,
				Languages: []string{movieItem.Language},
			}

			// ignore existing media item
			if p.fnIgnoreExistingMediaItem != nil && p.fnIgnoreExistingMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring existing: %+v", mediaItem)
				continue
			}

			// media item wanted?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring: %+v", mediaItem)
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

func (p *Trakt) getShows(endpoint string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
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
			return nil, errors.WithMessage(err, "failed retrieving shows api response")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid shows api response: %s", resp.Response().Status)
		}

		// decode response
		var s []TraktShowsResponse
		if err := resp.ToJSON(&s); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding shows api response")
		}

		_ = resp.Response().Body.Close()

		// process response
		for _, item := range s {
			// set movie item
			var showItem *TraktShow = p.translateShow(item)
			if showItem == nil {
				p.log.Tracef("Failed translating trakt show: %#v", item)
				continue
			}

			// skip this item?
			if showItem.Ids.Slug == "" {
				continue
			} else if showItem.Ids.Tvdb == 0 {
				continue
			} else if showItem.Runtime == 0 {
				continue
			} else if showItem.FirstAired.IsZero() {
				continue
			} else if lists.StringListContains([]string{
				"canceled",
				"planned",
				"in production",}, showItem.Status, true) {
				continue
			}

			// does item already exist?
			itemId := strconv.Itoa(showItem.Ids.Tvdb)
			if _, exists := mediaItems[itemId]; exists {
				continue
			} else if _, exists := mediaItems[showItem.Ids.Imdb]; exists {
				continue
			}

			// init media item
			mediaItem := config.MediaItem{
				Provider:  "trakt",
				TvdbId:    itemId,
				TmdbId:    strconv.Itoa(showItem.Ids.Tmdb),
				ImdbId:    showItem.Ids.Imdb,
				Slug:      showItem.Ids.Slug,
				Title:     showItem.Title,
				Country:   showItem.Country,
				Network:   showItem.Network,
				Date:      showItem.FirstAired,
				Year:      showItem.FirstAired.Year(),
				Runtime:   showItem.Runtime,
				Status:    showItem.Status,
				Genres:    showItem.Genres,
				Languages: []string{showItem.Language},
			}

			// media item wanted?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring: %+v", mediaItem)
				ignoredItemsSize += 1
				continue
			} else if !media.ValidateTvdbId(itemId) {
				p.log.Debugf("Ignoring, bad TvdbId: %+v", mediaItem)
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
