package deepstylelib

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const (
	DesignDocName = "unprocessed_jobs"
	ViewName      = "unprocessed_jobs"
)

func numJobsReadyOrBeingProcessed(syncGwAdminUrl string) (metricValue float64, err error) {

	viewResults, err := getJobsReadyOrBeingProcessed(syncGwAdminUrl)
	if err != nil {
		return 0.0, err
	}
	numRows := viewResults["total_rows"].(float64)
	return float64(numRows), nil

}

func getJobDocsBeingProcessed(syncGwAdminUrl string) (jobs []JobDocument, err error) {

	jobs = []JobDocument{}

	db, err := GetDbConnection(syncGwAdminUrl)
	if err != nil {
		return jobs, fmt.Errorf("Error connecting to db: %v.  Err: %v", syncGwAdminUrl, err)
	}

	config := configuration{
		Database: db,
	}

	viewResults, err := getJobsReadyOrBeingProcessed(syncGwAdminUrl)
	if err != nil {
		return jobs, err
	}
	rows := viewResults["rows"].([]interface{})
	log.Printf("rows: %+v", rows)
	for _, row := range rows {
		log.Printf("row: %+v type: %T", row, row)
		rowMap := row.(map[string]interface{})
		docId := rowMap["id"].(string)
		log.Printf("id: %v", docId)

		// TODO: get configuration with database
		jobDoc, err := NewJobDocument(docId, config)
		if err != nil {
			log.Printf("Error %v retrieving job doc: %v, skipping", err, docId)
			continue
		}
		jobs = append(jobs, *jobDoc)

		log.Printf("job doc: %+v", jobDoc)

	}

	log.Printf("returning %v jobs", len(jobs))
	return jobs, nil

}

func getJobsReadyOrBeingProcessed(syncGwAdminUrl string) (viewResults map[string]interface{}, err error) {

	// try to query view
	//    curl localhost:4985/deepstyle/_design/unprocessed_jobs/_view/unprocessed_jobs
	// if we get a 404, then install the view and then requery

	output := map[string]interface{}{}

	db, err := GetDbConnection(syncGwAdminUrl)
	if err != nil {
		return output, fmt.Errorf("Error connecting to db: %v.  Err: %v", syncGwAdminUrl, err)
	}

	viewUrl := fmt.Sprintf("_design/%v/_view/%v", DesignDocName, ViewName)
	options := map[string]interface{}{}
	options["stale"] = "false"

	err = db.Query(viewUrl, options, &output)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			// the view doesn't exist yet, attempt to install view
			if errInstallView := installView(syncGwAdminUrl); errInstallView != nil {
				// failed to install view, give up
				return output, errInstallView

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
				return output, errInner
			}
		} else {
			return output, err
		}
	}
	return output, nil

}

type ViewParams struct {
	JobDocType string
	JobState1  string
	JobState2  string
	JobState3  string
}

func installView(syncGwAdminUrl string) error {

	// if url has a trailing slash, remove it
	syncGwAdminUrl = strings.TrimSuffix(syncGwAdminUrl, "/")

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
		JobDocType: Job,
		JobState1:  StateNotReadyToProcess,
		JobState2:  StateReadyToProcess,
		JobState3:  StateBeingProcessed,
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

func AddCloudWatchMetrics(syncGwAdminUrl string) error {
	for {

		jobs, err := getJobDocsBeingProcessed(syncGwAdminUrl)
		if err != nil {
			log.Printf("err: %v", err)
		}
		log.Printf("jobs: %v", jobs)

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
