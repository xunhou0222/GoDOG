/*
A package used to recover a RINEX file from a Compact RINEX (CRINEX) file.
It is compatable with CRX2RNX 4.2.0.

Reference:
 1. Hatanaka, Y. (2008), A Compression Format and Tools for GNSS Observation Data,
    Bulletin of the Geospatioal Information Authority of Japan, 55, 21-30.
    (available at https://www.gsi.go.jp/ENGLISH/Bulletin55.html)
*/
package crx2rnx

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

/***** FUNCTION ********************************/

func CRX2RNX(inFile string, outFile *string) error {
	// 1. check the input (crx) file and the output (rnx) file
	if len(inFile) == 0 {
		return errors.New("the input file name is empty")
	}

	// if the output file is empty, then the default name will be used
	if len(*outFile) == 0 {
		idx := strings.LastIndexByte(inFile, '.')

		if idx < 0 || len(inFile)-1-idx != 3 ||
			inFile[idx+3] != 'd' && inFile[idx+3] != 'D' && inFile[idx+1:] != "CRX" && inFile[idx+1:] != "crx" {
			return errors.New("invalid extension of the input file name, which should be [.??d], [.??D], [.crx] or [.CRX]")
		} else {
			if inFile[idx+3] == 'd' {
				*outFile = inFile[:idx+3] + "o"
			} else if inFile[idx+3] == 'D' {
				*outFile = inFile[:idx+3] + "O"
			} else if inFile[idx+1:] == "crx" {
				*outFile = inFile[:idx+1] + "rnx"
			} else if inFile[idx+1:] == "CRX" {
				*outFile = inFile[:idx+1] + "RNX"
			}
		}
	}

	// 2. open the input file and the output file
	fi, err := os.Open(inFile)

	if err != nil {
		return err
	}

	defer fi.Close()

	fo, err := os.Create(*outFile)

	if err != nil {
		return err
	}

	defer fo.Close()

	scanner := bufio.NewScanner(fi)
	writer := bufio.NewWriter(fo)
	defer writer.Flush()

	// 3. read and write the header
	var (
		nl             int64
		crxVer, rnxVer int
		TypeNumGNSS    map[byte]int = make(map[byte]int)
	)

	err = header(scanner, writer, &crxVer, &rnxVer, TypeNumGNSS, &nl)

	if err != nil {
		return fmt.Errorf("failed to generate the header. %s", err)
	}

	// 4. read and write the body
	err = body(scanner, writer, crxVer, rnxVer, TypeNumGNSS, &nl)

	if err != nil {
		return fmt.Errorf("failed to generate the body, %s", err)
	}

	return nil
}

/***********************************************/
