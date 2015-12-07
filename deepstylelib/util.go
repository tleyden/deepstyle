package deepstylelib

import (
	"io"
	"os"
	"os/exec"
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

func hasGPU() bool {

	cmd := exec.Command("nvidia-smi")

	if err := cmd.Start(); err != nil {
		return false
	}

	if err := cmd.Wait(); err != nil {
		return false
	}

	return true
}

func torchInstalled() bool {

	cmd := exec.Command("th", "--help")

	if err := cmd.Start(); err != nil {
		return false
	}

	if err := cmd.Wait(); err != nil {
		return false
	}

	return true

}

func cp(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
