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

type Radarr struct {
	cfg              *config.Pvr
	log              *logrus.Entry
	apiUrl           string
	reqHeaders       req.Header
	qualityProfileId int

	ignoresExpr []*vm.Program
	acceptsExpr []*vm.Program
}

type RadarrSystemStatus struct {
	Version string
}

type RadarrQualityProfiles struct {
	Name string
	Id   int
}

type RadarrMovies struct {
	Title  string
	Status string
	ImdbId string
	TmdbId int
}

/* Initializer */

func NewRadarr(name string, c *config.Pvr) *Radarr {
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

	return &Radarr{
		cfg:        c,
		log:        logger.GetLogger(name),
		apiUrl:     apiUrl,
		reqHeaders: reqHeaders,
	}
}

/* Private */

func (p *Radarr) getSystemStatus() (*RadarrSystemStatus, error) {
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
	var s RadarrSystemStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding system status api response")
	}

	return &s, nil
}

func (p *Radarr) compileExpressions() error {
	exprEnv := &config.MediaItem{}

	// compile ignores
	for _, ignoreExpr := range p.cfg.Filters.Ignores {
		program, err := expr.Compile(ignoreExpr, expr.Env(exprEnv), expr.AsBool())
		if err != nil {
			return errors.Wrapf(err, "failed compiling ignore expression for: %q", ignoreExpr)
		}

		p.ignoresExpr = append(p.ignoresExpr, program)
	}

	// compile accepts
	for _, acceptExpr := range p.cfg.Filters.Accepts {
		program, err := expr.Compile(acceptExpr, expr.Env(exprEnv), expr.AsBool())
		if err != nil {
			return errors.Wrapf(err, "failed compiling accept expression for: %q", acceptExpr)
		}

		p.acceptsExpr = append(p.acceptsExpr, program)
	}

	return nil
}

/* Interface Implements */

func (p *Radarr) Init(mediaType MediaType) error {
	// validate we support this media type
	switch mediaType {
	case MOVIE:
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
		return errors.WithMessage(err, "failed initializing radarr pvr")
	}

	// validate supported version
	switch status.Version[0:3] {
	case "0.2", "3.0":
		break
	default:
		return fmt.Errorf("unsupported version of radarr pvr: %s", status.Version)
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

func (p *Radarr) ShouldIgnore(mediaItem *config.MediaItem) (bool, error) {
	for _, expression := range p.ignoresExpr {
		result, err := expr.Run(expression, mediaItem)
		if err != nil {
			return true, errors.Wrap(err, "failed checking ignore expression")
		}

		expResult, ok := result.(bool)
		if !ok {
			return true, errors.New("failed type asserting ignore expression result")
		}

		if expResult {
			p.log.Tracef("Ignoring: %+v", mediaItem)
			return true, nil
		}
	}

	return false, nil
}

func (p *Radarr) ShouldAccept(mediaItem *config.MediaItem) (bool, error) {
	for _, expression := range p.acceptsExpr {
		result, err := expr.Run(expression, mediaItem)
		if err != nil {
			return false, errors.Wrapf(err, "failed checking accept expression")
		}

		expResult, ok := result.(bool)
		if !ok {
			return false, errors.New("failed type asserting accept expression result")
		}

		if expResult {
			return true, nil
		}
	}

	return false, nil
}

func (p *Radarr) GetQualityProfileId(profileName string) (int, error) {
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
	var s []RadarrQualityProfiles
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

func (p *Radarr) GetExistingMedia() (map[string]config.MediaItem, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/movies"), 60, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving movies api response")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid movies api response: %s", resp.Response().Status)
	}

	// decode response
	var s []RadarrMovies
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding movies api response")
	}

	// parse response
	existingMediaItems := make(map[string]config.MediaItem, 0)
	itemsSize := 0

	for _, item := range s {
		added := false

		if item.ImdbId != "" {
			existingMediaItems[item.ImdbId] = config.MediaItem{
				Provider: "radarr",
				Id:       item.ImdbId,
				Name:     item.Title,
				Date:     time.Time{},
				Genre:    nil,
				Language: nil,
			}

			added = true
		}

		if item.TmdbId > 0 {
			tmdbId := strconv.Itoa(item.TmdbId)
			existingMediaItems[tmdbId] = config.MediaItem{
				Provider: "radarr",
				Id:       tmdbId,
				Name:     item.Title,
				Date:     time.Time{},
				Genre:    nil,
				Language: nil,
			}

			added = true
		}

		if added {
			itemsSize += 1
		}
	}

	p.log.WithField("movies", itemsSize).Info("Retrieved media items")
	return existingMediaItems, nil
}
