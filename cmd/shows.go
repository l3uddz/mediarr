package cmd

import (
	"github.com/l3uddz/mediarr/database"
	providerObj "github.com/l3uddz/mediarr/provider"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/l3uddz/mediarr/utils/media"
	"github.com/spf13/cobra"
)

var showsCmd = &cobra.Command{
	Use:   "shows [PVR] [PROVIDER]",
	Short: "Search for new shows",
	Long:  `This command can be used to search for new shows.`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// validate inputs
		if err := parseValidateInputs(args); err != nil {
			log.WithError(err).Fatal("Failed validating inputs")
		}

		// init database
		if err := database.Init(flagDatabaseFile); err != nil {
			log.WithError(err).Fatal("Failed opening database file")
		}
		defer database.Close()

		// init provider object
		if err := provider.Init(providerObj.SHOW); err != nil {
			log.WithError(err).Fatalf("Failed initializing provider object for: %s", providerName)
		}

		// init pvr object
		if err := pvr.Init(pvrObj.SHOW); err != nil {
			log.WithError(err).Fatalf("Failed initializing pvr object for: %s", pvrName)
		}

		// get existing media
		existingMediaItems, err := pvr.GetExistingMedia()
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving existing media from pvr")
		}

		// retrieve media
		foundMediaItems, err := provider.GetShows()
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving media from provider")
		}

		// remove existing media items
		newMediaItems, err := media.PruneExistingMedia(existingMediaItems, foundMediaItems)
		if err != nil {
			log.WithError(err).Fatal("Failed removing existing media from provider media items")
		}

		newMediaItemsSize := len(newMediaItems)
		log.WithField("new_media_items", newMediaItemsSize).Info("Pruned existing media items from provider items")

		// iterate items evaluating against filters
		for _, mediaItem := range newMediaItems {
			// ignore this item?
			ignore, err := pvr.ShouldIgnore(&mediaItem)
			if err != nil {
				log.WithError(err).Error("Failed evaluating ignore expressions against: %+v", mediaItem)
				continue
			}

			if ignore {
				log.Debugf("Ignoring: %+v", mediaItem)
				continue
			}

			log.Infof("Accepted: %+v", mediaItem)
		}
	},
}

func init() {
	rootCmd.AddCommand(showsCmd)

	showsCmd.Flags().BoolVarP(&flagRefreshCache, "refresh-cache", "r", false, "Refresh the locally stored cache.")
}
