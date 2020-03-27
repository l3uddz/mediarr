package database

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func ExistsValidatedProviderItem(provider string, itemId string) bool {
	var existingItem ValidatedProviderItem

	// does item already exist?
	err := db.First(&existingItem, "provider = ? AND id = ?", provider, itemId).Error
	return gorm.IsRecordNotFoundError(err)
}

func AddValidatedProviderItem(provider string, itemId string) error {
	if err := db.FirstOrCreate(&ValidatedProviderItem{
		Provider: provider,
		Id:       itemId,
	}).Error; err != nil {
		return errors.Wrapf(err, "failed creating valid provider item for %q: %q", provider, itemId)
	}

	return nil
}
