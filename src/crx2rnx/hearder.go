package crx2rnx

import (
	"bufio"
	"fmt"
	"strings"
)

func header(scanner *bufio.Scanner, writer *bufio.Writer, crxVer, rnxVer, TypeNum *int,
	TypeNumGNSS *[256]int, nl *int64) error {
	var strline string
	var flag bool

	for flag = scanner.Scan(); flag; flag = scanner.Scan() {
		*nl++
		strline = strings.TrimRight(scanner.Text(), " ")

		if len(strline) <= 60 {
			return fmt.Errorf("after reading line %d, \"%s\", "+
				              "the file seems to be truncated in the middle", *nl, strline)
		}

		if *nl == 1 { // CRINEX VERS   / TYPE
			if strline[60:] != "CRINEX VERS   / TYPE" ||
				(strline[0:3] != "1.0" && strline[0:3] != "2.0" && strline[0:3] != "3.0") {
				return fmt.Errorf("the file format is not CRX or the version is unsupported, " +
					              "only CRX format ver 1.0, 2.0, 3.0 could be dealt with")
			}

			fmt.Sscanf(strline, "%d", crxVer)
			continue
		} else if *nl == 2 { // CRINEX PROG / DATE
			continue
		} else if *nl == 3 { // RINEX VERSION / TYPE
			if strline[60:] != "RINEX VERSION / TYPE" ||
				(strline[5] != '2' && strline[5] != '3' && strline[5] != '4') {
				return fmt.Errorf("the format version of the original RNX file is unsupported, " +
					              "only RNX format ver 2.x, 3.x or 4.x could be dealt with")
			}

			fmt.Sscanf(strline, "%d", rnxVer)
		} else if strline[60:] == "# / TYPES OF OBSERV" && strline[5] != ' ' { // for RINEX2
			fmt.Sscanf(strline, "%d", TypeNum)

			if *TypeNum <= 0 {
				return fmt.Errorf("after reading line %d, \"%s\", error occured. invalid value", *nl, strline)
			} else if *TypeNum > maxSatNum {
				return fmt.Errorf("after reading line %d, \"%s\", number of obs types exceed limit (%d)",
					              *nl, strline, maxTypeNum)
			}
		} else if strline[60:] == "SYS / # / OBS TYPES" && strline[0] != ' ' { // for RINEX3, RINEX 4
			fmt.Sscanf(strline[3:], "%d", &TypeNumGNSS[strline[0]])

			if TypeNumGNSS[strline[0]] <= 0 {
				return fmt.Errorf("after reading line %d, \"%s\", error occured. invalid value", *nl, strline)
			} else if TypeNumGNSS[strline[0]] > maxTypeNum {
				return fmt.Errorf("after reading line %d, \"%s\", number of obs types exceed limit (%d)",
					              *nl, strline, maxTypeNum)
			}
		}

		fmt.Fprintln(writer, strline)

		if strline[60:] == "END OF HEADER" {
			break
		}
	}

	if ! flag {
		err := scanner.Err()

		if err != nil {
			return fmt.Errorf("after reading line %d, \"%s\", error occured. %s", *nl, strline, err)
		} else {
			return fmt.Errorf("line \"END OF HEADER\" is missing")
		}
	}

	return nil
}
