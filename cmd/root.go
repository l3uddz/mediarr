package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/l3uddz/mediarr/build"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/database"
	"github.com/l3uddz/mediarr/logger"
	providerObj "github.com/l3uddz/mediarr/provider"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/l3uddz/mediarr/utils/paths"
	stringutils "github.com/l3uddz/mediarr/utils/strings"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	flagLogLevel     = 0
	flagConfigFolder = paths.GetCurrentBinaryPath()
	flagConfigFile   = "config.yaml"
	flagDatabaseFile = "vault.db"
	flagLogFile      = "activity.log"

	flagSearchType string
	flagNoFilter   bool
	flagLimit      int

	flaglistUser string
	flaglistName string
	flagQueryStr string
	flagCountry  string
	flagLanguage string
	flagGenre    string
	flagYear     string
	flagRating   string
	flagNetwork  string
	flagStatus   string
	flagDryRun   bool

	// Global vars
	log *logrus.Entry

	pvrName   string
	pvrConfig *config.Pvr
	pvr       pvrObj.Interface

	existingMediaItems map[string]config.MediaItem

	providerName      string
	lowerProviderName string
	providerCfg       map[string]string
	provider          providerObj.Interface
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mediarr",
	Short: "A CLI application to find new media for the arr sute",
	Long: `A CLI application that can be used to add new media to the arr suite.
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Parse persistent flags
	rootCmd.PersistentFlags().StringVar(&flagConfigFolder, "config-dir", flagConfigFolder, "Config folder")
	rootCmd.PersistentFlags().StringVarP(&flagConfigFile, "config", "c", flagConfigFile, "Config file")
	rootCmd.PersistentFlags().StringVarP(&flagDatabaseFile, "database", "d", flagDatabaseFile, "Database file")
	rootCmd.PersistentFlags().StringVarP(&flagLogFile, "log", "l", flagLogFile, "Log file")
	rootCmd.PersistentFlags().CountVarP(&flagLogLevel, "verbose", "v", "Verbose level")

	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Dry run mode")
}

func initCore() {
	// Set core variables
	if !rootCmd.PersistentFlags().Changed("config") {
		flagConfigFile = filepath.Join(flagConfigFolder, flagConfigFile)
	}
	if !rootCmd.PersistentFlags().Changed("database") {
		flagDatabaseFile = filepath.Join(flagConfigFolder, flagDatabaseFile)
	}
	if !rootCmd.PersistentFlags().Changed("log") {
		flagLogFile = filepath.Join(flagConfigFolder, flagLogFile)
	}

	// Init Logging
	if err := logger.Init(flagLogLevel, flagLogFile); err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	log = logger.GetLogger("app")

	// Init Config
	if err := config.Init(flagConfigFile); err != nil {
		log.WithError(err).Fatal("Failed to initialize config")
	}
}

func showUsing() {
	// version
	log.Infof("Using %s = %s (%s@%s)", stringutils.StringLeftJust("VERSION", " ", 10),
		build.Version, build.GitCommit, build.Timestamp)
	// logging
	logger.ShowUsing()
	// config
	config.ShowUsing()
	// database
	database.ShowUsing(&flagDatabaseFile)

	// separator
	log.Info("------------------")
}

/* Private Helpers */

func parseValidateInputs(args []string) error {
	var ok bool
	var err error

	// validate cli flags
	if flagSearchType == "person" && flagQueryStr == "" {
		return errors.New("person search must have a --query string, e.g. bryan-cranston")
	}

	// validate pvr exists in config
	pvrName = args[0]

	pvrConfig, ok = config.Config.Pvr[pvrName]
	if !ok {
		return fmt.Errorf("no pvr configuration found for: %q", pvrName)
	}

	// set pvr
	pvr, err = pvrObj.Get(pvrName, pvrConfig.Type, pvrConfig)
	if err != nil {
		return errors.WithMessage(err, "failed loading pvr object")
	}

	// set provider
	providerName = args[1]
	lowerProviderName = strings.ToLower(providerName)

	provider, err = providerObj.Get(lowerProviderName)
	if err != nil {
		return errors.WithMessage(err, "failed loading provider object")
	}

	// set provider config if exists
	for pName, pCfg := range config.Config.Provider {
		if strings.EqualFold(pName, providerName) {
			providerCfg = pCfg
			break
		}
	}

	return nil
}

func shouldAcceptMediaItem(mediaItem *config.MediaItem) bool {
	if flagNoFilter {
		// when no-filter is enabled, dont check ignore filters
		return true
	}

	if ignore, err := pvr.ShouldIgnore(mediaItem); err != nil {
		log.WithError(err).Errorf("Failed evaluating ignore expressions against: %+v", mediaItem)
		return false
	} else if ignore {
		return false
	}

	return true
}

func ignoreExistingMediaItem(mediaItem *config.MediaItem) bool {

	if mediaItem.TvdbId != "" {
		if _, exists := existingMediaItems[mediaItem.TvdbId]; exists {
			return true
		}
	}
	if mediaItem.TmdbId != "" {
		if _, exists := existingMediaItems[mediaItem.TmdbId]; exists {
			return true
		}
	}
	if mediaItem.ImdbId != "" {
		if _, exists := existingMediaItems[mediaItem.ImdbId]; exists {
			return true
		}
	}

	return false
}
