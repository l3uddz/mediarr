package database

type ValidatedProviderItem struct {
	Provider string `gorm:"primary_key"`
	Id       string `gorm:"primary_key"`
}
