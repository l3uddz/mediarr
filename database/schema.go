package database

import "time"

type ValidatedProviderItem struct {
	Provider string `gorm:"primary_key"`
	Id       string `gorm:"primary_key"`
	Expires  time.Time
}

type ProviderItemMetadata struct {
	Provider string `gorm:"primary_key"`
	Id       string `gorm:"primary_key"`
	Json     string `gorm:"type:text"`
}
