package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tleyden/deepstyle/deepstylelib"
)

var (
	processJobs       *bool
	sendNotifications *bool
	since             *string
)

var follow_sync_gwCmd = &cobra.Command{

	Use:   "follow_sync_gw",
	Short: "Follow the sync gateway changes feed and process jobs",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if err := cmd.ParseFlags(args); err != nil {
			log.Printf("err: %v", err)
			return
		}

		// Sync Gateway URL
		urlFlag := cmd.Flag("url")
		urlVal := urlFlag.Value.String()
		log.Printf("url val: %v", urlVal)
		if urlVal == "" {
			log.Printf("ERROR: Missing: --url.\n  %v", cmd.UsageString())

			return
		}

		// Uniqush URL
		uniqushUrlFlag := cmd.Flag("uniqush-url")
		uniqushUrlVal := uniqushUrlFlag.Value.String()
		log.Printf("uniqush url val: %v", uniqushUrlVal)

		shouldProcessJobs := *processJobs
		shouldSendNotifications := *sendNotifications

		if !shouldProcessJobs && !shouldSendNotifications {
			log.Panicf("You need to either set the --process-jobs or --send-notifications flag, otherwise there is nothing to do!")
		}

		if shouldSendNotifications && uniqushUrlVal == "" {
			log.Panicf("You must pass a --uniqush-url to send notifications")
		}

		// Create Changes follower
		changesFollower, err := deepstylelib.NewChangesFeedFollower(*since, urlVal)
		if err != nil {
			log.Panicf("%v", err)
		}

		changesFollower.ProcessJobs = shouldProcessJobs
		changesFollower.SendNotifications = shouldSendNotifications

		// Set uniqush url if one was passed in
		if uniqushUrlVal != "" {
			changesFollower.UniqushURL = uniqushUrlVal
		}

		// Start following changes
		changesFollower.Follow()

	},
}

func init() {

	RootCmd.AddCommand(follow_sync_gwCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands
	follow_sync_gwCmd.PersistentFlags().String("url", "", "Sync Gateway URL")
	follow_sync_gwCmd.MarkPersistentFlagRequired("url")

	follow_sync_gwCmd.PersistentFlags().String("uniqush-url", "", "Uniqush URL (push notifications)")

	processJobs = follow_sync_gwCmd.PersistentFlags().BoolP("process-jobs", "p", false, "Process DeepStyle jobs (requires deps + GPU)")

	sendNotifications = follow_sync_gwCmd.Flags().BoolP("send-notifications", "s", false, "Send push notifications (requires Uniqush url)")

	since = follow_sync_gwCmd.PersistentFlags().String("since", "", "Since value to start changes feed at (defaults to last sequence)")

	// Cobra supports local flags which will only run when this command is called directly
	// follow_sync_gwCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
