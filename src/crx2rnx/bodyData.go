package crx2rnx

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

func readData(line string, info _SatInfo, flag *[]byte, data0 [][]_DataFormat, data *[]_DataFormat) error {
	var idx int = 0
	var idx1, length int

	if len(*data) < info.TypeNum {
		*data = make([]_DataFormat, info.TypeNum)
	}

	for i := 0; i < info.TypeNum; i++ {
		if idx >= len(line) || line[idx] == ' ' {
			(*data)[i].order = -1
			(*data)[i].arcOrder = -1 // < 0 means that the field is blank
			idx++
		} else {
			if idx+1 < len(line) && line[idx+1] == '&' { // arc initialization
				(*data)[i].order = -1 // < 0 means that the field is blank
				fmt.Sscanf(string(line[idx:]), "%d&", &(*data)[i].arcOrder)
				idx += 2

				if (*data)[i].arcOrder > _MAX_DIFF_ORDER {
					return fmt.Errorf("exceed maximum order of difference (%d)", _MAX_DIFF_ORDER)
				}
			} else if info.OldIdx < 0 || data0[info.OldIdx][i].arcOrder < 0 {
				return fmt.Errorf("readData_skip")
			} else {
				(*data)[i].order = data0[info.OldIdx][i].order
				(*data)[i].arcOrder = data0[info.OldIdx][i].arcOrder
			}

			length = strings.IndexByte(line[idx:], ' ')

			if length < 0 {
				length = len(line[idx:])
			}

			idx1 = idx + length

			if line[idx] == '-' {
				length--
			}

			if length < 6 {
				(*data)[i].upper[0] = 0
				fmt.Sscanf(line[idx:], "%d", &(*data)[i].lower[0])
			} else {
				fmt.Sscanf(line[idx:idx1-5], "%d", &(*data)[i].upper[0])
				fmt.Sscanf(line[idx1-5:idx1], "%d", &(*data)[i].lower[0])

				if (*data)[i].upper[0] < 0 {
					(*data)[i].lower[0] *= -1
				}
			}

			idx = idx1 + 1
		}
	}

	if idx >= len(line) {
		idx = len(line)
	}

	if idx < 0 {
		idx = 0
	}

	*flag = append(*flag, line[idx:]...)

	return nil
}

func repairData(rnxVer int, info _SatInfo, dflag []byte, flag0 [][]byte, flag *[]byte,
	data0 [][]_DataFormat, data *[]_DataFormat) {
	// repair the data flags
	if info.OldIdx < 0 { // new satellite
		if rnxVer < 3 {
			*flag = append(*flag, fmt.Sprintf("%-*s", 2*info.TypeNum, dflag)...)
		}
	} else {
		*flag = append(*flag, flag0[info.OldIdx]...)
	}

	repair(dflag, flag)

	// recover the observation data
	for i := 0; i < info.TypeNum; i++ {
		if (*data)[i].arcOrder >= 0 {
			if (*data)[i].order < (*data)[i].arcOrder {
				(*data)[i].order++

				for k1, k2 := 0, 1; k1 < (*data)[i].order; k1, k2 = k1+1, k2+1 {
					(*data)[i].upper[k2] = (*data)[i].upper[k1] + data0[info.OldIdx][i].upper[k1]
					(*data)[i].lower[k2] = (*data)[i].lower[k1] + data0[info.OldIdx][i].lower[k1]
					(*data)[i].upper[k2] += (*data)[i].lower[k2] / 100000 // to avoid overflow
					(*data)[i].lower[k2] %= 100000
				}
			} else {
				for k1, k2 := 0, 1; k1 < (*data)[i].order; k1, k2 = k1+1, k2+1 {
					(*data)[i].upper[k2] = (*data)[i].upper[k1] + data0[info.OldIdx][i].upper[k2]
					(*data)[i].lower[k2] = (*data)[i].lower[k1] + data0[info.OldIdx][i].lower[k2]
					(*data)[i].upper[k2] += (*data)[i].lower[k2] / 100000 // to avoid overflow
					(*data)[i].lower[k2] %= 100000
				}
			}

			// make signs of data[i].upper and data[i].lower the same (or zero)
			odr := (*data)[i].order

			if (*data)[i].upper[odr] < 0 && (*data)[i].lower[odr] > 0 {
				(*data)[i].upper[odr]++
				(*data)[i].lower[odr] -= 100000
			} else if (*data)[i].upper[odr] > 0 && (*data)[i].lower[odr] < 0 {
				(*data)[i].upper[odr]--
				(*data)[i].lower[odr] += 100000
			}
		}
	}
}

