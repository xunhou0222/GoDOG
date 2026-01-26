/*
A package used to decompress files in .gz or .Z format.

Reference:
 1. GNU Gzip 1.13
*/
package unzip

import (
	"compress/gzip"
	"godog/unzip/lzw"
	"io"
	"os"
)

func UnzipGZ(srcFile, desFile string) error {
	srcFilePt, err := os.Open(srcFile)

	if err != nil {
		return err
	}

	defer srcFilePt.Close()

	srcReader, err := gzip.NewReader(srcFilePt)

	if err != nil {
		return err
	}

	defer srcReader.Close()

	desFilePt, err := os.Create(desFile)

	if err != nil {
		return err
	}

	defer desFilePt.Close()

	_, err = io.Copy(desFilePt, srcReader)

	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func UnzipZ(srcFile, desFile string) error {
	srcFilePt, err := os.Open(srcFile)

	if err != nil {
		return err
	}

	defer srcFilePt.Close()

	srcReader, err := lzw.NewReader(srcFilePt)

	if err != nil {
		return err
	}

	desFilePt, err := os.Create(desFile)

	if err != nil {
		return err
	}

	defer desFilePt.Close()

	_, err = io.Copy(desFilePt, srcReader)

	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
