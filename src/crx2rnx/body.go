package crx2rnx

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"
)

func body(scanner *bufio.Scanner, writer *bufio.Writer, crxVer, rnxVer int,
	TypeNum *int, TypeNumGNSS *[256]int, nl *int64) error {
	var crxEpochSym, rnxEpochSym byte
	var YearMonIdx, EventFlagIdx, SatNumIdx, SatListIdx, ShiftClock int

	if rnxVer == 2 {
		crxEpochSym = '&' // Symbols indicating beginning of an epoch in CRinex file
		rnxEpochSym = ' ' // Symbols indicating beginning of an epoch in Rinex file
		YearMonIdx = 3    // index of the space between "year" and "month" in the epoch line
		EventFlagIdx = 28 // index of the event flag in the epoch line
		SatNumIdx = 29    // index of the satellite number in the epoch line
		SatListIdx = 32   // index of the satellite listin the epoch line
		ShiftClock = 3
	} else {
		crxEpochSym = '>'
		rnxEpochSym = '>'
		YearMonIdx = 6
		EventFlagIdx = 31
		SatNumIdx = 32
		SatListIdx = 41
		ShiftClock = 6
	}

	var SatNum0, SatNum int // the number of satellites
	var SatList0, SatList [maxSatNum]TypePRN
	var SatInfoList [maxSatNum]SatInfo // the satellite map between prn and number of types
	var ClkOrder, ClkArcOrder = 0, 0
	var clk0, clk ClockFormat
	var data0, data [maxSatNum][maxTypeNum]DataFormat
	var DataFlag0, DataFlag [maxSatNum][]byte

	var StrLine string
	var ByteLine []byte
	var SkipFlag = true

	var buffer []byte

out:
	for scanner.Scan() {
		*nl++
		StrLine = scanner.Text()

		// Escape lines of CRINEX version 3
		if len(StrLine) > 0 && (crxVer == 3 && StrLine[0] == '&') {
			continue out
		}

		// something would be skipped
		if SkipFlag && (len(StrLine) <= SatNumIdx ||
			StrLine[0] != crxEpochSym || StrLine[YearMonIdx] != ' ' ||
			StrLine[YearMonIdx + 3] != ' ' || StrLine[YearMonIdx + 6] != ' ' ||
			StrLine[YearMonIdx + 9] != ' ' || StrLine[YearMonIdx + 12] != ' ' ||
			StrLine[YearMonIdx + 23] != ' ' || StrLine[YearMonIdx + 24] != ' ' ||
			! unicode.IsDigit(rune(StrLine[EventFlagIdx]))) {
			if rnxVer == 2 {
				fmt.Fprintf(writer, "%29d%3d\n%-60sCOMMENT\n", 4, 1, "  *** Something is skipped by CRX2RNX ***")
			} else {
				fmt.Fprintf(writer, ">%31d%3d\n%-60sCOMMENT\n", 4, 1, "  *** Something is skipped by CRX2RNX ***")
			}

			continue out
		}

		// Start of one epoch which initialize the differential operation
		if len(StrLine) > 0 && StrLine[0] == crxEpochSym {
			StrLine = string(rnxEpochSym) + StrLine[1:]
			SkipFlag = false

			// Event occurs
			if len(StrLine) > EventFlagIdx && (StrLine[EventFlagIdx] != '0' && StrLine[EventFlagIdx] != '1') {
				fmt.Fprintln(writer, StrLine)

				if len(StrLine) > SatNumIdx {
					var n int

					fmt.Sscanf(StrLine[SatNumIdx:], "%d", &n)

					for i := 0; i < n; i++ {
						if !scanner.Scan() {
							break
						}

						*nl++
						StrLine = scanner.Text()
						fmt.Fprintln(writer, StrLine)

						if len(StrLine) > 78 && StrLine[60:79] == "# / TYPES OF OBSERV" &&
							StrLine[5] != ' ' { // for RINEX2
							fmt.Sscanf(StrLine, "%d", TypeNum)

							if *TypeNum <= 0 {
								return fmt.Errorf("after reading line %d, \"%s\", error occured. invalid value", *nl, StrLine)
							} else if *TypeNum > maxSatNum {
								return fmt.Errorf("after reading line %d, \"%s\", number of obs types exceed limit (%d)",
									              *nl, StrLine, maxTypeNum)
							}
						} else if len(StrLine) > 78 && StrLine[60:79] == "SYS / # / OBS TYPES" { // for RINEX3
							if StrLine[0] != ' ' {
								fmt.Sscanf(StrLine[3:], "%d", &TypeNumGNSS[StrLine[0]])

								if TypeNumGNSS[StrLine[0]] <= 0 {
									return fmt.Errorf("after reading line %d, \"%s\", error occured. invalid value", *nl, StrLine)
								} else if TypeNumGNSS[StrLine[0]] > maxSatNum {
									return fmt.Errorf("after reading line %d, \"%s\", number of obs types exceed limit (%d)",
										              *nl, StrLine, maxTypeNum)
								}
							}
						}
					}
				}

				SkipFlag = true
				continue out
			}

			ByteLine = ByteLine[0:0] // initialize arc for epoch data
			SatNum0 = 0
		}

		// repair the epoch line
		repair(StrLine, &ByteLine)

		if len(ByteLine) <= SatNumIdx || ByteLine[0] != rnxEpochSym ||
			ByteLine[YearMonIdx+23] != ' ' || ByteLine[YearMonIdx+24] != ' ' ||
			! unicode.IsDigit(rune(ByteLine[YearMonIdx + 25])) {
			SkipFlag = true
			continue out
		}

		buffer = buffer[0:0] // initialize output buffer

		// get number of satellites, and set the satellite map between prn and number of types
		fmt.Sscanf(string(ByteLine[SatNumIdx:]), "%d", &SatNum)

		if SatNum <= 0 {
			return fmt.Errorf("after reading line %d, \"%s\", error occured. invalid number of satellites", *nl, StrLine)
		} else if SatNum > maxSatNum {
			return fmt.Errorf("after reading line %d, \"%s\", number of satellites exceed limit (%d)",
				              *nl, StrLine, maxSatNum)
		}

		err := setSatInfo(rnxVer, *TypeNum, TypeNumGNSS, ByteLine[SatListIdx:],
			              SatNum0, &SatList0, SatNum, &SatList, &SatInfoList)

		if err != nil {
			return fmt.Errorf("after reading line %d, \"%s\", %s", *nl, StrLine, err)
		}

		// read the clock line and recover the clock offset value
		if ! scanner.Scan() {
			SkipFlag = true
			continue out
		}

		*nl ++
		StrLine = scanner.Text()

		err = readClock(&ClkArcOrder, &ClkOrder, StrLine, &clk)

		if err != nil {
			return fmt.Errorf("after reading line %d, \"%s\", %s", *nl, StrLine, err)
		}

		if len(StrLine) > 0 {
			// Recover the clock offset value
			repairClock(ClkArcOrder, &ClkOrder, &clk0, &clk)
		}

		// read the differenced observation data and recover them
		var TmpFlag [maxSatNum][]byte

		for i := 0; i < SatNum; i++ {
			if ! scanner.Scan() {
				SkipFlag = true
				continue out
			}

			*nl ++
			StrLine = scanner.Text()

			err = readData(StrLine, SatInfoList[i], &TmpFlag[i], &data0, &data[i])

			if err != nil {
				if err.Error() == "skip" {
					SkipFlag = true
					continue out
				} else {
					return fmt.Errorf("after reading line %d, \"%s\", %s", *nl, StrLine, err)
				}
			}

			// Recover the observation data
			DataFlag[i] = []byte{}

			repairData(rnxVer, SatInfoList[i], TmpFlag[i], &DataFlag0, &DataFlag[i],
				       &data0, &data[i])
		}

		// print epoch and clock offset
		if rnxVer == 2 {
			if ClkOrder >= 0 {
				buffer = append(buffer, fmt.Sprintf("%-68.68s", ByteLine)...)

				err = printClock(clk.upper[ClkOrder], clk.lower[ClkOrder], ShiftClock, &buffer)

				if err != nil {
					return fmt.Errorf("after reading line %d, \"%s\", failed to print clock offset, %s",
						              *nl, StrLine, err)
				}
			} else {
				buffer = append(buffer, fmt.Sprintf("%.68s\n", ByteLine)...)
			}

			for i, idx := SatNum-12, 68; i > 0; i, idx = i-12, idx+36 {
				tmpStr := fmt.Sprintf("%32s%.36s", " ", ByteLine[idx:])
				buffer = append(buffer, strings.TrimRight(tmpStr, " ")...)
				buffer = append(buffer, '\n')
			}
		} else {
			if ClkOrder >= 0 {
				buffer = append(buffer, fmt.Sprintf("%.41s", ByteLine)...)

				err = printClock(clk.upper[ClkOrder], clk.lower[ClkOrder], ShiftClock, &buffer)

				if err != nil {
					return fmt.Errorf("after reading line %d, \"%s\", failed to print clock offset, %s",
						              *nl, StrLine, err)
				}
			} else {
				tmpStr := fmt.Sprintf("%.41s", ByteLine)
				buffer = append(buffer, strings.TrimRight(tmpStr, " ")...)
				buffer = append(buffer, '\n')
			}
		}

		// print observation data
		for i := 0; i < SatNum; i++ {
			err = printData(crxVer, rnxVer, &SatList[i], SatInfoList[i].TypeNum,
				&DataFlag[i], &data[i], &buffer)

			if err != nil {
				return fmt.Errorf("after reading line %d, \"%s\", %s"+
					              "failed to print observation data", *nl, StrLine, err)
			}
		}

		_, err = writer.Write(buffer)

		if err != nil {
			return fmt.Errorf("after reading line %d, \"%s\", "+
				              "failed to write to the output file", *nl, StrLine)
		}

		// store the values
		SatNum0 = SatNum
		clk0 = clk

		for i := 0; i < SatNum; i++ {
			SatList0[i] = SatList[i]
			DataFlag0[i] = append(DataFlag0[i][0:0], DataFlag[i]...)

			for j := 0; j < SatInfoList[i].TypeNum; j++ {
				data0[i][j] = data[i][j]
			}
		}
	}

	return nil
}
