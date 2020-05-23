package provider

import (
	"fmt"
	"strconv"
	"time"

	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/lists"
	"github.com/l3uddz/mediarr/utils/web"

	"github.com/imroc/req"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

/* Const */

const (
	TmdbRateLimit int = 3
)

/* Struct */

type Tmdb struct {
	log                       *logrus.Entry
	cfg                       map[string]string
	fnIgnoreExistingMediaItem func(*config.MediaItem) bool
	fnAcceptMediaItem         func(*config.MediaItem) bool

	apiUrl  string
	apiKey  string
	timeout int

	reqRatelimit *ratelimit.Limiter
	reqRetry     web.Retry

	genres map[int]string

	supportedShowsSearchTypes  []string
	supportedMoviesSearchTypes []string
}

type TmdbGenre struct {
	Id   int
	Name string
}

type TmdbGenreResponse struct {
	Genres []TmdbGenre
}

type TmdbMoviesResponse struct {
	Results []struct {
		Popularity       float64 `json:"popularity"`
		VoteCount        int     `json:"vote_count"`
		Video            bool    `json:"video"`
		PosterPath       string  `json:"poster_path"`
		ID               int     `json:"id"`
		Adult            bool    `json:"adult"`
		BackdropPath     string  `json:"backdrop_path"`
		OriginalLanguage string  `json:"original_language"`
		OriginalTitle    string  `json:"original_title"`
		GenreIds         []int   `json:"genre_ids"`
		Title            string  `json:"title"`
		Overview         string  `json:"overview"`
		ReleaseDate      string  `json:"release_date"`
	} `json:"results"`
	Page         int `json:"page"`
	TotalResults int `json:"total_results"`
	Dates        struct {
		Maximum string `json:"maximum"`
		Minimum string `json:"minimum"`
	} `json:"dates"`
	TotalPages int `json:"total_pages"`
}

type TmdbMovieDetailsResponse struct {
	Adult               bool        `json:"adult"`
	BackdropPath        string      `json:"backdrop_path"`
	BelongsToCollection interface{} `json:"belongs_to_collection"`
	Budget              int         `json:"budget"`
	Genres              []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Homepage            string        `json:"homepage"`
	ID                  int           `json:"id"`
	ImdbID              string        `json:"imdb_id"`
	OriginalLanguage    string        `json:"original_language"`
	OriginalTitle       string        `json:"original_title"`
	Overview            string        `json:"overview"`
	Popularity          float64       `json:"popularity"`
	PosterPath          string        `json:"poster_path"`
	ProductionCompanies []interface{} `json:"production_companies"`
	ProductionCountries []struct {
		Iso31661 string `json:"iso_3166_1"`
		Name     string `json:"name"`
	} `json:"production_countries"`
	ReleaseDate     string `json:"release_date"`
	Revenue         int    `json:"revenue"`
	Runtime         int    `json:"runtime"`
	SpokenLanguages []struct {
		Iso6391 string `json:"iso_639_1"`
		Name    string `json:"name"`
	} `json:"spoken_languages"`
	Status    string `json:"status"`
	Tagline   string `json:"tagline"`
	Title     string `json:"title"`
	Video     bool   `json:"video"`
	VoteCount int    `json:"vote_count"`
}

/* Initializer */

func NewTmdb() *Tmdb {
	return &Tmdb{
		log:               logger.GetLogger("tmdb"),
		cfg:               nil,
		fnAcceptMediaItem: nil,

		apiUrl:  "https://api.themoviedb.org/3",
		apiKey:  "",
		timeout: providerDefaultTimeout,

		genres: make(map[int]string),

		supportedShowsSearchTypes: []string{},
		supportedMoviesSearchTypes: []string{
			SearchTypeNow,
			SearchTypeUpcoming,
			SearchTypePopular,
		},
	}
}

/* Interface Implements */

func (p *Tmdb) Init(mediaType MediaType, cfg map[string]string) error {
	// validate we support this media type
	switch mediaType {
	case Movie:
		break
	default:
		return errors.New("unsupported media type")
	}

	// set provider config
	p.cfg = cfg

	// validate api key set
	if p.cfg == nil {
		return errors.New("provider has no configuration data set")
	} else if v, err := config.GetProviderSetting(cfg, "api_key"); err != nil {
		return errors.New("provider requires an api_key to be configured")
	} else {
		p.apiKey = *v
	}

	// set ratelimiter
	p.reqRatelimit = web.GetRateLimiter("tmdb", TmdbRateLimit)

	// set default retry
	p.reqRetry = providerDefaultRetry

	// load genres
	if err := p.loadGenres(); err != nil {
		return err
	}

	return nil
}

func (p *Tmdb) SetIgnoreExistingMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnIgnoreExistingMediaItem = fn
}

