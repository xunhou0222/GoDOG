package crx2rnx

import (
	"bufio"
	"bytes"
	"fmt"
)

func readClock(lineSb []byte, clkArcOrder, clkOrder *int, clk *_ClockFormat, clkSb *[]byte, picoSecSb *[]byte) error {
	*clkSb, *picoSecSb, _ = bytes.Cut(lineSb, []byte{' '})
	clkStr := string(*clkSb)

	if len(*clkSb) == 0 {
		*clkOrder = -1
	} else {
		idx0 := 0

		if len(*clkSb) >= 2 && (*clkSb)[1] == '&' {
			fmt.Sscanf(clkStr, "%d&", clkArcOrder)

			if *clkArcOrder > _MAX_DIFF_ORDER {
				return fmt.Errorf("exceed maximum order of difference (%d)", _MAX_DIFF_ORDER)
			}

			*clkOrder = -1
			idx0 += 2
		}

		idx := idx0

		if clkStr[idx0] == '-' {
			idx++
		}

		if len(clkStr[idx:]) < 9 {
			clk.upper[0] = 0
			fmt.Sscanf(clkStr[idx0:], "%d", &clk.lower[0])
		} else {
			fmt.Sscanf(clkStr[len(clkStr)-8:], "%d", &clk.lower[0])
			fmt.Sscanf(clkStr[idx0:len(clkStr)-8], "%d", &clk.upper[0])

			if clk.upper[0] < 0 {
				clk.lower[0] *= -1
			}
		}
	}

	return nil
}

func repairClock(clkArcOrder int, clkOrder *int, clk0 *_ClockFormat, clk *_ClockFormat) {
	if *clkOrder < clkArcOrder {
		*clkOrder++

		for i, j := 0, 1; i < *clkOrder; i, j = i+1, j+1 {
			clk.upper[j] = clk.upper[i] + clk0.upper[i]
			clk.lower[j] = clk.lower[i] + clk0.lower[i]
			clk.upper[j] += clk.lower[j] / 100000000 // to avoid overflow
			clk.lower[j] %= 100000000
		}
	} else {
		for i, j := 0, 1; i < *clkOrder; i, j = i+1, j+1 {
			clk.upper[j] = clk.upper[i] + clk0.upper[j]
			clk.lower[j] = clk.lower[i] + clk0.lower[j]
			clk.upper[j] += clk.lower[j] / 100000000 // to avoid overflow
			clk.lower[j] %= 100000000
		}
	}
}

func printClock(writer *bufio.Writer, upper, lower int64, shift int) error {
	if upper < 0 && lower > 0 {
		upper++
		lower -= 100000000
	} else if upper > 0 && lower < 0 {
		upper--
		lower += 100000000
	}

	var line string

	// add one more digit to handle '-0'(RINEX2) or '-0000'(RINEX3)
	// AT LEAST fractional parts are filled with 0
	if lower < 0 {
		line = fmt.Sprintf("%.*d", shift+1, upper*10-1)
	} else {
		line = fmt.Sprintf("%.*d", shift+1, upper*10+1)
	}

	var n = len(line) - 1 // number of digits excluding the additional digit
	var idx = n - shift
	var bs []byte

	bs = fmt.Appendf(bs, "  .%s", line[idx:n])

	if n > shift {
		idx--
		idx1 := len(bs) - shift - 2
		bs[idx1] = line[idx]

		if n > shift+1 {
			bs[idx1-1] = line[idx-1]

			if n > shift+2 {
				return fmt.Errorf("clock offset out of range")
			}
		}
	}

	writer.Write(bs)

	if lower < 0 {
		fmt.Fprintf(writer, "%8.8d\n", -lower)
	} else {
		fmt.Fprintf(writer, "%8.8d\n", lower)
	}

	return nil
}
