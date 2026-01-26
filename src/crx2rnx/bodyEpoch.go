package crx2rnx

import (
	"fmt"
	"io"
	"slices"
)

func comment(writer io.Writer, rnxVer int, line string) {
	if rnxVer == 2 {
		fmt.Fprintf(writer, "%29d%3d\n%-60sCOMMENT\n", 4, 1, line)
	} else {
		fmt.Fprintf(writer, ">%31d%3d\n%-60sCOMMENT\n", 4, 1, line)
	}
}

func repair(lineSb []byte, lineSbAll *[]byte) {
	if len(*lineSbAll) < len(lineSb) {
		*lineSbAll = append(*lineSbAll, lineSb[len(*lineSbAll):]...)
	}

	for i, c := range lineSb {
		if c == ' ' {
			continue
		} else if c == '&' {
			(*lineSbAll)[i] = ' '
		} else {
			(*lineSbAll)[i] = c
		}
	}
}

func setSatInfo(RnxVer int, TypeNumGNSS map[byte]int, lineSbAll []byte, satNum int,
	satList0 []_TypePRN, satList *[]_TypePRN, satInfoList *[]_SatInfo) error {
	var prn _TypePRN

	if len(*satInfoList) < satNum {
		if cap(*satInfoList) >= satNum {
			*satInfoList = (*satInfoList)[:satNum]
		} else {
			*satInfoList = make([]_SatInfo, satNum)
		}
	}

	if len(*satList) < satNum {
		if cap(*satList) >= satNum {
			*satList = (*satList)[:satNum]
		} else {
			*satList = make([]_TypePRN, satNum)
		}
	}

	for i := 0; i < satNum; i++ {
		if 3*i+2 >= len(lineSbAll) {
			return fmt.Errorf("the satellite list seems to be truncated in the middle")
		}

		prn[0] = lineSbAll[3*i]
		prn[1] = lineSbAll[3*i+1]
		prn[2] = lineSbAll[3*i+2]

		if RnxVer == 2 {
			(*satInfoList)[i].TypeNum = TypeNumGNSS[0]
		} else {
			if _, ok := TypeNumGNSS[prn[0]]; !ok {
				return fmt.Errorf("a GNSS system not defined in the header was found")
			}

			(*satInfoList)[i].TypeNum = TypeNumGNSS[prn[0]]
		}

		(*satInfoList)[i].OldIdx = slices.Index(satList0, prn)
		(*satList)[i] = prn
	}

	return nil
}
