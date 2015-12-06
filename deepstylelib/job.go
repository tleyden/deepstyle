package deepstylelib

import (
	"log"
	"path"

	"github.com/tleyden/go-couch"
)

const (
	SourceImageAttachment = "source_image"
	StyleImageAttachment  = "style_image"
	ResultImageAttachment = "result_image"
)

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

func (d DeepStyleJob) Execute() (err error, outputFilePath, tmpDirPath string) {

	if d.config.UnitTestMode == true {
		return nil, "/tmp/foo", "/tmp"
	}

	if err := d.DownloadAttachments(); err != nil {
		return err, "", ""
	}

	outputFilePath = path.Join(
		d.config.TempDir,
		d.jobDoc.Id,
		"_result.png",
	)

	// Invoke neural style via Exec() and pass in result filename
	// cd ~/neural-style
	// th neural_style.lua -backend cudnn -image_size 900 -style_image images/andrea_field.jpeg -content_image images/backyard.jpg -output_image images/styled-andrea-field-backyard_900.jpg

	return nil, outputFilePath, d.config.TempDir

}

func (d DeepStyleJob) DownloadAttachments() error {

	attachmentNames := []string{SourceImageAttachment, StyleImageAttachment}
	for _, attachmentName := range attachmentNames {

		attachmentReader, err := d.jobDoc.RetrieveAttachment(attachmentName)
		if err != nil {
			return err
		}
		attachmentFilename := path.Join(
			d.config.TempDir,
			d.jobDoc.Id,
			"_",
			attachmentName,
			".png", // TODO: look at attachment content type and get correct extentions
		)
		err = writeToFile(attachmentReader, attachmentFilename)
		if err != nil {
			return err
		}

	}
	return nil

}

func executeDeepStyleJob(config configuration, jobDoc JobDocument) error {

	jobDoc.SetConfiguration(config)
	deepStyleJob := NewDeepStyleJob(jobDoc, config)
	err, outputFilePath, tmpDirPath := deepStyleJob.Execute()

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
	log.Printf("Delete %v", tmpDirPath)

	return nil
}
