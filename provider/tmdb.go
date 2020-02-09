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
)

/* Struct */

type Tmdb struct {
	log *logrus.Entry
	cfg *config.Provider

	apiUrl string
	apiKey string

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

/* Initializer */

func NewTmdb() *Tmdb {
	return &Tmdb{
		log:    logger.GetLogger("tmdb"),
		cfg:    nil,
		apiUrl: "https://api.themoviedb.org/3",
		apiKey: "",
		genres: make(map[int]string, 0),

		supportedShowsSearchTypes: []string{},
		supportedMoviesSearchTypes: []string{
			SEARCH_TYPE_NOW,
		},
	}
}

/* Private */

func (p *Tmdb) loadGenres() error {
	// set request params
	params := req.Param{
		"api_key": p.apiKey,
	}

	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/genre/movie/list"), providerDefaultTimeout,
		params, &providerDefaultTimeout)
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

/* Interface Implements */

func (p *Tmdb) Init(mediaType MediaType, cfg *config.Provider) error {
	// validate we support this media type
	switch mediaType {
	case MOVIE:
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

	// load genres
	if err := p.loadGenres(); err != nil {
		return err
	}

	return nil
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

func (p *Tmdb) GetShows() (map[string]config.MediaItem, error) {
	return nil, errors.New("unsupported media type")
}

func (p *Tmdb) GetMovies() (map[string]config.MediaItem, error) {
	return nil, errors.New("unsupported media type")
}
