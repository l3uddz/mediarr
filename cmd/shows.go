package cmd

import (
	"github.com/l3uddz/mediarr/database"
	pvrObj "github.com/l3uddz/mediarr/pvr"
	"github.com/spf13/cobra"
)

var showsCmd = &cobra.Command{
	Use:   "shows [PVR]",
	Short: "Search for new shows",
	Long:  `This command can be used to search for new shows.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate inputs
		if err := parseValidateInputs(args); err != nil {
			log.WithError(err).Fatal("Failed validating inputs")
		}

		// load database
		if err := database.Init(flagDatabaseFile); err != nil {
			log.WithError(err).Fatal("Failed opening database file")
		}
		defer database.Close()

		// init pvr object
		if err := pvr.Init(pvrObj.SHOW); err != nil {
			log.WithError(err).Fatalf("Failed initializing pvr object for: %s", pvrName)
		}

	},
}

func init() {
	rootCmd.AddCommand(showsCmd)

	showsCmd.Flags().BoolVarP(&flagRefreshCache, "refresh-cache", "r", false, "Refresh the locally stored cache.")
}
