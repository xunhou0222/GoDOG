package crx2rnx

import (
	"bytes"
	"fmt"
	"strings"
)

func repair(StrLine string, ByteLine *[]byte) {
	var i, j int

	for i, j = 0, 0; i < len(StrLine) && j < len(*ByteLine); i, j = i+1, j+1 {
		if StrLine[i] == ' ' {
			continue
		} else if StrLine[i] == '&' {
			(*ByteLine)[j] = ' '
		} else {
			(*ByteLine)[j] = StrLine[i]
		}
	}

	if i < len(StrLine) {
		*ByteLine = append((*ByteLine)[0:j], StrLine[i:]...)

		for ; j < len(*ByteLine); j++ {
			if (*ByteLine)[j] == '&' {
				(*ByteLine)[j] = ' '
			}
		}
	}
}

func setSatInfo(RnxVer, TypeNum int, TypeNumGNSS *[256]int, ByteLine []byte,
	SatNum0 int, SatList0 *[maxSatNum]TypePRN, SatNum int, SatList *[maxSatNum]TypePRN,
	SatInfoList *[maxSatNum]SatInfo) error {
	var prn TypePRN

	for i := 0; i < SatNum; i++ {
		if 3*i+2 >= len(ByteLine) {
			return fmt.Errorf("the satellite list seems to be truncated in the middle")
		}

		prn[0] = ByteLine[3*i]
		prn[1] = ByteLine[3*i+1]
		prn[2] = ByteLine[3*i+2]

		SatInfoList[i] = SatInfo{0, -1}

		if RnxVer == 2 {
			SatInfoList[i].TypeNum = TypeNum
		} else {
			if TypeNumGNSS[prn[0]] == 0 {
				return fmt.Errorf("a GNSS type not defined in the header is found")
			}

			SatInfoList[i].TypeNum = TypeNumGNSS[prn[0]]
		}

		for j := 0; j < SatNum0; j++ {
			if SatList0[j] == prn {
				SatInfoList[i].OldIdx = j
			}
		}

		SatList[i] = prn
	}

	return nil
}

func readClock(ClkArcOrder, ClkOrder *int, StrLine string, clk *ClockFormat) error {
	if len(StrLine) == 0 {
		*ClkOrder = -1
	} else {
		var idx0 int = 0

		if len(StrLine) >= 2 && StrLine[1] == '&' {
			fmt.Sscanf(StrLine, "%d&", ClkArcOrder)

			if *ClkArcOrder > maxDiffOrder {
				return fmt.Errorf("exceed maximum order of difference (%d)", maxDiffOrder)
			}

			*ClkOrder = -1
			idx0 += 2
		}

		var idx = idx0

		if StrLine[idx0] == '-' {
			idx++
		}

		if len(StrLine[idx:]) < 9 {
			clk.upper[0] = 0
			fmt.Sscanf(StrLine[idx0:], "%d", &clk.lower[0])
		} else {
			fmt.Sscanf(StrLine[len(StrLine)-8:], "%d", &clk.lower[0])
			fmt.Sscanf(StrLine[idx0:len(StrLine)-8], "%d", &clk.upper[0])

			if clk.upper[0] < 0 {
				clk.lower[0] *= -1
			}
		}
	}

	return nil
}

func repairClock(clkArcOrder int, ClkOrder *int, clk0 *ClockFormat, clk *ClockFormat) {
	if *ClkOrder < clkArcOrder {
		*ClkOrder++
	}

	for i, j := 0, 1; i < *ClkOrder; i, j = i+1, j+1 {
		clk.upper[j] = clk.upper[i] + clk0.upper[i]
		clk.lower[j] = clk.lower[i] + clk0.lower[i]
		clk.upper[j] += clk.lower[j] / 100000000 // to avoid overflow
		clk.lower[j] %= 100000000
	}
}

