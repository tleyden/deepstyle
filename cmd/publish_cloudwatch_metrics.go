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

		err := addCloudWatchMetrics(urlVal)
		if err != nil {
			log.Printf("ERROR: %v", err)
			return
		}

	},
}

func numJobsReadyOrBeingProcessed(syncGwAdminUrl string) (metricValue float64, err error) {

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

	viewUrl := fmt.Sprintf("_design/%v/_view/%v", DesignDocName, ViewName)
	options := map[string]interface{}{}
	options["stale"] = "false"
	output := map[string]interface{}{}
	err = db.Query(viewUrl, options, &output)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			// the view doesn't exist yet, attempt to install view
			if errInstallView := installView(rawUrl); errInstallView != nil {
				// failed to install view, give up
				return 0.0, errInstallView

			}

			// without this workaround, I'm getting:
			// ERROR: HTTP Error 500 Internal Server Error - {"error":"Internal Server Error","reason":"Internal error: error executing view req at http://127.0.0.1:8092/deepstyle/_design/unprocessed_jobs/_view/unprocessed_jobs?stale=false: 500 Internal Server Error - {\"error\":\"unknown_error\",\"reason\":\"view_undefined\"}\n"}

			log.Printf("Sleeping 10s to wait for view to be ready")
			<-time.After(time.Duration(10) * time.Second)
			log.Printf("Done sleeping 10s to wait for view to be ready")

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
	outputRows := output["total_rows"].(float64)
	return float64(outputRows), nil

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
            "map":"function (doc, meta) { if (doc.type != '{{.JobDocType}}') { return; } if (doc.state == '{{.JobState1}}' || doc.state == '{{.JobState2}}' || doc.state == '{{.JobState3}}') { emit(doc.state, meta.id); }}"
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
	for {

		log.Printf("Adding metrics for queue")
		addCloudWatchMetric(syncGwAdminUrl)

		numSecondsToSleep := 60
		log.Printf("Sleeping %v seconds", numSecondsToSleep)
		<-time.After(time.Duration(numSecondsToSleep) * time.Second)

	}
}

func addCloudWatchMetric(syncGwAdminUrl string) error {

	metricValue, err := numJobsReadyOrBeingProcessed(syncGwAdminUrl)
	log.Printf("Adding metric: numJobsReadyOrBeingProcessed = %v", metricValue)
	if err != nil {
		return err
	}

	cloudwatchSvc := cloudwatch.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})

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

	_, err = cloudwatchSvc.PutMetricData(putMetricDataInput)
	if err != nil {
		log.Printf("ERROR adding metric data  %v", err)
		return err
	}

	return nil

}

func init() {
	RootCmd.AddCommand(publish_cloudwatch_metricsCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands

	publish_cloudwatch_metricsCmd.PersistentFlags().String("admin_url", "", "Sync Gateway Admin URL")

	// Cobra supports local flags which will only run when this command is called directly
	// publish_cloudwatch_metricsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
