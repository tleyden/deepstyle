package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
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

		svc := ec2.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})

		// Call the DescribeInstances Operation
		resp, err := svc.DescribeInstances(nil)
		if err != nil {
			panic(err)
		}

		// resp has all of the response data, pull out instance IDs:
		fmt.Println("> Number of reservation sets: ", len(resp.Reservations))
		for idx, res := range resp.Reservations {
			fmt.Println("  > Number of instances: ", len(res.Instances))
			for _, inst := range resp.Reservations[idx].Instances {
				fmt.Println("    - Instance ID: ", *inst.InstanceId)
				fmt.Println("    - Key Name: ", *inst.KeyName)
			}
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
		metricValue := 0.0
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
			return
		}
		log.Printf("Metric data output: %v", out)

	},
}

func init() {
	RootCmd.AddCommand(publish_cloudwatch_metricsCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands

	publish_cloudwatch_metricsCmd.PersistentFlags().String("aws_key", "", "AWS Key")

	// Cobra supports local flags which will only run when this command is called directly
	// publish_cloudwatch_metricsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
