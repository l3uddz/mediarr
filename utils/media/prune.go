package media

import "github.com/l3uddz/mediarr/config"

func PruneExistingMedia(pvrMediaItems map[string]config.MediaItem, providerMediaItems map[string]config.MediaItem) (map[string]config.MediaItem, error) {
	newMediaItems := make(map[string]config.MediaItem, 0)

	// iterate new media items
	for mediaId, mediaItem := range providerMediaItems {
		// do we already have this media item?
		if _, ok := pvrMediaItems[mediaId]; ok {
			continue
		}

		newMediaItems[mediaId] = mediaItem
	}

	return newMediaItems, nil
}
