package deepstylelib

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/couchbaselabs/logg"
	"github.com/tleyden/go-couch"
	"github.com/tleyden/uqclient/libuqclient"
)

/*
* For each change where type=job and state=READY_TO_PROCESS:
    * Change state to BEING_PROCESSED and update doc
    * Download attachments to temp files
    * Kick off exec and tell it to store result in a temp file
    * Wait for exec to finish
    * Add new attachment to doc with result
    * Change state to PROCESSING_SUCCESSFUL (or failed if exec failed)
    * Delete temp files
*/

type ChangesFeedFollower struct {
	Database          couch.Database
	UniqushURL        string
	ProcessJobs       bool // Run NeuralStyle (typically only on AWS+GPU)
	SendNotifications bool // Send push notifications when jobs done
}

func NewChangesFeedFollower(syncGatewayUrl string) (*ChangesFeedFollower, error) {

	db, err := GetDbConnection(syncGatewayUrl)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to db: %v.  Err: %v", syncGatewayUrl, err)
	}

	return &ChangesFeedFollower{
		Database: db,
	}, nil
}

func (f ChangesFeedFollower) Follow() {

	var since interface{}

	handleChange := func(reader io.Reader) interface{} {
		changes, err := decodeChanges(reader)
		if err != nil {
			// it's very common for this to timeout while waiting for new changes.
			// since we want to follow the changes feed forever, just log an error
			// TODO: don't even log an error if its an io.Timeout, just noise
			log.Printf("%T error decoding changes: %v.", err, err)
			return since
		}

		f.processChanges(changes)

		since = changes.LastSequence

		return since

	}

	options := map[string]interface{}{}
	options["feed"] = "longpoll"

	f.Database.Changes(handleChange, options)

}

func (f ChangesFeedFollower) processChanges(changes couch.Changes) {

	for _, change := range changes.Results {
		if err := f.processChange(change); err != nil {
			errMsg := fmt.Errorf("Error %v processing change %v", err, change)
			logg.LogError(errMsg)
		}

	}

}

func (f ChangesFeedFollower) processChange(change couch.Change) error {

	docId := change.Id
	log.Printf("processChange: %v", docId)

	if change.Deleted {
		return nil
	}

	// ignore any doc ids that start with "_user"
	if strings.HasPrefix(docId, "_user") {
		return nil
	}

	doc := TypedDocument{}
	err := f.Database.Retrieve(docId, &doc)
	if err != nil {
		return err
	}

	// skip any docs that aren't jobs
	if !doc.IsJob() {
		return nil
	}
	log.Printf("doc: %+v. isJob: %v", doc, doc.IsJob())

	// re-retrieve from db, I wish I knew a better way.
	jobDoc := JobDocument{}
	err = f.Database.Retrieve(docId, &jobDoc)
	if err != nil {
		return err
	}
	log.Printf("jobdoc: %+v", jobDoc)

	// skip any jobs that aren't ready to process
	if !jobDoc.IsReadyToProcess() {
		return nil
	}

	if f.ProcessJobs {

		// Run the job (call neural style)
		config := configuration{
			Database: f.Database,
			TempDir:  "/tmp",
		}

		if err := executeDeepStyleJob(config, jobDoc); err != nil {
			return err
		}
	}

	if f.SendNotifications {
		if err := f.sendNotifications(jobDoc); err != nil {
			return err
		}
	}

	return nil

}

func (f ChangesFeedFollower) sendNotifications(jobDoc JobDocument) error {

	log.Printf("Sending notification for %v@%v", jobDoc.Id, jobDoc.Revision)

	message := ""
	switch jobDoc.State {
	case StateProcessingSuccessful:
		message = "Your DeepStyle work of art is ready!"
	case StateProcessingFailed:
		message = "Oops, something went wrong making your DeepStyle work of art!"
	default:
		// Job isn't finished, don't send any notification
		return nil
	}

	// create subscriber in uniqush
	uniqushClient := libuqclient.NewUniqushClient(f.UniqushURL)
	uniqushService := uniqushClient.NewService("deepstyle", libuqclient.APNS)
	subscriber := uniqushService.NewSubscriber(jobDoc.Owner, jobDoc.OwnerDeviceToken)
	_, err := subscriber.Create()
	if err != nil {
		return err
	}

	_, err = subscriber.Push(message)
	if err != nil {
		return err
	}

	log.Printf("Sent notification for %v@%v", jobDoc.Id, jobDoc.Revision)

	return nil

}

func decodeChanges(reader io.Reader) (couch.Changes, error) {

	changes := couch.Changes{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&changes)
	return changes, err

}
