package cmd

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/spf13/cobra"
	"github.com/tleyden/deepstyle/deepstylelib"
	"github.com/tleyden/go-couch"
)

const (
	DesignDocName = "unprocessed_jobs"
	ViewName      = "unprocessed_jobs"
)

// publish_cloudwatch_metricsCmd respresents the publish_cloudwatch_metrics command
var publish_cloudwatch_metricsCmd = &cobra.Command{
	Use:   "publish_cloudwatch_metrics",
	Short: "Publish queue metrics to CloudWatch in order to trigger auto-scale alarms",
	Long:  `Publish queue metrics to CloudWatch in order to trigger auto-scale alarms`,
	Run: func(cmd *cobra.Command, args []string) {

		if err := cmd.ParseFlags(args); err != nil {
			log.Printf("err: %v", err)
			return
		}

		awsKey := cmd.Flag("aws_key")

		awsKeyVal := awsKey.Value.String()

		log.Printf("awsKey: %v", awsKeyVal)
		if awsKeyVal == "" {
			log.Printf("ERROR: Missing: --aws_key.\n  %v", cmd.UsageString())
			return
		}

		urlFlag := cmd.Flag("admin_url")

		urlVal := urlFlag.Value.String()
		if urlVal == "" {
			log.Printf("ERROR: Missing: --url.\n  %v", cmd.UsageString())
			return
		}

		err := addCloudWatchMetrics(urlVal)
		if err != nil {
			log.Printf("ERROR: %v", err)
			return
		}

	},
}

func numJobsReadOrBeingProcessed(syncGwAdminUrl string) (metricValue float64, err error) {

	// try to query view
	//    curl localhost:4985/deepstyle/_design/unprocessed_jobs/_view/unprocessed_jobs
	// if we get a 404, then install the view and then requery

	// if it has a trailing slash, remove it
	rawUrl := strings.TrimSuffix(syncGwAdminUrl, "/")

	// url validation
	url, err := url.Parse(rawUrl)
	if err != nil {
		return 0.0, err
	}

	db, err := couch.Connect(url.String())
	if err != nil {
		return 0.0, fmt.Errorf("Error connecting to db: %v.  Err: %v", syncGwAdminUrl, err)
	}
	log.Printf("connected to db: %v", db)

	viewUrl := fmt.Sprintf("_design/%v/_view/%v", DesignDocName, ViewName)
	options := map[string]interface{}{}
	output := map[string]interface{}{}
	err = db.Query(viewUrl, options, &output)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			// the view doesn't exist yet, attempt to install view
			if errInstallView := installView(rawUrl); errInstallView != nil {
				// failed to install view, give up
				return 0.0, errInstallView
			}
			// now retry
			errInner := db.Query(viewUrl, options, &output)
			if errInner != nil {
				// failed again, give up
				return 0.0, errInner
			}
		} else {
			return 0.0, err
		}
	}
	log.Printf("output: %+v", output)

	// TODO: count the number of rows / keys in the output and
	// convert to a float and return that value

	return 0.0, nil
}

type ViewParams struct {
	JobDocType string
	JobState1  string
	JobState2  string
	JobState3  string
}

func installView(syncGwAdminUrl string) error {

	viewJsonTemplate := `
{
    "views":{
        "unprocessed_jobs":{
            "map":"function (doc, meta) { if (doc._sync === undefined || meta.id.substring(0,6) == \"_sync:\") { return; } if (doc.type != {{.JobDocType}}) { return; } if (doc.state != {{.JobState1}}) { return; } if (doc.state != {{.JobState2}}) { return; } if (doc.state != {{.JobState3}}) { return; } emit(doc.state, meta.id); }"
        }
    }
}
`

	viewParams := ViewParams{
		JobDocType: deepstylelib.Job,
		JobState1:  deepstylelib.StateNotReadyToProcess,
		JobState2:  deepstylelib.StateReadyToProcess,
		JobState3:  deepstylelib.StateBeingProcessed,
	}
	tmpl, err := template.New("UnprocessedJobsView").Parse(viewJsonTemplate)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer // A Buffer needs no initialization.

	err = tmpl.Execute(&buffer, viewParams)
	if err != nil {
		return err
	}

	log.Printf("installView called")

	// curl -X PUT -H "Content-type: application/json" localhost:4985/todolite/_design/all_lists --data @testview
	viewUrl := fmt.Sprintf("%v/_design/%v", syncGwAdminUrl, DesignDocName)

	bufferBytes := buffer.Bytes()
	log.Printf("view: %v", string(bufferBytes))

	req, err := http.NewRequest("PUT", viewUrl, bytes.NewReader(bufferBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	log.Printf("put view resp: %v", resp)

	return nil

}

func addCloudWatchMetrics(syncGwAdminUrl string) error {

	metricValue, err := numJobsReadOrBeingProcessed(syncGwAdminUrl)
	if err != nil {
		return err
	}

	// TODO: push something to CloudWatch
	// CLI example:
	//   aws cloudwatch put-metric-data
	//     --metric-name PageViewCount
	//      --namespace "MyService"
	//     --value 2
	//     --timestamp 2014-02-14T12:00:00.000Z
	cloudwatchSvc := cloudwatch.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})

	log.Printf("cloudwatchSvc: %v", cloudwatchSvc)

	metricName := "NumJobsReadyOrBeingProcessed"
	timestamp := time.Now()

	metricDatum := &cloudwatch.MetricDatum{
		MetricName: &metricName,
		Value:      &metricValue,
		Timestamp:  &timestamp,
	}

	metricDatumSlice := []*cloudwatch.MetricDatum{metricDatum}
	namespace := "DeepStyleQueue"

	putMetricDataInput := &cloudwatch.PutMetricDataInput{
		MetricData: metricDatumSlice,
		Namespace:  &namespace,
	}

	out, err := cloudwatchSvc.PutMetricData(putMetricDataInput)
	if err != nil {
		log.Printf("ERROR adding metric data  %v", err)
		return err
	}
	log.Printf("Metric data output: %v", out)

	return nil

}

func init() {
	RootCmd.AddCommand(publish_cloudwatch_metricsCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands

	publish_cloudwatch_metricsCmd.PersistentFlags().String("aws_key", "", "AWS Key")

	publish_cloudwatch_metricsCmd.PersistentFlags().String("admin_url", "", "Sync Gateway Admin URL")

	// Cobra supports local flags which will only run when this command is called directly
	// publish_cloudwatch_metricsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