func printData(writer *bufio.Writer, crxVer, rnxVer int, prn _TypePRN, TypeNum int,
	flag []byte, data []_DataFormat) error {
	var idx int
	var bs []byte

	if rnxVer >= 3 {
		writer.Write(prn[:])
	}

	for i := 0; i < TypeNum; i++ {
		if i*2 >= len(flag) {
			flag = append(flag, ' ')
		}

		if i*2+1 >= len(flag) {
			flag = append(flag, ' ')
		}

		if data[i].arcOrder >= 0 {
			var odr = data[i].order

			if data[i].upper[odr] != 0 { // e.g., 123.456, -123.456
				if data[i].lower[odr] < 0 {
					bs = fmt.Appendf(bs, "%8d %5.5d%c%c", data[i].upper[odr], -data[i].lower[odr], flag[2*i], flag[2*i+1])
				} else {
					bs = fmt.Appendf(bs, "%8d %5.5d%c%c", data[i].upper[odr], data[i].lower[odr], flag[2*i], flag[2*i+1])
				}

				idx = len(bs)

				bs[idx-8] = bs[idx-7]
				bs[idx-7] = bs[idx-6]

				if data[i].upper[odr] > 99999999 || data[i].upper[odr] < -9999999 {
					return fmt.Errorf("observation data out of range")
				}
			} else {
				if data[i].lower[odr] < 0 {
					bs = fmt.Appendf(bs, "         %5.5d%c%c", -data[i].lower[odr], flag[2*i], flag[2*i+1])
				} else {
					bs = fmt.Appendf(bs, "         %5.5d%c%c", data[i].lower[odr], flag[2*i], flag[2*i+1])
				}

				idx = len(bs)

				if bs[idx-7] != '0' { // e.g., 12.345, -2.345
					bs[idx-8] = bs[idx-7]
					bs[idx-7] = bs[idx-6]

					if data[i].lower[odr] < 0 {
						bs[idx-9] = '-'
					}
				} else if bs[idx-6] != '0' { // e.g., 1.234, -1.234
					bs[idx-7] = bs[idx-6]

					if data[i].lower[odr] < 0 {
						bs[idx-8] = '-'
					} else {
						bs[idx-8] = ' '
					}
				} else { // e.g., .123, -.123
					if data[i].lower[odr] < 0 {
						bs[idx-7] = '-'
					} else {
						bs[idx-7] = ' '
					}
				}
			}

			bs[idx-6] = '.'
		} else {
			if crxVer == 1 { // CRINEX 1 assumes that flags are always blank if data field is blank
				bs = append(bs, bytes.Repeat([]byte{' '}, 16)...)
				flag[i*2] = ' '
				flag[i*2+1] = ' '
			} else { //CRINEX 3 evaluate flags independently
				bs = append(bs, bytes.Repeat([]byte{' '}, 14)...)
				bs = append(bs, flag[i*2], flag[i*2+1])
			}
		}

		if i+1 == TypeNum || (rnxVer == 2 && (i+1)%5 == 0) {
			bs = bytes.TrimRight(bs, " ")
			writer.Write(bs)
			writer.WriteByte('\n')
			bs = bs[:0]
		}
	}

	return nil
}
