package cmd

import (
	"fmt"
	"github.com/l3uddz/mediarr/build"
	"github.com/l3uddz/mediarr/config"
	"github.com/l3uddz/mediarr/logger"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/l3uddz/mediarr/utils/paths"
	stringutils "github.com/l3uddz/mediarr/utils/strings"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/atomic"
	"os"
	"path/filepath"
	"strings"
)

var (
	// Global flags
	flagLogLevel     = 0
	flagConfigFolder = paths.GetCurrentBinaryPath()
	flagConfigFile   = "config.yaml"
	flagDatabaseFile = "vault.db"
	flagLogFile      = "activity.log"
	flagRefreshCache = false

	// Global vars
	pvrName         string
	lowerPvrName    string
	pvrConfig       *config.Pvr
	pvr             pvrObj.Interface
	log             *logrus.Entry
	continueRunning *atomic.Bool
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
	cobra.OnInitialize(initConfig)

	// Parse persistent flags
	rootCmd.PersistentFlags().StringVar(&flagConfigFolder, "config-dir", flagConfigFolder, "Config folder")
	rootCmd.PersistentFlags().StringVarP(&flagConfigFile, "config", "c", flagConfigFile, "Config file")
	rootCmd.PersistentFlags().StringVarP(&flagDatabaseFile, "database", "d", flagDatabaseFile, "Database file")
	rootCmd.PersistentFlags().StringVarP(&flagLogFile, "log", "l", flagLogFile, "Log file")
	rootCmd.PersistentFlags().CountVarP(&flagLogLevel, "verbose", "v", "Verbose level")

}

func initConfig() {
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

	log.Infof("Using %s = %s (%s@%s)", stringutils.StringLeftJust("VERSION", " ", 10),
		build.Version, build.GitCommit, build.Timestamp)
	logger.ShowUsing()

	// Init Config
	if err := config.Init(flagConfigFile); err != nil {
		log.WithError(err).Fatal("Failed to initialize config")
	}

	// Init Globals
	continueRunning = atomic.NewBool(true)
}

/* Private Helpers */

func parseValidateInputs(args []string) error {
	var ok bool = false
	var err error = nil

	// validate pvr exists in config
	pvrName = args[0]
	lowerPvrName = strings.ToLower(pvrName)
	pvrConfig, ok = config.Config.Pvr[pvrName]
	if !ok {
		return fmt.Errorf("no pvr configuration found for: %q", pvrName)
	}

	// init pvrObj
	pvr, err = pvrObj.Get(pvrName, pvrConfig.Type, pvrConfig)
	if err != nil {
		return errors.WithMessage(err, "failed loading pvr object")
	}

	return nil
}
