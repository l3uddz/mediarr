package provider

import (
	"fmt"
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
	TvMazeRateLimit int = 2
)

/* Struct */

type TvMaze struct {
	log                       *logrus.Entry
	cfg                       map[string]string
	fnIgnoreExistingMediaItem func(*config.MediaItem) bool
	fnAcceptMediaItem         func(*config.MediaItem) bool

	apiUrl string
	apiKey string

	reqRatelimit *ratelimit.Limiter
	reqRetry     web.Retry

	supportedShowsSearchTypes  []string
	supportedMoviesSearchTypes []string
}

type TvMazeScheduleItem struct {
	ID       int         `json:"id"`
	URL      string      `json:"url"`
	Name     string      `json:"name"`
	Season   int         `json:"season"`
	Number   int         `json:"number"`
	Airdate  string      `json:"airdate"`
	Airtime  string      `json:"airtime"`
	Airstamp time.Time   `json:"airstamp"`
	Runtime  int         `json:"runtime"`
	Image    interface{} `json:"image"`
	Summary  string      `json:"summary"`
	Embedded struct {
		Show struct {
			ID           int      `json:"id"`
			URL          string   `json:"url"`
			Name         string   `json:"name"`
			Type         string   `json:"type"`
			Language     string   `json:"language"`
			Genres       []string `json:"genres"`
			Status       string   `json:"status"`
			Runtime      int      `json:"runtime"`
			Premiered    string   `json:"premiered"`
			OfficialSite string   `json:"officialSite"`
			Schedule     struct {
				Time string   `json:"time"`
				Days []string `json:"days"`
			} `json:"schedule"`
			Rating struct {
				Average interface{} `json:"average"`
			} `json:"rating"`
			Weight  int `json:"weight"`
			Network struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Country struct {
					Name     string `json:"name"`
					Code     string `json:"code"`
					Timezone string `json:"timezone"`
				} `json:"country"`
			} `json:"network"`
			WebChannel interface{} `json:"webChannel"`
			Externals  struct {
				Tvrage  interface{} `json:"tvrage"`
				Thetvdb int         `json:"thetvdb"`
				Imdb    string      `json:"imdb"`
			} `json:"externals"`
			Image struct {
				Medium   string `json:"medium"`
				Original string `json:"original"`
			} `json:"image"`
			Summary string `json:"summary"`
			Updated int    `json:"updated"`
			Links   struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Previousepisode struct {
					Href string `json:"href"`
				} `json:"previousepisode"`
				Nextepisode struct {
					Href string `json:"href"`
				} `json:"nextepisode"`
			} `json:"_links"`
		} `json:"show"`
	} `json:"_embedded"`
}

/* Initializer */

func NewTvMaze() *TvMaze {
	return &TvMaze{
		log:               logger.GetLogger("tvmaze"),
		cfg:               nil,
		fnAcceptMediaItem: nil,

		apiUrl: "http://api.tvmaze.com",
		apiKey: "",

		supportedShowsSearchTypes: []string{
			SearchTypeSchedule,
		},
		supportedMoviesSearchTypes: []string{},
	}
}

/* Interface Implements */

func (p *TvMaze) Init(mediaType MediaType, cfg map[string]string) error {
	// validate we support this media type
	switch mediaType {
	case Show:
		break
	default:
		return errors.New("unsupported media type")
	}

	// set provider config
	p.cfg = cfg

	// set ratelimiter
	p.reqRatelimit = web.GetRateLimiter("tvmaze", TvMazeRateLimit)

	// set default retry
	p.reqRetry = providerDefaultRetry

	return nil
}

func (p *TvMaze) SetIgnoreExistingMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnIgnoreExistingMediaItem = fn
}

func (p *TvMaze) SetAcceptMediaItemFn(fn func(*config.MediaItem) bool) {
	p.fnAcceptMediaItem = fn
}

func (p *TvMaze) GetShowsSearchTypes() []string {
	return p.supportedShowsSearchTypes
}

func (p *TvMaze) SupportsShowsSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedShowsSearchTypes, searchType, false)
}

func (p *TvMaze) GetMoviesSearchTypes() []string {
	return p.supportedMoviesSearchTypes
}

func (p *TvMaze) SupportsMoviesSearchType(searchType string) bool {
	return lists.StringListContains(p.supportedMoviesSearchTypes, searchType, false)
}

func (p *TvMaze) GetShows(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {

	switch searchType {
	case SearchTypeSchedule:
		return p.getScheduleShows(logic, params)
	default:
		break
	}

	return nil, fmt.Errorf("unsupported search_type: %q", searchType)
}

func (p *TvMaze) GetMovies(searchType string, logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
	return nil, errors.New("unsupported media type")
}

/* Private - Sub-Implements */

func (p *TvMaze) getScheduleShows(logic map[string]interface{}, params map[string]string) (map[string]config.MediaItem, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/schedule/full"), providerDefaultTimeout,
		&p.reqRetry, p.reqRatelimit)
	if err != nil {
		return nil, errors.WithMessage(err, "failed retrieving full schedule api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid full schedule api response: %s", resp.Response().Status)
	}

	// decode response
	var s []TvMazeScheduleItem
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding full schedule api response")
	}

	// parse logic params
	limit := 0

	if v := getLogicParam(logic, "limit"); v != nil {
		limit = v.(int)
	}

	// process response
	mediaItems := make(map[string]config.MediaItem, 0)
	mediaItemsSize := 0
	ignoredItemsSize := 0

	for _, item := range s {
		// skip invalid items
		if item.Embedded.Show.Externals.Thetvdb < 1 {
			continue
		}

		itemId := strconv.Itoa(item.Embedded.Show.Externals.Thetvdb)

		// skip non-english language shows
		if item.Embedded.Show.Language != "English" {
			continue
		}

		// does this media item already exist?
		if _, ok := mediaItems[itemId]; ok {
			continue
		}

		// parse premier date
		date, err := time.Parse("2006-01-02", item.Embedded.Show.Premiered)
		if err != nil {
			p.log.WithError(err).Tracef("Failed parsing premier date for item: %+v", item)
			continue
		}

		// init media item
		mediaItem := config.MediaItem{
			Provider:  "tvmaze",
			TvdbId:    itemId,
			ImdbId:    item.Embedded.Show.Externals.Imdb,
			Title:     item.Embedded.Show.Name,
			Network:   item.Embedded.Show.Network.Name,
			Date:      date,
			Year:      date.Year(),
			Runtime:   item.Runtime,
			Languages: []string{item.Embedded.Show.Language},
			Genres:    []string{item.Embedded.Show.Type},
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
		} else if !media.ValidateTvdbId(itemId) {
			p.log.Debugf("Ignoring, bad TvdbId: %+v", mediaItem)
			ignoredItemsSize += 1
			continue
		} else {
			p.log.Debugf("Accepted: %+v", mediaItem)
		}

		// set item
		mediaItems[itemId] = mediaItem
		mediaItemsSize += 1

		// stop when limit reached
		if limit > 0 && mediaItemsSize >= limit {
			// limit was supplied via cli and we have reached this limit
			break
		}
	}

	p.log.WithFields(logrus.Fields{
		"accepted": mediaItemsSize,
		"ignored":  ignoredItemsSize,
	}).Info("Retrieved media items")
	return mediaItems, nil
}
