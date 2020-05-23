package config

import (
	"fmt"
	"os"

	"github.com/l3uddz/mediarr/logger"
	stringutils "github.com/l3uddz/mediarr/utils/strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Configuration struct {
	Pvr      map[string]*Pvr
	Provider map[string]map[string]string
}

/* Vars */

var (
	// Config exports the config object
	Config *Configuration
	// Internal
	log          = logger.GetLogger("cfg")
	json         = jsoniter.ConfigCompatibleWithStandardLibrary
	cfgFilePath  string
	newOptionLen = 0
)

/* Public */

func (cfg Configuration) ToJsonString() (string, error) {
	c := viper.AllSettings()
	bs, err := json.MarshalIndent(c, "", "  ")
	return string(bs), err
}

func Init(configFilePath string) error {
	cfgFilePath = configFilePath

	/* Initialize Configuration */
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFilePath)

	// read matching env vars
	viper.AutomaticEnv()

	// Load config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			// set the default config to be written
			if err := setConfigDefaults(false); err != nil {
				log.WithError(err).Error("Failed to add config defaults")
				return errors.Wrap(err, "failed adding config defaults")
			}

			// write default config
			if err := viper.WriteConfig(); err != nil {
				log.WithError(err).Fatalf("Failed dumping default configuration to %q", configFilePath)
			}

			log.Infof("Dumped default configuration to %q. Please edit before running again!",
				viper.ConfigFileUsed())
			log.Logger.Exit(0)
		}

		log.WithError(err).Error("Configuration read error")
		return errors.Wrap(err, "failed reading config")
	}

	// Set defaults (checking whether new options were added)
	if err := setConfigDefaults(true); err != nil {
		log.WithError(err).Error("Failed to add new config defaults")
		return errors.Wrap(err, "failed adding new config defaults")
	}

	// Unmarshal into Config struct
	if err := viper.Unmarshal(&Config); err != nil {
		log.WithError(err).Error("Configuration decode error")
		return errors.Wrap(err, "failed decoding config")
	}

	return nil
}

func ShowUsing() {
	log.Infof("Using %s = %q", stringutils.StringLeftJust("CONFIG", " ", 10), cfgFilePath)
}

/* Private */

func setConfigDefault(key string, value interface{}, check bool) int {
	if check {
		if viper.IsSet(key) {
			return 0
		}

		// determine padding to use for new key
		if keyLen := len(key); (keyLen + 2) > newOptionLen {
			newOptionLen = keyLen + 2
		}

		log.Warnf("New config option: %s = %v", stringutils.StringLeftJust(fmt.Sprintf("%q", key),
			" ", newOptionLen), value)
	}

	viper.SetDefault(key, value)

	return 1
}

func setConfigDefaults(check bool) error {
	added := 0

	// pvr settings
	added += setConfigDefault("pvr", map[string]Pvr{}, check)

	// were new settings added?
	if check && added > 0 {
		if err := viper.WriteConfig(); err != nil {
			log.WithError(err).Error("Failed saving configuration with new options...")
			return errors.Wrap(err, "failed saving updated configuration")
		}

		log.Info("Configuration was saved with new options!")
		log.Logger.Exit(0)
	}

	return nil
}
