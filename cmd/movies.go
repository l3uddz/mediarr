package cmd

import (
	"github.com/l3uddz/mediarr/database"
	providerObj "github.com/l3uddz/mediarr/provider"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/spf13/cobra"
	"strings"
)

var moviesCmd = &cobra.Command{
	Use:   "movies [PVR] [PROVIDER]",
	Short: "Search for new movies",
	Long:  `This command can be used to search for new movies.`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// validate core inputs
		if err := parseValidateInputs(args); err != nil {
			log.WithError(err).Fatal("Failed validating inputs")
		}

		// init database
		if err := database.Init(flagDatabaseFile); err != nil {
			log.WithError(err).Fatal("Failed opening database file")
		}
		defer database.Close()

		// init provider object
		if err := provider.Init(providerObj.Movie, providerCfg); err != nil {
			log.WithError(err).Fatalf("Failed initializing provider object for: %s", providerName)
		}

		provider.SetAcceptMediaItemFn(shouldAcceptMediaItem)

		// validate provider supports search type
		if supported := provider.SupportsMoviesSearchType(flagSearchType); !supported {
			log.WithField("search_type", flagSearchType).Fatalf("Unsupported search type, valid types: %s",
				strings.Join(provider.GetMoviesSearchTypes(), ", "))
		}

		// init pvr object
		if err := pvr.Init(pvrObj.MOVIE); err != nil {
			log.WithError(err).Fatalf("Failed initializing pvr object for: %s", pvrName)
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
			"genre":    flagGenre,
			"year":     flagYear,
		}

		// retrieve media
		foundMediaItems, err := provider.GetMovies(flagSearchType, logic, params)
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
	rootCmd.AddCommand(moviesCmd)

	// required flags
	moviesCmd.Flags().StringVarP(&flagSearchType, "search-type", "t", "", "Search type.")
	_ = moviesCmd.MarkFlagRequired("search-type")

	// optional flags
	moviesCmd.Flags().BoolVarP(&flagRefreshCache, "refresh-cache", "r", false, "Refresh the locally stored cache.")

	moviesCmd.Flags().IntVar(&flagLimit, "limit", 0, "Max accepted items to add.")

	moviesCmd.Flags().StringVar(&flagCountry, "country", "", "Countries to filter results.")
	moviesCmd.Flags().StringVar(&flagLanguage, "language", "", "Languages to filter results.")
	moviesCmd.Flags().StringVar(&flagGenre, "genre", "", "Genres to filter results.")
	moviesCmd.Flags().StringVar(&flagYear, "year", "", "Years to filter results.")

}