func readData(StrLine string, info SatInfo, flag *[]byte, data0 *[maxSatNum][maxTypeNum]DataFormat,
	data *[maxTypeNum]DataFormat) error {
	var idx int = 0
	var idx1, length int

	for i := 0; i < info.TypeNum; i++ {
		if idx >= len(StrLine) || StrLine[idx] == ' ' {
			data[i].order = -1
			data[i].arcOrder = -1 // < 0 means that the field is blank
			idx++
		} else {
			if idx+1 < len(StrLine) && StrLine[idx+1] == '&' { // arc initialization
				data[i].order = -1 // < 0 means that the field is blank
				fmt.Sscanf(StrLine[idx:], "%d&", &data[i].arcOrder)
				idx += 2

				if data[i].arcOrder > maxDiffOrder {
					return fmt.Errorf("exceed maximum order of difference (%d)", maxDiffOrder)
				}
			} else if info.OldIdx < 0 || data0[info.OldIdx][i].arcOrder < 0 {
				return fmt.Errorf("skip")
			} else {
				data[i].order = data0[info.OldIdx][i].order
				data[i].arcOrder = data0[info.OldIdx][i].arcOrder
			}

			length = strings.IndexByte(StrLine[idx:], ' ')

			if length < 0 {
				length = len(StrLine[idx:])
			}

			idx1 = idx + length

			if StrLine[idx] == '-' {
				length--
			}

			if length < 6 {
				data[i].upper[0] = 0
				fmt.Sscanf(StrLine[idx:], "%d", &data[i].lower[0])
			} else {
				fmt.Sscanf(StrLine[idx:idx1-5], "%d", &data[i].upper[0])
				fmt.Sscanf(StrLine[idx1-5:idx1], "%d", &data[i].lower[0])

				if data[i].upper[0] < 0 {
					data[i].lower[0] *= -1
				}
			}

			idx = idx1 + 1
		}
	}

	if idx >= len(StrLine) {
		idx = len(StrLine)
	}

	if idx < 0 {
		idx = 0
	}

	*flag = append(*flag, StrLine[idx:]...)

	return nil
}

func repairData(rnxVer int, info SatInfo, dflag []byte, flag0 *[maxSatNum][]byte, flag *[]byte,
	data0 *[maxSatNum][maxTypeNum]DataFormat, data *[maxTypeNum]DataFormat) {
	// repair the data flags
	if info.OldIdx < 0 { // new satellite
		if rnxVer < 3 {
			tmpStr := fmt.Sprintf("%-*s", 2*info.TypeNum, dflag)
			*flag = append(*flag, tmpStr...)
		}
	} else {
		*flag = append(*flag, flag0[info.OldIdx]...)
	}

	repair(string(dflag), flag)

	// recover the observation data
	for i := 0; i < info.TypeNum; i++ {
		if data[i].arcOrder >= 0 {
			if data[i].order < data[i].arcOrder {
				data[i].order++

				for k1, k2 := 0, 1; k1 < data[i].order; k1, k2 = k1+1, k2+1 {
					data[i].upper[k2] = data[i].upper[k1] + data0[info.OldIdx][i].upper[k1]
					data[i].lower[k2] = data[i].lower[k1] + data0[info.OldIdx][i].lower[k1]
					data[i].upper[k2] += data[i].lower[k2] / 100000 // to avoid overflow
					data[i].lower[k2] %= 100000
				}
			} else {
				for k1, k2 := 0, 1; k1 < data[i].order; k1, k2 = k1+1, k2+1 {
					data[i].upper[k2] = data[i].upper[k1] + data0[info.OldIdx][i].upper[k2]
					data[i].lower[k2] = data[i].lower[k1] + data0[info.OldIdx][i].lower[k2]
					data[i].upper[k2] += data[i].lower[k2] / 100000 // to avoid overflow
					data[i].lower[k2] %= 100000
				}
			}

			// make signs of data[i].upper and data[i].lower the same (or zero)
			odr := data[i].order

			if data[i].upper[odr] < 0 && data[i].lower[odr] > 0 {
				data[i].upper[odr]++
				data[i].lower[odr] -= 100000
			} else if data[i].upper[odr] > 0 && data[i].lower[odr] < 0 {
				data[i].upper[odr]--
				data[i].lower[odr] += 100000
			}
		}
	}
}

