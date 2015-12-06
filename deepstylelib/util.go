package deepstylelib

import (
	"io"
	"os"
)

func writeToFile(reader io.Reader, destFilename string) error {

	destFile, err := os.Create(destFilename)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, reader)
	return err

}
