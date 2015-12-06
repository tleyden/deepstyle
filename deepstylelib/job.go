package deepstylelib

import "github.com/tleyden/go-couch"

type configuration struct {
	Database     couch.Database
	TempDir      string // Where to store attachments and output
	UnitTestMode bool   // Are we in "Unit Test Mode"?
}

type DeepStyleJob struct {
	config configuration
	jobDoc JobDocument
}

func NewDeepStyleJob(jobDoc JobDocument, config configuration) *DeepStyleJob {
	return &DeepStyleJob{
		config: config,
		jobDoc: jobDoc,
	}
}

func (d DeepStyleJob) Execute() (err error, outputFilePath string) {

	if d.config.UnitTestMode == true {
		return nil, "/tmp/foo"
	}

	// Save attachments to temp dir

	// Invoke neural style via Exec()

	// Return path to output file

	return nil, "todo"

}

func executeDeepStyleJob(config configuration, jobDoc JobDocument) error {

	jobDoc.SetConfiguration(config)
	deepStyleJob := NewDeepStyleJob(jobDoc, config)
	err, outputFilePath := deepStyleJob.Execute()

	// Did the job fail?
	if err != nil {
		// Record failure
		jobDoc.UpdateState(StateProcessingFailed)
		jobDoc.SetErrorMessage(err)
		return err
	}

	// Record successful result in job
	jobDoc.AddAttachment("result", outputFilePath)
	jobDoc.UpdateState(StateProcessingSuccessful)

	// Delete all temp files

	return nil
}
