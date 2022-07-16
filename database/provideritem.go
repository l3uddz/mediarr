package database

import (
	"time"

	"github.com/pkg/errors"
)

func ExistsValidatedProviderItem(provider string, itemId string) bool {
	var existingItem ValidatedProviderItem

	// does item already exist?
	err := db.First(&existingItem, "provider = ? AND id = ?", provider, itemId).Error

	switch {
	case err == nil && (existingItem.Expires.IsZero() || existingItem.Expires.Before(time.Now().UTC())):
		// the item has expired
		if err := db.Delete(&existingItem).Error; err != nil {
			log.WithError(err).Errorf("Failed removing expired provider item for %q: %q", provider, itemId)
		}

		return false
	}

	return err == nil
}

func AddValidatedProviderItem(provider string, itemId string) error {
	if err := db.FirstOrCreate(&ValidatedProviderItem{
		Provider: provider,
		Id:       itemId,
		Expires:  time.Now().UTC().Add(time.Hour * 168),
	}).Error; err != nil {
		return errors.Wrapf(err, "failed creating valid provider item for %q: %q", provider, itemId)
	}
	return nil
}
