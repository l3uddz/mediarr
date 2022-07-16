package database

import (
	"github.com/glebarez/sqlite"
	jsoniter "github.com/json-iterator/go"
	"gorm.io/gorm"
	gcl "gorm.io/gorm/logger"

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

	// prepare gorm config
	gc := &gorm.Config{
		PrepareStmt: true,
	}
	gc.Logger = gcl.Default.LogMode(gcl.Silent)

	// open database
	var err error
	if db, err = gorm.Open(sqlite.Open(databaseFilePath), gc); err != nil {
		return err
	}

	// migrate schema
	return db.AutoMigrate(&ValidatedProviderItem{}, &ProviderItemMetadata{})
}

func ShowUsing(databaseFilePath *string) {
	if databaseFilePath != nil {
		log.Infof("Using %s = %q", stringutils.StringLeftJust("DATABASE", " ", 10),
			*databaseFilePath)
		return
	}

	log.Infof("Using %s = %q", stringutils.StringLeftJust("DATABASE", " ", 10), dbFilePath)
}
