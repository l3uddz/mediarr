package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/req"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/lists"
	"github.com/l3uddz/mediarr/utils/media"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
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
	timeout    int

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
	Character             string        `json:"character"`
}

type TraktMoviesResponse struct {
	TraktMovie
	Character *string     `json:"character"`
	Movie     *TraktMovie `json:"movie"`
}

type TraktPersonMovieCastResponse struct {
	Cast []TraktMoviesResponse
}

type TraktPersonShowCastResponse struct {
	Cast []TraktShowsResponse
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
	Character             string       `json:"character"`
}

type TraktShowsResponse struct {
	TraktShow
	Character *string    `json:"character"`
	Show      *TraktShow `json:"show"`
}

/* Initializer */

func NewTrakt() *Trakt {
	return &Trakt{
		log:               logger.GetLogger("trakt"),
		cfg:               nil,
		fnAcceptMediaItem: nil,

		apiUrl:     "https://api.trakt.tv",
		apiHeaders: make(req.Header, 0),
		timeout:    providerDefaultTimeout,

		genres: make(map[int]string, 0),

		supportedShowsSearchTypes: []string{
			SearchTypePopular,
			SearchTypeTrending,
			SearchTypeAnticipated,
			SearchTypeWatched,
			SearchTypePlayed,
			SearchTypeCollected,
			SearchTypePerson,
			SearchTypeQuery,
			SearchTypeList,

		},
		supportedMoviesSearchTypes: []string{
			SearchTypeTrending,
			SearchTypeAnticipated,
			SearchTypePopular,
			SearchTypeNow,
			SearchTypeWatched,
			SearchTypePlayed,
			SearchTypeCollected,
			SearchTypePerson,
			SearchTypeQuery,
			SearchTypeList,

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
	case SearchTypeAnticipated:
		return p.getShows("/shows/anticipated", logic, params)
	case SearchTypeWatched, SearchTypePlayed, SearchTypeCollected:
		// get period from query param (default to weekly if not provided)
		period, err := p.getPeriodFromQueryStr(params)
		if err != nil {
			return nil, err
		}

		return p.getShows(fmt.Sprintf("/shows/%s/%s", searchType, period), logic, params)
	case SearchTypePerson:
		queryStr, ok := params["query"]
		if !ok || queryStr == "" {
			return nil, errors.New("person search must have a --query string, e.g. bryan-cranston")
		}

		return p.getShows(fmt.Sprintf("/people/%s/shows", queryStr), logic, params)
	case SearchTypeQuery:
		queryStr, ok := params["query"]
		if !ok || queryStr == "" {
			return nil, errors.New("Query search must have a --query string, e.g. imdb_ratings=5.0-10")
		}

		return p.getShows(fmt.Sprintf("/search/show?query=&%s", queryStr), logic, params)

	case SearchTypeList:
		listUser, ok := params["listuser"]
		if !ok || listUser == "" {
			return nil, errors.New("List search must have a --listuser string, e.g. enormoz")
		}
		listName, ok := params["listname"]
		if !ok || listName == "" {
			return nil, errors.New("List search must have a --listname string, e.g. netflix-movies")
		}

		return p.getShows(fmt.Sprintf("/users/%s/lists/%s/items/shows", listUser, listName), logic, params)

	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

func (p *Trakt) GetMovies(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {

	switch searchType {
	case SearchTypePopular:
		return p.getMovies("/movies/popular", logic, params)
	case SearchTypeAnticipated:
		return p.getMovies("/movies/anticipated", logic, params)
	case SearchTypeTrending:
		return p.getMovies("/movies/trending", logic, params)
	case SearchTypeNow:
		return p.getMovies("/movies/boxoffice", logic, params)
	case SearchTypeWatched, SearchTypePlayed, SearchTypeCollected:
		// get period from query param (default to weekly if not provided)
		period, err := p.getPeriodFromQueryStr(params)
		if err != nil {
			return nil, err
		}
		return p.getMovies(fmt.Sprintf("/movies/%s/%s", searchType, period), logic, params)
	case SearchTypePerson:
		queryStr, ok := params["query"]
		if !ok || queryStr == "" {
			return nil, errors.New("person search must have a --query string, e.g. bryan-cranston")
		}

		return p.getMovies(fmt.Sprintf("/people/%s/movies", queryStr), logic, params)
	case SearchTypeQuery:
		queryStr, ok := params["query"]
		if !ok || queryStr == "" {
			return nil, errors.New("Query search must have a --query string, e.g. imdb_ratings=5.0-10")
		}

		return p.getMovies(fmt.Sprintf("/search/movie?query=&%s", queryStr), logic, params)

	case SearchTypeList:
		listUser, ok := params["listuser"]
		if !ok || listUser == "" {
			return nil, errors.New("List search must have a --listuser string, e.g. enormoz")
		}
		listName, ok := params["listname"]
		if !ok || listName == "" {
			return nil, errors.New("List search must have a --listname string, e.g. netflix-movies")
		}

		return p.getMovies(fmt.Sprintf("/users/%s/lists/%s/items/movies", listUser, listName), logic, params)

	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

/* Private - Sub-Implements */

func (p *Trakt) getPeriodFromQueryStr(params map[string]string) (string, error) {
	queryStr, ok := params["query"]

	if ok && queryStr != "" {
		switch queryStr {
		case "weekly", "monthly", "yearly", "all":
			return queryStr, nil
		default:
			return "", errors.New("watched search defaults to weekly, valid query params: weekly, monthly, yearly, all")
		}
	}

	return "weekly", nil
}

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
		m := response.Movie
		if response.Character != nil && *response.Character != "" {
			m.Character = *response.Character
		}

		return m
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
		Character:             "",
	}
}

func (p *Trakt) translateShow(response TraktShowsResponse) *TraktShow {
	if response.Show != nil {
		s := response.Show
		if response.Character != nil && *response.Character != "" {
			s.Character = *response.Character
		}

		return s
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
	existingItemsSize := 0

	page := 1

	for {
		// set params
		reqParams["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, endpoint), p.timeout, p.apiHeaders, reqParams,
			&p.reqRetry, p.reqRatelimit)
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

		if !strings.Contains(endpoint, "/people/") {
			// non person search
			if err := resp.ToJSON(&s); err != nil {
				_ = resp.Response().Body.Close()
				return nil, errors.WithMessage(err, "failed decoding movies api response")
			}
		} else {
			// person search
			var tmp TraktPersonMovieCastResponse
			if err := resp.ToJSON(&tmp); err != nil {
				_ = resp.Response().Body.Close()
				return nil, errors.WithMessage(err, "failed decoding person movies api response")
			}

			s = tmp.Cast
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
				"in production"}, movieItem.Status, true) {
				continue
			}

			// have we already pulled this item?
			// -- tmdb check
			itemId := strconv.Itoa(movieItem.Ids.Tmdb)
			if _, exists := mediaItems[itemId]; exists {
				continue
			} else if _, exists := mediaItems[itemId]; exists {
				continue
			}

			// -- imdb check
			if movieItem.Ids.Imdb != "" {
				if _, exists := mediaItems[itemId]; exists {
					continue
				}
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
				Endpoint:  endpoint,
				TvdbId:    "",
				TmdbId:    itemId,
				ImdbId:    movieItem.Ids.Imdb,
				Slug:      movieItem.Ids.Slug,
				Title:     movieItem.Title,
				Summary:   movieItem.Overview,
				Country:   []string{movieItem.Country},
				Network:   "",
				Date:      date,
				Year:      date.Year(),
				Runtime:   movieItem.Runtime,
				Status:    movieItem.Status,
				Genres:    movieItem.Genres,
				Languages: []string{movieItem.Language},
				Character: movieItem.Character,
			}

			// does the pvr already have this item?
			if p.fnIgnoreExistingMediaItem != nil && p.fnIgnoreExistingMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring existing: %+v", mediaItem)
				existingItemsSize += 1
				continue
			}

			// item passes ignore expressions?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring: %+v", mediaItem)
				ignoredItemsSize += 1
				continue
			} else if !media.ValidateTmdbId("movie", itemId) {
				p.log.Debugf("Ignoring, invalid TmdbId: %+v", mediaItem)
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
			"existing": existingItemsSize,
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
	existingItemsSize := 0

	page := 1

	for {
		// set params
		reqParams["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, endpoint), p.timeout, p.apiHeaders, reqParams,
			&p.reqRetry, p.reqRatelimit)
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

		if !strings.Contains(endpoint, "/people/") {
			// non person search
			if err := resp.ToJSON(&s); err != nil {
				_ = resp.Response().Body.Close()
				return nil, errors.WithMessage(err, "failed decoding shows api response")
			}
		} else {
			// person search
			var tmp TraktPersonShowCastResponse
			if err := resp.ToJSON(&tmp); err != nil {
				_ = resp.Response().Body.Close()
				return nil, errors.WithMessage(err, "failed decoding person shows api response")
			}

			s = tmp.Cast
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
				"in production"}, showItem.Status, true) {
				continue
			}

			// have we already pulled this item?
			// - tvdb check
			itemId := strconv.Itoa(showItem.Ids.Tvdb)
			if _, exists := mediaItems[itemId]; exists {
				continue
			} else if _, exists := mediaItems[itemId]; exists {
				continue
			}

			// init media item
			mediaItem := config.MediaItem{
				Provider:  "trakt",
				Endpoint:  endpoint,
				TvdbId:    itemId,
				TmdbId:    strconv.Itoa(showItem.Ids.Tmdb),
				ImdbId:    showItem.Ids.Imdb,
				Slug:      showItem.Ids.Slug,
				Title:     showItem.Title,
				Summary:   showItem.Overview,
				Country:   []string{showItem.Country},
				Network:   showItem.Network,
				Date:      showItem.FirstAired,
				Year:      showItem.FirstAired.Year(),
				Runtime:   showItem.Runtime,
				Status:    showItem.Status,
				Genres:    showItem.Genres,
				Languages: []string{showItem.Language},
				Character: showItem.Character,
			}

			// does the pvr already have this item?
			if p.fnIgnoreExistingMediaItem != nil && p.fnIgnoreExistingMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring existing: %+v", mediaItem)
				existingItemsSize += 1
				continue
			}

			// item passes ignore expressions and is a valid tvdb item?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring: %+v", mediaItem)
				ignoredItemsSize += 1
				continue
			} else if !media.ValidateTvdbId(itemId) {
				p.log.Debugf("Ignoring, invalid TvdbId: %+v", mediaItem)
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
			"existing": existingItemsSize,
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
