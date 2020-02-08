package provider

import (
	"fmt"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

/* Struct */

type TvMaze struct {
	log *logrus.Entry

	apiUrl string
	apiKey string
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
		log:    logger.GetLogger("tvmaze"),
		apiUrl: "http://api.tvmaze.com",
		apiKey: "",
	}
}

/* Interface Implements */

func (p *TvMaze) Init(mediaType MediaType) error {
	// validate we support this media type
	switch mediaType {
	case SHOW:
		break
	default:
		return errors.New("unsupported media type")
	}

	return nil
}

func (p *TvMaze) GetShows() error {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/schedule/full"), providerDefaultTimeout,
		&providerDefaultRetry)
	if err != nil {
		return errors.New("failed retrieving full schedule api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return fmt.Errorf("failed retrieving valid full schedule api response: %s", resp.Response().Status)
	}

	// decode response
	var s []TvMazeScheduleItem
	if err := resp.ToJSON(&s); err != nil {
		return errors.WithMessage(err, "failed decoding full schedule api response")
	}

	// process response
	mediaItems := make(map[string]MediaItem, 0)

	for _, item := range s {
		itemId := strconv.Itoa(item.Embedded.Show.Externals.Thetvdb)

		// skip non-english language shows
		if item.Embedded.Show.Language != "English" {
			continue
		}

		// does this media item already exist?
		if _, ok := mediaItems[itemId]; ok {
			continue
		}

		// add item
		mediaItems[itemId] = MediaItem{
			Id:       itemId,
			Name:     item.Embedded.Show.Name,
			Date:     item.Embedded.Show.Premiered,
			Language: []string{item.Embedded.Show.Language},
			Genre:    []string{item.Embedded.Show.Type},
		}
	}

	p.log.Info(mediaItems)
	p.log.WithField("shows", len(mediaItems)).Info("Found shows")
	return nil
}
