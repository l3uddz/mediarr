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

type Tmdb struct {
	log               *logrus.Entry
	cfg               *config.Provider
	fnAcceptMediaItem func(*config.MediaItem) bool

	apiUrl string
	apiKey string

	rl *ratelimit.Limiter

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

type TmdbMoviesNowPlaying struct {
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

/* Initializer */

func NewTmdb() *Tmdb {
	return &Tmdb{
		log:               logger.GetLogger("tmdb"),
		cfg:               nil,
		fnAcceptMediaItem: nil,

		apiUrl: "https://api.themoviedb.org/3",
		apiKey: "",
		genres: make(map[int]string, 0),

		supportedShowsSearchTypes: []string{},
		supportedMoviesSearchTypes: []string{
			SearchTypeNow,
			SearchTypeUpcoming,
			SearchTypePopular,
		},
	}
}

/* Interface Implements */

func (p *Tmdb) Init(mediaType MediaType, cfg *config.Provider) error {
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
	if p.cfg == nil || p.cfg.ApiKey == "" {
		return errors.New("provider requires an api_key to be configured")
	} else {
		p.apiKey = p.cfg.ApiKey
	}

	// set ratelimiter
	p.rl = web.GetRateLimiter("tmdb", 3)

	// load genres
	if err := p.loadGenres(); err != nil {
		return err
	}

	return nil
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
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/genre/movie/list"), providerDefaultTimeout,
		params, &providerDefaultTimeout, p.rl)
	if err != nil {
		return errors.WithMessage(err, "failed retrieving genres api response")
	}
	defer resp.Response().Body.Close()

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
	mediaItems := make(map[string]config.MediaItem, 0)
	mediaItemsSize := 0
	ignoredItemsSize := 0

	page := 1

	for {
		// set params
		reqParams["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, endpoint), providerDefaultTimeout, reqParams,
			&providerDefaultRetry, p.rl)
		if err != nil {
			return nil, errors.WithMessage(err, "failed retrieving movies api response")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid movies api response: %s", resp.Response().Status)
		}

		// decode response
		var s TmdbMoviesNowPlaying
		if err := resp.ToJSON(&s); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding movies api response")
		}

		_ = resp.Response().Body.Close()

		// process response
		for _, item := range s.Results {
			// skip this item?
			if item.Adult || item.Video {
				continue
			}

			// does item already exist?
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
				TvdbId:    "",
				TmdbId:    itemId,
				ImdbId:    "",
				Title:     item.Title,
				Network:   "",
				Date:      date,
				Year:      date.Year(),
				Runtime:   0,
				Genres:    genres,
				Languages: []string{item.OriginalLanguage},
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

		p.log.WithFields(logrus.Fields{
			"page":     page,
			"pages":    s.TotalPages,
			"accepted": mediaItemsSize,
			"ignored":  ignoredItemsSize,
		}).Info("Retrieved")

		// loop logic
		if limitReached {
			// the limit has been reached for accepted items
			break
		}

		if s.Page >= s.TotalPages {
			break
		} else {
			page += 1
		}
	}

	p.log.WithField("accepted_items", mediaItemsSize).Info("Retrieved media items")
	return mediaItems, nil
}
