package datetime

import (
	"fmt"
	"strings"
)

/***** FUNCTION ********************************/

func isLeapYear(year int32) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

/***********************************************/

// Convert year/month/day to ordinal.
func ymd2ord(year int32, month, day uint8) int32 {
	if month < 1 || month > 12 {
		panic("month must be in 1..12")
	}

	var maxDim uint8

	if isLeapYear(year) && month == 2 {
		maxDim = 29
	} else {
		maxDim = _DAYS_IN_MONTH[month-1]
	}

	if day < 1 || day > maxDim {
		panic(fmt.Sprintf("day must be in 1..%d", maxDim))
	}

	var refYear int32 = _ORD_REF_YEAR
	var n400 int32 = 0
	var dby int32
	var dbm int16

	if year < refYear {
		n400 = (refYear-year)/400 + 1
		refYear -= n400 * 400
	}

	// number of days before January 1st of year, considering 01-Jan-1601 as ordinal 1
	dby = (year-refYear)*365 + (year-refYear)/4 - (year-refYear)/100 + (year-refYear)/400
	// number of days in year preceding first day of month
	dbm = int16(_DAYS_BEFORE_MONTH[month-1])

	if isLeapYear(year) && month > 2 {
		dbm++
	}

	return dby + int32(dbm) + int32(day) - n400*(400*365+97)
}

/***********************************************/

// Convert ordinal to year/month/day.
func ord2ymd(ord int32) (year int32, month, day uint8) {
	var di4y int32 = 4*365 + 1
	var di100y int32 = 25*di4y - 1
	var di400y int32 = 4*di100y + 1

	var n400, n100, n4, n1 int32
	var refYear int32 = _ORD_REF_YEAR

	if ord <= 0 {
		n400 = (1-ord)/(400*365+97) + 1
		refYear -= n400 * 400
		ord += n400 * (400*365 + 97)
	}

	ord--
	n400 = ord / di400y
	ord %= di400y
	n100 = ord / di100y
	ord %= di100y
	n4 = ord / di4y
	ord %= di4y
	n1 = ord / 365
	ord %= 365

	year = 400*n400 + 100*n100 + 4*n4 + n1 + refYear

	if (n1 == 4 || n100 == 4) && ord == 0 {
		year -= 1
		month = 12
		day = 31
	} else {
		month = uint8((ord + 50) >> 5)
		var dbm int16 = int16(_DAYS_BEFORE_MONTH[month-1])

		if isLeapYear(year) && month > 2 {
			dbm++
		}

		if int32(dbm) > ord {
			month -= 1
			dbm -= int16(_DAYS_IN_MONTH[month-1])

			if isLeapYear(year) && month == 2 {
				dbm--
			}
		}

		day = uint8(ord - int32(dbm) + 1)
	}

	return
}

/***********************************************/

func getLeapSec(mjd int32) (value int8, total int16) {
	value = 0
	total = 0

	for _, item := range UTC_LEAP_SEC {
		if mjd+1 == item.Mjd {
			value = item.Value
		}

		if mjd >= item.Mjd {
			total = item.Total
			break
		}
	}

	return
}

/***********************************************/

func fromTAI(t *Time, sys TimeSys) {
	switch sys {
	case TIME_SYS_TT:
		t.SubEq(Seconds2Time(DELTA_TAI_TT))
	case TIME_SYS_UTC:
		t.SubEq(Seconds2Time(DELTA_TAI_UTC))
	case TIME_SYS_GPST:
		t.SubEq(Seconds2Time(DELTA_TAI_GPST))
	case TIME_SYS_GLONASST:
		t.SubEq(Seconds2Time(DELTA_TAI_UTC - DELTA_GLOT_UTC))
	case TIME_SYS_BDT:
		t.SubEq(Seconds2Time(DELTA_TAI_BDT))
	case TIME_SYS_GST:
		t.SubEq(Seconds2Time(DELTA_TAI_GST))
	}

	t.sys = sys
}

/***********************************************/

func toTAI(t *Time) {
	switch t.sys {
	case TIME_SYS_TT:
		t.AddEq(Seconds2Time(DELTA_TAI_TT))
	case TIME_SYS_UTC:
		t.AddEq(Seconds2Time(DELTA_TAI_UTC))
	case TIME_SYS_GPST:
		t.AddEq(Seconds2Time(DELTA_TAI_GPST))
	case TIME_SYS_GLONASST:
		t.AddEq(Seconds2Time(DELTA_TAI_UTC - DELTA_GLOT_UTC))
	case TIME_SYS_BDT:
		t.AddEq(Seconds2Time(DELTA_TAI_BDT))
	case TIME_SYS_GST:
		t.AddEq(Seconds2Time(DELTA_TAI_GST))
	}

	t.sys = TIME_SYS_TAI
}

/***********************************************/

func ParseTimeSys(strSys string) TimeSys {
	strSys = strings.ToUpper(strSys)

	if value, ok := Name2TimeSys[strSys]; ok {
		return value
	} else {
		return TIME_SYS_GPST
	}
}

/***********************************************/
