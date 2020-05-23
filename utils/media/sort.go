package media

import (
	"sort"

	"github.com/l3uddz/mediarr/config"
)

type SortType string

const (
	SortTypeReleaseDate SortType = "release"
)

func SortedMediaItemSlice(mediaItems map[string]config.MediaItem, sortType SortType) []config.MediaItem {
	var sortedMediaItems []config.MediaItem

	// add items to sortedMediaItems
	for _, v := range mediaItems {
		sortedMediaItems = append(sortedMediaItems, v)
	}

	// sort items
	switch sortType {
	default:
		// sort by Release Date
		sort.Slice(sortedMediaItems, func(i, j int) bool {
			return sortedMediaItems[i].Date.After(sortedMediaItems[j].Date)
		})
	}

	return sortedMediaItems
}