func printClock(upper, lower int64, shift int, buffer *[]byte) error {
	var tmpStr string

	// add ond more digit to handle '-0'(RINEX2) or '-0000'(RINEX3)
	// AT LEAST fractional parts are filled with 0
	if lower < 0 {
		tmpStr = fmt.Sprintf("%.*d", shift + 1, upper*10 - 1)
	} else {
		tmpStr = fmt.Sprintf("%.*d", shift + 1, upper*10 + 1)
	}

	var n = len(tmpStr) - 1 // number of digits excluding the additional digit
	var idx = n - shift
	*buffer = append(*buffer, fmt.Sprintf("  .%s", tmpStr[idx:])...)

	if n > shift {
		idx--
		var idx1 = len(*buffer) - shift - 2
		(*buffer)[idx1] = tmpStr[idx]

		if n > shift+1 {
			(*buffer)[idx1-1] = tmpStr[idx-1]

			if n > shift+2 {
				return fmt.Errorf("clock offset out of range")
			}
		}
	}

	if lower < 0 {
		tmpStr = fmt.Sprintf("%8.8d", -lower)
	} else {
		tmpStr = fmt.Sprintf("%8.8d", lower)
	}

	*buffer = append(*buffer, tmpStr...)

	return nil
}

func printData(crxVer, rnxVer int, prn *TypePRN, TypeNum int, flag *[]byte, 
	           data *[maxTypeNum]DataFormat, buffer *[]byte) error {
	var tmpStr string
	var idx int

	if rnxVer >= 3 {
		*buffer = append(*buffer, prn[:]...)
	}

	for i := 0; i < TypeNum; i++ {
		if i*2 >= len(*flag) {
			*flag = append(*flag, ' ')
		}

		if i*2+1 >= len(*flag) {
			*flag = append(*flag, ' ')
		}

		if data[i].arcOrder >= 0 {
			var odr = data[i].order

			if data[i].upper[odr] != 0 { // e.g., 123.456, -123.456
				if data[i].lower[odr] < 0 {
					tmpStr = fmt.Sprintf("%8d %5.5d%c%c", data[i].upper[odr], -data[i].lower[odr],
						(*flag)[2*i], (*flag)[2*i+1])
				} else {
					tmpStr = fmt.Sprintf("%8d %5.5d%c%c", data[i].upper[odr], data[i].lower[odr],
						(*flag)[2*i], (*flag)[2*i+1])
				}

				*buffer = append(*buffer, tmpStr...)
				idx = len(*buffer)

				(*buffer)[idx-8] = (*buffer)[idx-7]
				(*buffer)[idx-7] = (*buffer)[idx-6]

				if data[i].upper[odr] > 99999999 || data[i].upper[odr] < -9999999 {
					return fmt.Errorf("observation data out of range")
				}
			} else {
				if data[i].lower[odr] < 0 {
					tmpStr = fmt.Sprintf("         %5.5d%c%c", -data[i].lower[odr], (*flag)[2*i], (*flag)[2*i+1])
				} else {
					tmpStr = fmt.Sprintf("         %5.5d%c%c", data[i].lower[odr], (*flag)[2*i], (*flag)[2*i+1])
				}

				*buffer = append(*buffer, tmpStr...)
				idx = len(*buffer)

				if (*buffer)[idx-7] != '0' { // e.g., 12.345, -2.345
					(*buffer)[idx-8] = (*buffer)[idx-7]
					(*buffer)[idx-7] = (*buffer)[idx-6]

					if data[i].lower[odr] < 0 {
						(*buffer)[idx-9] = '-'
					}
				} else if (*buffer)[idx-6] != '0' { // e.g., 1.234, -1.234
					(*buffer)[idx-7] = (*buffer)[idx-6]

					if data[i].lower[odr] < 0 {
						(*buffer)[idx-8] = '-'
					} else {
						(*buffer)[idx-8] = ' '
					}
				} else { // e.g., .123, -.123
					if data[i].lower[odr] < 0 {
						(*buffer)[idx-7] = '-'
					} else {
						(*buffer)[idx-7] = ' '
					}
				}
			}

			(*buffer)[idx-6] = '.'
		} else {
			if crxVer == 1 { // CRINEX 1 assumes that flags are always
				*buffer = append(*buffer, "                "...) // blank if data field is blank
				(*flag)[i*2] = ' '
				(*flag)[i*2+1] = ' '
			} else { //CRINEX 3 evaluate flags independently
				tmpStr = fmt.Sprintf("              %c%c", (*flag)[i*2], (*flag)[i*2+1])
				*buffer = append(*buffer, tmpStr...)
			}
		}

		if i+1 == TypeNum || (rnxVer == 2 && (i+1)%5 == 0) {
			*buffer = bytes.TrimRight(*buffer, " ")
			*buffer = append(*buffer, '\n')
		}
	}

	return nil
}
