package pvr

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/imroc/req"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
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
	timeout          int

	ignoresExpr []*vm.Program
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

type SonarrAddRequest struct {
	Title            string           `json:"title"`
	TitleSlug        string           `json:"titleSlug"`
	Year             int              `json:"year"`
	QualityProfileId int              `json:"qualityProfileId"`
	Images           []string         `json:"images"`
	Tags             []string         `json:"tags"`
	Monitored        bool             `json:"monitored"`
	RootFolderPath   string           `json:"rootFolderPath"`
	AddOptions       SonarrAddOptions `json:"addOptions"`
	Seasons          []string         `json:"seasons"`
	SeriesType       string           `json:"seriesType"`
	SeasonFolder     bool             `json:"seasonFolder"`
	TvdbId           int              `json:"tvdbId"`
}

type SonarrAddOptions struct {
	SearchForMissingEpisodes   bool `json:"searchForMissingEpisodes"`
	IgnoreEpisodesWithFiles    bool `json:"ignoreEpisodesWithFiles"`
	IgnoreEpisodesWithoutFiles bool `json:"ignoreEpisodesWithoutFiles"`
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
		timeout:    pvrDefaultTimeout,
	}
}

/* Private */

func (p *Sonarr) getSystemStatus() (*SonarrSystemStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/system/status"), p.timeout, p.reqHeaders,
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

func (p *Sonarr) compileExpressions() error {
	exprEnv := &config.ExprEnv{}

	// compile ignores
	for _, ignoreExpr := range p.cfg.Filters.Ignores {
		program, err := expr.Compile(ignoreExpr, expr.Env(exprEnv), expr.AsBool())
		if err != nil {
			return errors.Wrapf(err, "failed compiling ignore expression for: %q", ignoreExpr)
		}

		p.ignoresExpr = append(p.ignoresExpr, program)
	}

	return nil
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

	// compile and validate filter expressions
	if err := p.compileExpressions(); err != nil {
		return err
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

func (p *Sonarr) ShouldIgnore(mediaItem *config.MediaItem) (bool, error) {
	exprItem := config.GetExprEnv(mediaItem)

	for _, expression := range p.ignoresExpr {
		result, err := expr.Run(expression, exprItem)
		if err != nil {
			return true, errors.Wrap(err, "failed checking ignore expression")
		}

		expResult, ok := result.(bool)
		if !ok {
			return true, errors.New("failed type asserting ignore expression result")
		}

		if expResult {
			return true, nil
		}
	}

	return false, nil
}

func (p *Sonarr) GetQualityProfileId(profileName string) (int, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/profile"), p.timeout, p.reqHeaders,
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

func (p *Sonarr) AddMedia(item *config.MediaItem) error {
	// convert TvdbId to int
	tvdbId, err := strconv.Atoi(item.TvdbId)
	if err != nil {
		return fmt.Errorf("failed converting tvdb id to int: %q", item.TvdbId)
	}

	// set request params
	params := SonarrAddRequest{
		Title:            item.Title,
		TitleSlug:        item.Slug,
		Year:             item.Year,
		QualityProfileId: p.qualityProfileId,
		Images:           []string{},
		Tags:             []string{},
		Monitored:        true,
		RootFolderPath:   p.cfg.RootFolder,
		AddOptions: SonarrAddOptions{
			SearchForMissingEpisodes:   true,
			IgnoreEpisodesWithFiles:    false,
			IgnoreEpisodesWithoutFiles: false,
		},
		Seasons:      []string{},
		SeriesType:   "standard",
		SeasonFolder: true,
		TvdbId:       tvdbId,
	}

	// send request
	resp, err := web.GetResponse(web.POST, web.JoinURL(p.apiUrl, "/series"), p.timeout, p.reqHeaders,
		req.BodyJSON(params))
	if err != nil {
		return errors.New("failed retrieving add series api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 && resp.Response().StatusCode != 201 {
		return fmt.Errorf("failed retrieving valid add series api response: %s", resp.Response().Status)
	}

	return nil
}

func (p *Sonarr) GetExistingMedia() (map[string]config.MediaItem, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/series"), p.timeout, p.reqHeaders,
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
	existingMediaItems := make(map[string]config.MediaItem)
	itemsSize := 0

	for _, item := range s {
		itemsSize += 1

		itemId := strconv.Itoa(item.TvdbId)
		existingMediaItems[itemId] = config.MediaItem{
			Provider:  "sonarr",
			TvdbId:    itemId,
			Title:     item.Title,
			Date:      time.Time{},
			Genres:    nil,
			Languages: nil,
		}
	}

	p.log.WithField("shows", itemsSize).Info("Retrieved media items")
	return existingMediaItems, nil
}