func (p *Tmdb) SetAcceptMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnAcceptMediaItem = fn
}

func (p *Tmdb) GetShowsSearchTypes() []string {
	return p.supportedShowsSearchTypes
}

func (p *Tmdb) SupportsShowsSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedShowsSearchTypes, searchType, false)
}

func (p *Tmdb) GetMoviesSearchTypes() []string {
	return p.supportedMoviesSearchTypes
}

func (p *Tmdb) SupportsMoviesSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedMoviesSearchTypes, searchType, false)
}

func (p *Tmdb) GetShows(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
	return nil, errors.New("unsupported media type")
}

func (p *Tmdb) GetMovies(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {

	switch searchType {
	case SearchTypeNow:
		return p.getMovies("/movie/now_playing", logic, params)
	case SearchTypeUpcoming:
		return p.getMovies("/movie/upcoming", logic, params)
	case SearchTypePopular:
		return p.getMovies("/movie/popular", logic, params)
	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

/* Private - Sub-Implements */

func (p *Tmdb) getRequestParams(params map[string]string) req.Param {
	// set request params
	reqParams := req.Param{
		"api_key": p.apiKey,
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

func (p *Tmdb) loadGenres() error {
	// set request params
	params := req.Param{
		"api_key": p.apiKey,
	}

	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/genre/movie/list"), p.timeout, params,
		&providerDefaultTimeout, p.reqRatelimit)
	if err != nil {
		return errors.WithMessage(err, "failed retrieving genres api response")
	}
	defer web.DrainAndClose(resp.Response().Body)

	// validate response
	if resp.Response().StatusCode != 200 {
		return fmt.Errorf("failed retrieving valid genres api response: %s", resp.Response().Status)
	}

	// decode response
	var s TmdbGenreResponse
	if err := resp.ToJSON(&s); err != nil {
		return errors.WithMessage(err, "failed decoding genres api response")
	}

	// parse response
	for _, genre := range s.Genres {
		p.genres[genre.Id] = genre.Name
	}

	p.log.WithField("genres", len(p.genres)).Info("Retrieved genres")
	return nil
}

//
//func (p *Tmdb) getMovieDetails(tmdbId string) (*TmdbMovieDetailsResponse, error) {
//	// check database for this item
//	existingItemJson, err := database.GetMetadataItem("tmdb", tmdbId)
//	if err == nil && existingItemJson != nil {
//		// item was found in database, unmarshal
//		var n TmdbMovieDetailsResponse
//		if err := json.Unmarshal([]byte(*existingItemJson), &n); err != nil {
//			p.log.WithError(err).Errorf("Failed decoding metadata stored in database for tmdb id: %q", tmdbId)
//		} else {
//			return &n, nil
//		}
//	}
//
//	// set request params
//	params := req.Param{
//		"api_key": p.apiKey,
//	}
//
//	// send request
//	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/movie/"+tmdbId), p.timeout, params,
//		&providerDefaultTimeout, p.reqRatelimit)
//	if err != nil {
//		return nil, errors.WithMessage(err, "failed retrieving movie details api response")
//	}
//	defer web.DrainAndClose(resp.Response().Body)
//
//	// validate response
//	if resp.Response().StatusCode != 200 {
//		return nil, fmt.Errorf("failed retrieving valid movie details api response: %s", resp.Response().Status)
//	}
//
//	// decode response
//	var s TmdbMovieDetailsResponse
//	if err := resp.ToJSON(&s); err != nil {
//		return nil, errors.WithMessage(err, "failed decoding movie details api response")
//	}
//
//	// add item to database
//	if err := database.AddMetadataItem("tmdb", tmdbId, s); err != nil {
//		logrus.WithError(err).Errorf("Failed adding metadata item to database for tmdb id: %q", tmdbId)
//	}
//
//	return &s, nil
//}

func (p *Tmdb) getMovies(endpoint string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
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
	mediaItems := make(map[string]config.MediaItem)
	mediaItemsSize := 0
	ignoredItemsSize := 0
	existingItemsSize := 0

	page := 1

	for {
		// set params
		reqParams["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, endpoint), p.timeout, reqParams, &p.reqRetry,
			p.reqRatelimit)
		if err != nil {
			return nil, errors.WithMessage(err, "failed retrieving movies api response")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			web.DrainAndClose(resp.Response().Body)
			return nil, fmt.Errorf("failed retrieving valid movies api response: %s", resp.Response().Status)
		}

		// decode response
		var s TmdbMoviesResponse
		if err := resp.ToJSON(&s); err != nil {
			web.DrainAndClose(resp.Response().Body)
			return nil, errors.WithMessage(err, "failed decoding movies api response")
		}

		web.DrainAndClose(resp.Response().Body)

		// process response
		for _, item := range s.Results {
			// skip this item?
			if item.Adult || item.Video {
				continue
			}

			// have we already pulled this item?
			itemId := strconv.Itoa(item.ID)
			if _, exists := mediaItems[itemId]; exists {
				continue
			}

			// parse item genres
			var genres []string
			for _, genreId := range item.GenreIds {
				if genreName, exists := p.genres[genreId]; exists {
					genres = append(genres, genreName)
				}
			}

			// parse item date
			date, err := time.Parse("2006-01-02", item.ReleaseDate)
			if err != nil {
				p.log.WithError(err).Tracef("Failed parsing release date for item: %+v", item)
				continue
			}

			// init media item
			mediaItem := config.MediaItem{
				Provider:  "tmdb",
				Endpoint:  endpoint,
				TvdbId:    "",
				TmdbId:    itemId,
				ImdbId:    "",
				Title:     item.Title,
				Summary:   item.Overview,
				Network:   "",
				Date:      date,
				Year:      date.Year(),
				Runtime:   0,
				Genres:    genres,
				Languages: []string{item.OriginalLanguage},
			}

			// does the pvr already have this item?
			if p.fnIgnoreExistingMediaItem != nil && p.fnIgnoreExistingMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring existing: %+v", mediaItem)
				existingItemsSize++
				continue
			}

			// retrieve additional movie details
			//movieDetails, err := p.getMovieDetails(itemId)
			//if err != nil {
			//	// skip this item as it failed tmdb id validation
			//	p.log.Debugf("Ignoring, invalid TmdbId: %+v", mediaItem)
			//	ignoredItemsSize++
			//	continue
			//} else {
			//	// set additional movie details
			//	mediaItem.Runtime = movieDetails.Runtime
			//	mediaItem.ImdbId = movieDetails.ImdbID
			//	mediaItem.Status = movieDetails.Status
			//}

			// item passes ignore expressions?
			if p.fnAcceptMediaItem != nil && !p.fnAcceptMediaItem(&mediaItem) {
				p.log.Debugf("Ignoring: %+v", mediaItem)
				ignoredItemsSize++
				continue
			} else {
				p.log.Debugf("Accepted: %+v", mediaItem)
			}

			// set media item
			mediaItems[itemId] = mediaItem
			mediaItemsSize++

			// stop when limit reached
			if limit > 0 && mediaItemsSize >= limit {
				// limit was supplied via cli and we have reached this limit
				limitReached = true
				break
			}
		}

		p.log.WithFields(logrus.Fields{
			"page":     page,
			"pages":    s.TotalPages,
			"accepted": mediaItemsSize,
			"ignored":  ignoredItemsSize,
			"existing": existingItemsSize,
		}).Info("Retrieved")

		// loop logic
		if limitReached {
			// the limit has been reached for accepted items
			break
		}

		if s.Page >= s.TotalPages {
			break
		} else {
			page++
		}
	}

	p.log.WithField("accepted_items", mediaItemsSize).Info("Retrieved media items")
	return mediaItems, nil
}
