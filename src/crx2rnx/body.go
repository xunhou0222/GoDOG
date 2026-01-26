package crx2rnx

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

const (
	_MAX_DIFF_ORDER = 5
	_MAX_LINE_LEN   = 1024
	_MAX_SAT_NUM    = 100
)

type _TypePRN [3]byte

// structure for fields of clock offset
type _ClockFormat struct {
	upper [_MAX_DIFF_ORDER + 1]int64 // upper X digits for each difference order
	lower [_MAX_DIFF_ORDER + 1]int64 // lower 8 digits
}

// structure for fields of observation records
type _DataFormat struct {
	upper    [_MAX_DIFF_ORDER + 1]int64 // upper X digits for each difference order
	lower    [_MAX_DIFF_ORDER + 1]int64 // lower 5 digits
	order    int
	arcOrder int
}

type _SatInfo struct {
	TypeNum int // number of observation types
	OldIdx  int // The index in the old satellite list (-1 if is a new satellite)
}

func body(scanner *bufio.Scanner, writer *bufio.Writer, crxVer, rnxVer int,
	TypeNumGNSS map[byte]int, nl *int64) (err error) {
	var (
		crxEpochSym, rnxEpochSym                                  byte
		yearMonIdx, eventFlagIdx, satNumIdx, satListIdx, clkShift int
	)

	if rnxVer == 2 { // RINEX 2
		crxEpochSym = '&' // Symbols indicating beginning of an epoch in CRINEX file
		rnxEpochSym = ' ' // Symbols indicating beginning of an epoch in RINEX file
		yearMonIdx = 3    // index of the space between "year" and "month" in the epoch line
		eventFlagIdx = 28 // index of the event flag in the epoch line
		satNumIdx = 29    // index of the satellite number in the epoch line
		satListIdx = 32   // index of the satellite list in the epoch line
		clkShift = 1
	} else {
		crxEpochSym = '>'
		rnxEpochSym = '>'
		yearMonIdx = 6
		eventFlagIdx = 31
		satNumIdx = 32
		satListIdx = 41
		clkShift = 4
	}

	var (
		line                  string
		lineSb                []byte
		lineSbAll             = make([]byte, 0, _MAX_LINE_LEN)
		mustInit              = true
		satNum                int
		satList               = make([]_TypePRN, 0, _MAX_SAT_NUM)
		satList0              = make([]_TypePRN, 0, _MAX_SAT_NUM)
		satInfoList           = make([]_SatInfo, 0, _MAX_SAT_NUM)
		clkArcOrder, clkOrder int
		clkSb, picoSecSb      []byte
		picoSecSbAll          = make([]byte, 0, _MAX_LINE_LEN)
		clk0, clk             _ClockFormat
		data0                 = make([][]_DataFormat, 0, _MAX_SAT_NUM) // [i][j]_DataFormat, i for satellite, j for observation type
		data                  = make([][]_DataFormat, 0, _MAX_SAT_NUM) // [i][j]_DataFormat, i for satellite, j for observation type
		dataFlag0             = make([][]byte, 0, _MAX_SAT_NUM)        // [i][j]byte, i for satellite, j for observation type
		dataFlag              = make([][]byte, 0, _MAX_SAT_NUM)        // [i][j]byte, i for satellite, j for observation type
		tmpFlag               = make([][]byte, 0, _MAX_SAT_NUM)        // [i][j]byte, i for satellite, j for observation type
	)

outer:
	for scanner.Scan() {
		*nl++
		lineSb = bytes.TrimRight(scanner.Bytes(), " \t")
		line = string(lineSb)

		// skip escape lines of CRINEX 3
		if crxVer == 3 && len(line) > 0 && line[0] == '&' {
			continue outer
		}

		// skip abnormal lines, e.g., the next epoch after an event flag is not initialized
		if mustInit && len(line) > 0 && line[0] != crxEpochSym {
			comment(writer, rnxVer, "  *** Abnormal line, skipped by CRX2RNX ***")
			continue outer
		}

		// event flag or initialization of the differential operation for epoch and satellite list
		if len(line) > 0 && line[0] == crxEpochSym {
			lineSb[0] = rnxEpochSym

			// event occurs
			if len(line) > eventFlagIdx && (line[eventFlagIdx] != '0' && line[eventFlagIdx] != '1') {
				writer.Write(lineSb)
				writer.WriteByte('\n')

				if len(line) > satNumIdx {
					var count, iTmp int
					fmt.Sscanf(line[satNumIdx:], "%d", &count)

					for i := 0; i < count && scanner.Scan(); i++ {
						*nl++
						lineSb = bytes.TrimRight(scanner.Bytes(), " \t")
						line = string(lineSb)
						writer.Write(lineSb)
						writer.WriteByte('\n')

						if len(line) > 78 && line[60:] == "# / TYPES OF OBSERV" && line[5] != ' ' { // for RINEX2
							fmt.Sscanf(line, "%d", iTmp)

							if iTmp <= 0 {
								return fmt.Errorf(`after reading line %d, "%s", error occured. invalid value`, *nl, line)
							}

							TypeNumGNSS[0] = iTmp
						} else if len(line) > 78 && line[60:79] == "SYS / # / OBS TYPES" && line[0] != ' ' { // for RINEX3
							fmt.Sscanf(line[3:], "%d", &iTmp)

							if iTmp <= 0 {
								return fmt.Errorf(`after reading line %d, "%s", error occured. invalid value`, *nl, line)
							}

							TypeNumGNSS[line[0]] = iTmp
						}
					}
				}

				mustInit = true
				continue outer
			}

			// initialization
			lineSbAll = lineSbAll[0:0] // initialize epoch
			satList0 = satList0[0:0]   // initialize satellite list
			mustInit = false
		}

		// repair the line of epoch and satellite list
		repair(lineSb, &lineSbAll)

		if len(lineSbAll) <= satNumIdx || lineSbAll[0] != rnxEpochSym ||
			lineSbAll[yearMonIdx+23] != ' ' || lineSbAll[yearMonIdx+24] != ' ' ||
			lineSbAll[yearMonIdx+25] < '0' || lineSbAll[yearMonIdx+25] > '9' {
			return fmt.Errorf(`after reading line %d, "%s", error occured. invalid epoch line`, *nl, line)
		}

		// get number of satellites, and set the satellite map between prn and number of types
		if _, err = fmt.Sscanf(string(lineSbAll[satNumIdx:]), "%d", &satNum); err != nil || satNum < 0 {
			return fmt.Errorf(`after reading line %d, "%s", error occured. invalid number of satellites`, *nl, line)
		}

		if len(lineSbAll) <= satListIdx {
			return fmt.Errorf(`after reading line %d, "%s", invalid satellite list`, *nl, line)
		}

		if setSatInfo(rnxVer, TypeNumGNSS, lineSbAll[satListIdx:], satNum, satList0, &satList, &satInfoList) != nil {
			return fmt.Errorf(`after reading line %d, "%s", %s`, *nl, line, err)
		}

		// read the clock line and recover the clock offset value
		if !scanner.Scan() {
			comment(writer, rnxVer, " *** Abnormal clock line, skipped by CRX2RNX ***")
			continue outer
		}

		*nl++
		lineSb = bytes.TrimRight(scanner.Bytes(), " \t")
		line = string(lineSb)

		if readClock(lineSb, &clkArcOrder, &clkOrder, &clk, &clkSb, &picoSecSb) != nil {
			return fmt.Errorf(`after reading line %d, "%s", %s`, *nl, line, err)
		}

		if len(clkSb) > 0 {
			repairClock(clkArcOrder, &clkOrder, &clk0, &clk) // recover the clock offset value
		}

		if len(picoSecSb) > 0 {
			repair(picoSecSb, &picoSecSbAll)
		}

		// read the differenced observation data and recover them
		if len(tmpFlag) < satNum {
			if cap(tmpFlag) >= satNum {
				tmpFlag = tmpFlag[:satNum]
			} else {
				tmpFlag = make([][]byte, satNum)
			}
		}

		if len(data) < satNum {
			if cap(data) >= satNum {
				data = data[:satNum]
			} else {
				data = make([][]_DataFormat, satNum)
			}
		}

		if len(dataFlag) < satNum {
			if cap(dataFlag) >= satNum {
				dataFlag = dataFlag[:satNum]
			} else {
				dataFlag = make([][]byte, satNum)
			}
		}

		for i := 0; i < satNum; i++ {
			if !scanner.Scan() {
				return fmt.Errorf(`after reading line %d, "%s", invalid data line`, *nl, line)
			}

			*nl++
			lineSb = bytes.TrimRight(scanner.Bytes(), " \t")
			line = string(lineSb)
			tmpFlag[i] = tmpFlag[i][0:0]

			if err = readData(line, satInfoList[i], &tmpFlag[i], data0, &data[i]); err != nil {
				if err.Error() == "readData_skip" {
					continue outer
				} else {
					return fmt.Errorf(`after reading line %d, "%s", %s`, *nl, line, err)
				}
			}

			// recover the observation data
			dataFlag[i] = dataFlag[i][0:0]
			repairData(rnxVer, satInfoList[i], tmpFlag[i], dataFlag0, &dataFlag[i], data0, &data[i])
		}

		// print epoch and clock offset
		if rnxVer == 2 {
			if clkOrder >= 0 {
				fmt.Fprintf(writer, "%-68.68s", lineSbAll)

				if err = printClock(writer, clk.upper[clkOrder], clk.lower[clkOrder], clkShift); err != nil {
					return fmt.Errorf(`after reading line %d, "%s", failed to print clock offset. %s`, *nl, line, err)
				}
			} else {
				fmt.Fprintf(writer, "%.68s\n", lineSbAll)
			}

			for i, idx := satNum-12, 68; i > 0; i, idx = i-12, idx+36 {
				tmpStr := fmt.Sprintf("%32s%.36s", " ", lineSbAll[idx:])
				fmt.Fprintln(writer, strings.TrimRight(tmpStr, " "))
			}
		} else {
			if clkOrder >= 0 {
				fmt.Fprintf(writer, "%.41s", lineSbAll)

				if err = printClock(writer, clk.upper[clkOrder], clk.lower[clkOrder], clkShift); err != nil {
					return fmt.Errorf(`after reading line %d, "%s", failed to print clock offset, %s`, *nl, line, err)
				}
			} else {
				tmpStr := fmt.Sprintf("%.41s", lineSbAll)
				fmt.Fprintln(writer, strings.TrimRight(tmpStr, " "))
			}
		}

		// print observation data
		for i := 0; i < satNum; i++ {
			if err = printData(writer, crxVer, rnxVer, satList[i], satInfoList[i].TypeNum, dataFlag[i], data[i]); err != nil {
				return fmt.Errorf(`after reading line %d, "%s", failed to print observation data. %s`, *nl, line, err)
			}
		}

		// store the values
		clk0 = clk

		if len(satList0) < satNum {
			if cap(satList0) >= satNum {
				satList0 = satList0[:satNum]
			} else {
				satList0 = make([]_TypePRN, satNum)
			}
		}

		if len(data0) < satNum {
			data0 = make([][]_DataFormat, satNum)
		}

		if len(dataFlag0) < satNum {
			dataFlag0 = make([][]byte, satNum)
		}

		for i := 0; i < satNum; i++ {
			satList0[i] = satList[i]
			data0[i] = append(data0[i][0:0], data[i]...)
			dataFlag0[i] = append(dataFlag0[i][0:0], dataFlag[i]...)
		}
	}

	if err = scanner.Err(); err != nil {
		return fmt.Errorf(`after reading line %d, "%s", error occured. %s`, *nl, line, err)
	}

	return nil
}
