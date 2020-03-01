package database

type ValidatedProviderItem struct {
	Provider string `gorm:"primary_key"`
	Id       string `gorm:"primary_key"`
}

type ProviderItemMetadata struct {
	Provider string `gorm:"primary_key"`
	Id       string `gorm:"primary_key"`
	Json     string `gorm:"type:text"`
}
