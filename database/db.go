package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	jsoniter "github.com/json-iterator/go"
	"github.com/l3uddz/mediarr/logger"
	stringutils "github.com/l3uddz/mediarr/utils/strings"
)

var (
	db   *gorm.DB
	log  = logger.GetLogger("db")
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func Init(databaseFilePath string) error {
	// show log
	log.Infof("Using %s = %q", stringutils.StringLeftJust("DATABASE", " ", 10), databaseFilePath)

	// open database
	if dtb, err := gorm.Open("sqlite3", databaseFilePath); err != nil {
		return err
	} else {
		db = dtb
	}

	// migrate schema
	db.AutoMigrate(&ValidatedProviderItem{}, &ProviderItemMetadata{})

	return nil
}

func Close() {
	if err := db.Close(); err != nil {
		log.WithError(err).Error("Failed closing database gracefully...")
	}
}
