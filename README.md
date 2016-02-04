
## Overview

This is the backend "changes worker" for the [deepstyle-ios](https://github.com/tleyden/deepstyle-ios/) app.  

[Sync Gateway](https://waffle.io/couchbase/sync_gateway) provides the REST api, and [Couchbase Server](https://github.com/couchbase/manifest) provides the persistence layer.

* Sync Gateway changes listener to track the number of unprocessed jobs
* When new jobs are detected, publishes stats to CloudWatch
* CloudWatch Alarms cause AWS EC2 instances with GPU processors to spin up
    * Look for unprocessed jobs
    * Invoke [neural-style](https://github.com/jcjohnson/neural-style) to apply the artistic style to the photograph
    * Update job with results, which will sync down to the [deepstyle-ios](https://github.com/tleyden/deepstyle-ios/) app

## Steps to run

* Kick off ami `ami-5587c93f` (private AMI at the moment, stay tuned)
* Run `deepstyle follow_sync_gw --url http://demo.couchbasemobile.com:4984/deepstyle/`
* Use Paw/Curl to upload images

## JSON Docs

### Job

```
{
    "_attachments":{
        "photo":{
            "content_type":"image/png",
            "digest":"sha1-U8DAjp4S6T4HWWfo+HAdULGZpmw=",
            "length":3479844,
            "revpos":11,
            "stub":true
        },
        "style_image":{
            "content_type":"image/png",
            "digest":"sha1-IvVjV5TaT57zICYRpl/NJqJYVds=",
            "length":283092,
            "revpos":13,
            "stub":true
        }
    },
    "_id":"job",
    "_rev":"13-17ec2f655e6fc8e41809a00df7b15e53",
    "state":"READY_TO_PROCESS",
    "type":"job"
}
```

### Job States

* NOT_READY_TO_PROCESS (no attachments yet)
* READY_TO_PROCESS (attachments added)
* BEING_PROCESSED (worker running)
* PROCESSING_SUCCESSFUL (worker done, added result attachment)
* PROCESSING_FAILED (worker done, added error msg)

## Job Queue Processor

* For each change where type=job and state=READY_TO_PROCESS:
    * Change state to BEING_PROCESSED and update doc
    * Download attachments to temp files
    * Kick off exec and tell it to store result in a temp file
    * Wait for exec to finish
    * Add new attachment to doc with result
    * Change state to PROCESSING_SUCCESSFUL (or failed if exec failed)
    * Delete temp files

## Adding a new command (cobra)

```
cobra add publish_cloudwatch_metrics
```

## Creating cloudwatch alarms to trigger autoscale

```
$ aws autoscaling put-scaling-policy --policy-name deepstyle-scalout --auto-scaling-group-name DeepStyle --scaling-adjustment 1 --adjustment-type ChangeInCapacity --profile tleyden
$ aws autoscaling put-scaling-policy --policy-name deepstyle-scalein --auto-scaling-group-name DeepStyle --scaling-adjustment -1 --adjustment-type ChangeInCapacity --profile tleyden
$ aws cloudwatch put-metric-alarm --alarm-name AddCapacityToProcessDeepStyleQueue3 --metric-name NumJobsReadyOrBeingProcessed --namespace "DeepStyleQueue" --statistic Average --period 60 --threshold 1 --comparison-operator GreaterThanOrEqualToThreshold  --evaluation-periods 1 --alarm-actions $SCALE_OUT_ARN --profile tleyden
$ aws cloudwatch put-metric-alarm --alarm-name RemoveCapacityToProcessDeepStyleQueue3 --metric-name NumJobsReadyOrBeingProcessed --namespace "DeepStyleQueue" --statistic Average --period 60 --threshold 0 --comparison-operator LessThanOrEqualToThreshold  --evaluation-periods 1 --alarm-actions $SCALE_IN_ARN --profile tleyden

```
