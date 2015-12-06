
## Overview

DeepStyle in the cloud

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