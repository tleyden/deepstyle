package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tleyden/deepstyle/deepstylelib"
)

var follow_sync_gwCmd = &cobra.Command{

	Use:   "follow_sync_gw",
	Short: "Follow the sync gateway changes feed and process jobs",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if err := cmd.ParseFlags(args); err != nil {
			log.Printf("err: %v", err)
		}

		urlFlag := cmd.Flag("url")

		urlVal := urlFlag.Value.String()
		log.Printf("url val: %v", urlVal)
		if urlVal == "" {
			log.Printf("ERROR: Missing: --url.\n  %v", cmd.UsageString())

			return
		}

		changesFollower, err := deepstylelib.NewChangesFeedFollower(urlVal)
		if err != nil {
			log.Panicf("%v", err)
		}
		changesFollower.Follow()

	},
}

func init() {

	RootCmd.AddCommand(follow_sync_gwCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands
	follow_sync_gwCmd.PersistentFlags().String("url", "", "Sync Gateway URL")
	follow_sync_gwCmd.MarkPersistentFlagRequired("url")

	// Cobra supports local flags which will only run when this command is called directly
	// follow_sync_gwCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
