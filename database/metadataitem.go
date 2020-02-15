package database

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func GetMetadataItem(provider string, itemId string) (*string, error) {
	var existingItem ProviderItemMetadata

	// does item already exist?
	err := db.First(&existingItem, "provider = ? AND id = ?", provider, itemId).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithMessage(err, "metadata item not found")
	}

	return &existingItem.Json, nil
}

func AddMetadataItem(provider string, itemId string, item interface{}) error {
	// serialize item
	itemJson, err := json.Marshal(item)
	if err != nil {
		return errors.WithMessage(err, "failed marshalling item")
	}

	// insert or update item
	providerItem := ProviderItemMetadata{
		Provider: provider,
		Id:       itemId,
		Json:     string(itemJson),
	}

	if err := db.Where(providerItem).Assign(providerItem).FirstOrCreate(&providerItem).Error; err != nil {
		return errors.WithMessage(err, "failed storing metadata item")
	}

	return nil
}
