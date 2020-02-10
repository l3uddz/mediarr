package cmd

import (
	"github.com/l3uddz/mediarr/database"
	providerObj "github.com/l3uddz/mediarr/provider"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/spf13/cobra"
	"strings"
)

var showsCmd = &cobra.Command{
	Use:   "shows [PVR] [PROVIDER]",
	Short: "Search for new shows",
	Long:  `This command can be used to search for new shows.`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var err error

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
		if err := provider.Init(providerObj.Show, providerCfg); err != nil {
			log.WithError(err).Fatalf("Failed initializing provider object for: %s", providerName)
		}

		provider.SetAcceptMediaItemFn(shouldAcceptMediaItem)

		// init pvr object
		if err := pvr.Init(pvrObj.SHOW); err != nil {
			log.WithError(err).Fatalf("Failed initializing pvr object for: %s", pvrName)
		}

		// validate provider supports search type
		if supported := provider.SupportsShowsSearchType(flagSearchType); !supported {
			log.WithField("search_type", flagSearchType).Fatalf("Unsupported search type, valid types: %s",
				strings.Join(provider.GetShowsSearchTypes(), ", "))
		}

		// get existing media
		existingMediaItems, err = pvr.GetExistingMedia()
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving existing media from pvr")
		}

		// build logic map
		logic := map[string]interface{}{
			"limit": flagLimit,
		}

		// build param map
		params := map[string]string{
			"country":  flagCountry,
			"language": flagLanguage,
		}

		// retrieve media
		foundMediaItems, err := provider.GetShows(flagSearchType, logic, params)
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving media from provider")
		}

		// iterate accepted items
		for _, mediaItem := range foundMediaItems {
			log.Infof("Accepted: %+v", mediaItem)
		}
	},
}

func init() {
	rootCmd.AddCommand(showsCmd)

	// required flags
	showsCmd.Flags().StringVarP(&flagSearchType, "search-type", "t", "", "Search type.")
	_ = showsCmd.MarkFlagRequired("search-type")

	// optional flags
	showsCmd.Flags().BoolVarP(&flagRefreshCache, "refresh-cache", "r", false, "Refresh the locally stored cache.")

	showsCmd.Flags().IntVar(&flagLimit, "limit", 0, "Max accepted items to add.")

	showsCmd.Flags().StringVar(&flagCountry, "country", "", "Country to filter results.")
	showsCmd.Flags().StringVar(&flagLanguage, "language", "", "Language to filter results.")
}
