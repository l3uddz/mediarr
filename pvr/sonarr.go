package pvr

import (
	"fmt"
	"github.com/imroc/req"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	"github.com/l3uddz/mediarr/provider"
	"github.com/l3uddz/mediarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

/* Structs */

type Sonarr struct {
	cfg              *config.Pvr
	log              *logrus.Entry
	apiUrl           string
	reqHeaders       req.Header
	qualityProfileId int
}

type SonarrSystemStatus struct {
	Version string
}

type SonarrQualityProfiles struct {
	Name string
	Id   int
}

type SonarrSeries struct {
	Title  string
	Status string
	TvdbId int
}

/* Initializer */

func NewSonarr(name string, c *config.Pvr) *Sonarr {
	// set api url
	apiUrl := ""
	if strings.Contains(c.URL, "/api") {
		apiUrl = c.URL
	} else {
		apiUrl = web.JoinURL(c.URL, "/api")
	}

	// set headers
	reqHeaders := req.Header{
		"X-Api-Key": c.ApiKey,
	}

	return &Sonarr{
		cfg:        c,
		log:        logger.GetLogger(name),
		apiUrl:     apiUrl,
		reqHeaders: reqHeaders,
	}
}

/* Private */

func (p *Sonarr) getSystemStatus() (*SonarrSystemStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/system/status"), 15, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving system status api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid system status api response: %s", resp.Response().Status)
	}

	// decode response
	var s SonarrSystemStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding system status api response")
	}

	return &s, nil
}

/* Interface Implements */

func (p *Sonarr) Init(mediaType MediaType) error {
	// validate we support this media type
	switch mediaType {
	case SHOW:
		break
	default:
		return errors.New("unsupported media type")
	}

	// retrieve system status
	status, err := p.getSystemStatus()
	if err != nil {
		return errors.WithMessage(err, "failed initializing sonarr pvr")
	}

	// validate supported version
	switch status.Version[0:1] {
	case "2", "3":
		break
	default:
		return fmt.Errorf("unsupported version of sonarr pvr: %s", status.Version)
	}

	// find quality profile
	if id, err := p.GetQualityProfileId(p.cfg.QualityProfile); err != nil {
		return err
	} else {
		p.qualityProfileId = id

		p.log.WithFields(logrus.Fields{
			"quality_name": p.cfg.QualityProfile,
			"quality_id":   p.qualityProfileId,
		}).Info("Found quality profile")
	}

	return nil
}

func (p *Sonarr) GetQualityProfileId(profileName string) (int, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/profile"), 15, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return 0, errors.New("failed retrieving quality profiles api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return 0, fmt.Errorf("failed retrieving valid quality profiles api response: %s", resp.Response().Status)
	}

	// decode response
	var s []SonarrQualityProfiles
	if err := resp.ToJSON(&s); err != nil {
		return 0, errors.WithMessage(err, "failed decoding quality profiles api response")
	}

	// find quality profile
	for _, profile := range s {
		if strings.EqualFold(profile.Name, profileName) {
			return profile.Id, nil
		}
	}

	return 0, fmt.Errorf("failed finding quality profile: %q", profileName)
}

func (p *Sonarr) GetExistingMedia() (map[string]provider.MediaItem, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/series"), 60, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving series api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid series api response: %s", resp.Response().Status)
	}

	// decode response
	var s []SonarrSeries
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding series api response")
	}

	// parse response
	existingMediaItems := make(map[string]provider.MediaItem, 0)

	for _, item := range s {
		itemId := strconv.Itoa(item.TvdbId)
		existingMediaItems[itemId] = provider.MediaItem{
			Id:       itemId,
			Name:     item.Title,
			Date:     time.Time{},
			Genre:    nil,
			Language: nil,
		}
	}

	p.log.WithField("shows", len(existingMediaItems)).Info("Retrieved existing media")
	return existingMediaItems, nil
}
