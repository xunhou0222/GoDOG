package crx2rnx

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

/***** FUNCTION ********************************/

func header(scanner *bufio.Scanner, writer *bufio.Writer,
	crxVer, rnxVer *int, TypeNumGNSS map[byte]int, nl *int64) (err error) {
	var line, kw string
	var num int

	for {
		if !scanner.Scan() {
			err = scanner.Err()

			if err == nil {
				return errors.New(`no "END OF HEADER"`)
			} else {
				return fmt.Errorf(`after reading line %d, "%s", error occured. %s`, *nl, line, err)
			}
		}

		*nl++
		line = strings.TrimRight(scanner.Text(), " \t")

		if len(line) <= 60 {
			return fmt.Errorf(`after reading line %d, "%s", the file was truncated`, *nl, line)
		}

		kw = line[60:]

		if kw == "CRINEX VERS   / TYPE" {
			if line[0:3] != "1.0" && line[0:3] != "3.0" && line[0:3] != "3.1" {
				return errors.New("invalid format, only CRINEX version 1.0, 3.0, 3.1 could be dealt with")
			}

			*crxVer = int(line[0] - '0')
			continue
		} else if kw == "CRINEX PROG / DATE" {
			continue
		} else if kw == "RINEX VERSION / TYPE" {
			*rnxVer = int(line[5] - '0')

			if *rnxVer != 2 && *rnxVer != 3 && *rnxVer != 4 {
				return errors.New("unsupported RINEX version, only RINEX version 2.x, 3.x or 4.x could be dealt with")
			}
		} else if kw == "# / TYPES OF OBSERV" && line[5] != ' ' { // for RINEX 2
			fmt.Sscanf(line, "%d", &num)

			if num <= 0 {
				return fmt.Errorf(`after reading line %d, "%s", invalid number of obs types, "%d"`, *nl, line, num)
			}

			TypeNumGNSS[0] = num
		} else if kw == "SYS / # / OBS TYPES" && line[0] != ' ' { // for RINEX 3, RINEX 4
			fmt.Sscanf(line[3:], "%d", &num)

			if num <= 0 {
				return fmt.Errorf(`after reading line %d, "%s", error occured. invalid number of obs types`, *nl, line)
			}

			TypeNumGNSS[line[0]] = num
		}

		fmt.Fprintln(writer, line)

		if kw == "END OF HEADER" {
			break
		}
	}

	return nil
}

/***********************************************/
