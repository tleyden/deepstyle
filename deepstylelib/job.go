package deepstylelib

import (
	"fmt"
	"log"
	"os/exec"
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

func (d DeepStyleJob) Execute() (err error, outputFilePath, stdOutAndErr string) {

	if d.config.UnitTestMode == true {
		return nil, "/tmp/foo", "/tmp"
	}

	err, sourceImagePath, styleImagePath := d.DownloadAttachments()

	if err != nil {
		return err, "", ""
	}

	outputFilename := fmt.Sprintf(
		"%v_%v.png",
		d.jobDoc.Id,
		ResultImageAttachment,
	)
	outputFilePath = path.Join(
		d.config.TempDir,
		outputFilename,
	)

	stdOutAndErrByteSlice, err := d.executeNeuralStyle(
		sourceImagePath,
		styleImagePath,
		outputFilePath,
	)

	return err, outputFilePath, string(stdOutAndErrByteSlice)

}

func (d DeepStyleJob) executeNeuralStyle(sourceImagePath, styleImagePath, outputFilePath string) (stdOutAndErr []byte, err error) {

	torchInstalled := torchInstalled()

	if torchInstalled {
		useGpu := hasGPU()
		cmd := d.generateNeuralStyleCommand(
			sourceImagePath,
			styleImagePath,
			outputFilePath,
			useGpu,
		)
		// set the current working directory to ~/neural_style
		cmd.Dir = "/home/ubuntu/neural-style"

		// Execute the command and get the output
		log.Printf("Invoking neural-style")
		return cmd.CombinedOutput()

	} else {
		useGpu := hasGPU()
		log.Printf("useGpu: %v", useGpu)
		// copy the sourceImagePath to the outputFilePath
		cp(outputFilePath, sourceImagePath)
		return []byte("Torch not installed, just created a fake output file"), nil
	}

}

func (d DeepStyleJob) generateNeuralStyleCommand(sourceImagePath, styleImagePath, outputFilePath string, useGpu bool) (cmd *exec.Cmd) {

	gpuId := "-1"
	if useGpu {
		gpuId = "0"
	}

	return exec.Command(
		"th",
		"neural_style.lua",
		"-gpu",
		gpuId,
		"-style_image",
		styleImagePath,
		"-content_image",
		sourceImagePath,
		"-output_image",
		outputFilePath,
	)

}

func (d DeepStyleJob) DownloadAttachments() (err error, sourceImagePath, styleImagePath string) {

	attachmentNames := []string{SourceImageAttachment, StyleImageAttachment}
	attachmentPaths := []string{}

	for _, attachmentName := range attachmentNames {

		attachmentReader, err := d.jobDoc.RetrieveAttachment(attachmentName)
		if err != nil {
			return fmt.Errorf("Error retrieving attachment: %v", err), "", ""
		}

		filename := fmt.Sprintf(
			"%v_%v.png",
			d.jobDoc.Id,
			attachmentName,
		)
		attachmentFilepath := path.Join(
			d.config.TempDir,
			filename,
		)
		attachmentPaths = append(attachmentPaths, attachmentFilepath)

		err = writeToFile(attachmentReader, attachmentFilepath)
		if err != nil {
			return fmt.Errorf("Error writing file: %v", err), "", ""
		}

	}
	return err, attachmentPaths[0], attachmentPaths[1]

}

func executeDeepStyleJob(config configuration, jobDoc JobDocument) error {

	jobDoc.SetConfiguration(config)
	deepStyleJob := NewDeepStyleJob(jobDoc, config)
	err, outputFilePath, stdOutAndErr := deepStyleJob.Execute()

	// Did the job fail?
	if err != nil {
		// Record failure
		log.Printf("Job failed with error: %v", err)
		jobDoc.UpdateState(StateProcessingFailed)
		jobDoc.SetErrorMessage(err)
		jobDoc.SetStdOutAndErr(stdOutAndErr)
		return err
	}

	// Try to attach the result image, otherwise consider it a failure
	if err := jobDoc.AddAttachment(ResultImageAttachment, outputFilePath); err != nil {
		jobDoc.UpdateState(StateProcessingFailed)
		log.Printf("Set err message to: %v", err)
		updated, errSet := jobDoc.SetErrorMessage(err)
		log.Printf("setErrorMessage updated: %v errSet: %v", updated, errSet)
		updated, errSet = jobDoc.SetStdOutAndErr(stdOutAndErr)
		log.Printf("SetStdOutAndErr updated: %v errSet: %v", updated, errSet)
		return err
	}

	// Record successful result in job
	jobDoc.SetStdOutAndErr(stdOutAndErr)
	jobDoc.UpdateState(StateProcessingSuccessful)

	// TODO: Delete all temp files

	return nil
}
