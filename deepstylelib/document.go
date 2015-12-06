package deepstylelib

import "io"

// Doc types
const (
	Job = "job"
)

// Job States
const (
	StateNotReadyToProcess    = "NOT_READY_TO_PROCESS"  // no attachments yet
	StateReadyToProcess       = "READY_TO_PROCESS"      // attachments added
	StateBeingProcessed       = "BEING_PROCESSED"       // worker running
	StateProcessingSuccessful = "PROCESSING_SUCCESSFUL" // worker done
	StateProcessingFailed     = "PROCESSING_FAILED"     // processing failed
)

type Attachments map[string]interface{}

type Document struct {
	Revision string `json:"_rev"`
	Id       string `json:"_id"`
}

type TypedDocument struct {
	Document
	Type string `json:"type"`
}

func (doc TypedDocument) IsJob() bool {
	return doc.Type == Job
}

type JobDocument struct {
	TypedDocument
	Attachments  Attachments `json:"_attachments"`
	State        string      `json:"state"`
	ErrorMessage string      `json:"error_message"`
	config       configuration
}

func (doc JobDocument) IsReadyToProcess() bool {
	return doc.State == StateReadyToProcess
}

func (doc *JobDocument) RefreshFromDB() error {
	db := doc.config.Database
	jobDoc := JobDocument{}
	err := db.Retrieve(doc.Id, &jobDoc)
	if err != nil {
		return err
	}
	*doc = jobDoc
	return nil
}

func (doc *JobDocument) UpdateState(newState string) (updated bool, err error) {

	db := doc.config.Database

	retryUpdater := func() {
		doc.State = newState
	}

	retryDoneMetric := func() bool {
		return doc.State == newState
	}

	retryRefresh := func() error {
		return doc.RefreshFromDB()
	}

	return db.EditRetry(
		doc,
		retryUpdater,
		retryDoneMetric,
		retryRefresh,
	)

}

func (doc *JobDocument) SetErrorMessage(errorMessage error) (updated bool, err error) {

	db := doc.config.Database

	retryUpdater := func() {
		doc.ErrorMessage = errorMessage.Error()
	}

	retryDoneMetric := func() bool {
		return doc.ErrorMessage == errorMessage.Error()
	}

	retryRefresh := func() error {
		return doc.RefreshFromDB()
	}

	return db.EditRetry(
		doc,
		retryUpdater,
		retryDoneMetric,
		retryRefresh,
	)

}

func (doc *JobDocument) AddAttachment(attachmentName, outputFilePath string) {

}

func (doc *JobDocument) RetrieveAttachment(attachmentName string) (io.Reader, error) {
	db := doc.config.Database
	return db.RetrieveAttachment(doc.Id, attachmentName)
}

func (doc *JobDocument) SetConfiguration(config configuration) {
	doc.config = config
}
