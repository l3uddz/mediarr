package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	jsoniter "github.com/json-iterator/go"
	"github.com/l3uddz/mediarr/logger"
	stringutils "github.com/l3uddz/mediarr/utils/strings"
)

var (
	db         *gorm.DB
	log        = logger.GetLogger("db")
	json       = jsoniter.ConfigCompatibleWithStandardLibrary
	dbFilePath string
)

func Init(databaseFilePath string) error {
	dbFilePath = databaseFilePath

	// open database
	var err error
	if db, err = gorm.Open("sqlite3", databaseFilePath); err != nil {
		return err
	}

	// migrate schema
	db.AutoMigrate(&ValidatedProviderItem{}, &ProviderItemMetadata{})

	return nil
}

func ShowUsing(databaseFilePath *string) {
	if databaseFilePath != nil {
		log.Infof("Using %s = %q", stringutils.StringLeftJust("DATABASE", " ", 10),
			*databaseFilePath)
		return
	}

	log.Infof("Using %s = %q", stringutils.StringLeftJust("DATABASE", " ", 10), dbFilePath)
}

func Close() {
	if err := db.Close(); err != nil {
		log.WithError(err).Error("Failed closing database gracefully...")
	}
}
