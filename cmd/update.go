package cmd

import (
	"fmt"
	"github.com/equinox-io/equinox"
	"github.com/spf13/cobra"
	"os"
)

var (
	equinoxAppId  = "app_d3GUrCFKy57"
	equinoxKeyPub = []byte(`
-----BEGIN ECDSA PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEGNDmt9KcyTqnonXryPLomr0CsXZcGnGN
2wZJLdIkq4PDUBZOOv6uJpINUsUY9wtipXqB39KoBIMRP3SO0lN86NxPe8/r908K
a6FKnkhfZDaoZ2/akxMEMTMiWj2P674M
-----END ECDSA PUBLIC KEY-----
`)
)

var updateCmd = &cobra.Command{
	Use:   "update [CHANNEL]",
	Short: "Update to latest release",
	Long:  `This command can be used to update to the latest release.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate channel
		channelName := args[0]

		switch channelName {
		case "stable", "beta":
			break
		default:
			fmt.Println("You must provide a valid channel, e.g. stable or beta")
			os.Exit(1)
		}

		// init core
		initCore()
		showUsing()

		// init updater
		opts := equinox.Options{
			Channel: channelName,
		}
		if err := opts.SetPublicKeyPEM(equinoxKeyPub); err != nil {
			log.WithError(err).Fatal("Failed initializing updater...")
		}

		// check for the update
		resp, err := equinox.Check(equinoxAppId, opts)
		switch {
		case err == equinox.NotAvailableErr:
			log.Infof("No %q update available, already using the latest version!", channelName)
			os.Exit(0)
		case err != nil:
			log.WithError(err).Fatalf("Failed checking for latest %q version", channelName)
		}

		// fetch the update and apply it
		err = resp.Apply()
		if err != nil {
			log.WithError(err).Fatalf("Failed updating to latest %q version: %s", channelName, resp.ReleaseVersion)
		}

		log.Infof("Updated to latest %q version: %s", channelName, resp.ReleaseVersion)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
