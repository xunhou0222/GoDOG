/*
Reference:
    Hatanaka, Y. (2008), A Compression Format and Tools for GNSS Observation
        Data, Bulletin of the Geospatioal Information Authority of Japan, 55, 21-30.
    (available at https://www.gsi.go.jp/ENGLISH/Bulletin55.html)
*/

package crx2rnx

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func CRX2RNX(inFile string, outFile *string) (err error) {
	// 1. Check the input (crx) file
	var inFileLen int = len(inFile)

	if inFileLen == 0 {
		err = fmt.Errorf("the crx file name is empty")
		return
	}

	// 2. Check the output (rnx) file
	// If the output file is empty, then the default name will be used
	if outFileLen := len(*outFile); outFileLen == 0 {
		var idx int = strings.LastIndexByte(inFile, '.')

		if idx == -1 || inFileLen-1-idx != 3 ||
			inFile[idx+3] != 'd' && inFile[idx+3] != 'D' &&
				inFile[idx+1:] != "CRX" && inFile[idx+1:] != "crx" {
			err = fmt.Errorf("invalid file name \"%s\", the extension of the input file "+
				"name should be [.??d], [.??D], [.crx] or [.CRX]", inFile)
			return
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

	// 3. Open the input and output files
	inFilePt, err := os.Open(inFile)

	if err != nil {
		return
	}

	defer inFilePt.Close()

	scanner := bufio.NewScanner(inFilePt)

	outFilePt, err := os.Create(*outFile)

	if err != nil {
		return
	}

	defer outFilePt.Close()

	writer := bufio.NewWriter(outFilePt)
	defer writer.Flush()

	// 4. Read and write the header
	var nl int64 = 0
	var crxVer, rnxVer, TypeNum int
	var TypeNumGNSS [256]int

	err = header(scanner, writer, &crxVer, &rnxVer, &TypeNum, &TypeNumGNSS, &nl)

	if err != nil {
		err = fmt.Errorf("failed to generate the header, %s", err)
		return
	}

	// 5. Read and write the body
	err = body(scanner, writer, crxVer, rnxVer, &TypeNum, &TypeNumGNSS, &nl)

	if err != nil {
		err = fmt.Errorf("failed to generate the body, %s", err)
		return
	}

	return nil
}
