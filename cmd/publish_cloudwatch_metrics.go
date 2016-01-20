package cmd

import (
	_ "expvar"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/tleyden/deepstyle/deepstylelib"
)

// publish_cloudwatch_metricsCmd respresents the publish_cloudwatch_metrics command
var publish_cloudwatch_metricsCmd = &cobra.Command{
	Use:   "publish_cloudwatch_metrics",
	Short: "Publish queue metrics to CloudWatch in order to trigger auto-scale alarms",
	Long:  `Publish queue metrics to CloudWatch in order to trigger auto-scale alarms.  AWS keys will be taken from environment variables or ~/.aws/.  See github.com/aws/aws-sdk-go`,
	Run: func(cmd *cobra.Command, args []string) {

		if err := cmd.ParseFlags(args); err != nil {
			log.Printf("err: %v", err)
			return
		}

		urlFlag := cmd.Flag("admin_url")

		urlVal := urlFlag.Value.String()
		if urlVal == "" {
			log.Printf("ERROR: Missing: --url.\n  %v", cmd.UsageString())
			return
		}

		err := deepstylelib.AddCloudWatchMetrics(urlVal)
		if err != nil {
			log.Printf("ERROR: %v", err)
			return
		}

		go func() {
			log.Fatal(http.ListenAndServe(":4980", nil))
		}()

	},
}

func init() {
	RootCmd.AddCommand(publish_cloudwatch_metricsCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands

	publish_cloudwatch_metricsCmd.PersistentFlags().String("admin_url", "", "Sync Gateway Admin URL")

	// Cobra supports local flags which will only run when this command is called directly
	// publish_cloudwatch_metricsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
