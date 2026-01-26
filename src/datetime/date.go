package datetime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/***** STRUCT **********************************/

/*
Date representation.
The Date class imitates python's datetime.date class, and represents
a date in the Gregorian calender indefinitely extended in both directions.
Of cource this assumption is not always true, because the Julian calender
was used until 1582. However, the classes implemented here mainly focus on
time representaton and conversion in GNSS field, and such ancient time is
not relevant. So, just let the suttle difference between the two calenders
goes to the hell!
*/
type Date struct {
	ord int32
}

/***** FUNCTION ********************************/

// The default constructor.
func NewDate() Date {
	return Date{1}
}

/***********************************************/

// Date representation from date.
func Date2Date(year int32, month, day uint8) Date {
	ord := ymd2ord(year, month, day)
	return Date{ord}
}

/***********************************************/

func YearDoy2Date(year int32, doy uint16) Date {
	var maxDoy uint16 = 365

	if isLeapYear(year) {
		maxDoy++
	}

	if doy < 1 || doy > maxDoy {
		panic(fmt.Sprintf("day of year must be in 1..%d", maxDoy))
	}

	ord := ymd2ord(year, 1, 1)
	ord += int32(doy) - 1
	return Date{ord}
}

/***********************************************/

func Mjd2Date(mjd int32) Date {
	return Date{mjd - _MJD_ORD1 + 1}
}

/***********************************************/

func (d Date) Gt(other Date) bool {
	return d.ord > other.ord
}

/***********************************************/

func (d Date) Lt(other Date) bool {
	return d.ord < other.ord
}

/***********************************************/

func (d Date) Eq(other Date) bool {
	return d.ord == other.ord
}

/***********************************************/

func (d Date) Ne(other Date) bool {
	return d.ord != other.ord
}

/***********************************************/

func (d Date) Ge(other Date) bool {
	return d.ord >= other.ord
}

/***********************************************/

func (d Date) Le(other Date) bool {
	return d.ord <= other.ord
}

/***********************************************/

func (d Date) Add(days int32) Date {
	return Date{d.ord + days}
}

/***********************************************/

func (d Date) Sub(days int32) Date {
	return Date{d.ord - days}
}

/***********************************************/

func (d *Date) AddEq(days int32) {
	d.ord += days
}

/***********************************************/

func (d *Date) SubEq(days int32) {
	d.ord -= days
}

/***********************************************/

func (d Date) SubDate(other Date) int32 {
	return d.ord - other.ord
}

/***********************************************/

func (d Date) Format(format string) string {
	var (
		re                     = regexp.MustCompile(`\{([\+\- 0]*)(\d*)\.?(\d*)([YymdDO])\}`)
		flag                   string
		typer                  byte
		width, precision       int
		year                   int32
		month, day             uint8
		doy                    uint16
		fmtStr, resStr, result string
	)

	year, month, day = d.Date()
	doy = d.DayOfYear()
	result = format

	for _, matched := range re.FindAllStringSubmatch(format, -1) {
		flag = ""
		width = -1
		precision = -1

		if len(matched[1]) != 0 {
			flag = matched[1]
		}

		if len(matched[2]) != 0 {
			width, _ = strconv.Atoi(matched[2])
		}

		if len(matched[3]) != 0 {
			precision, _ = strconv.Atoi(matched[3])
		}

		typer = matched[4][0]
		fmtStr = "%"

		if flag != "" {
			fmtStr += flag
		}

		if width >= 0 {
			fmtStr += fmt.Sprintf("%d", width)
		}

		if precision >= 0 {
			fmtStr += fmt.Sprintf(".%d", precision)
		}

		switch typer {
		case 'Y': // 2-digit year
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, year%100)
		case 'y': // 4-digit year
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "04d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, year)
		case 'm': // month
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, month)
		case 'd': // day
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, day)
		case 'D':
			resStr = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		case 'O': // day of year
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "03d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, doy)
		default:
			panic(fmt.Sprintf("unknown formater '%s'", flag))
		}

		result = strings.ReplaceAll(result, matched[0], resStr)
	}

	return result
}

/***********************************************/

func (d Date) Ordinal() int32 {
	return d.ord
}

/***********************************************/

func (d Date) Date() (year int32, month, day uint8) {
	year, month, day = ord2ymd(d.ord)
	return
}

/***********************************************/

func (d Date) Year() (year int32) {
	year, _, _ = ord2ymd(d.ord)
	return
}

/***********************************************/

func (d Date) Month() (month uint8) {
	_, month, _ = ord2ymd(d.ord)
	return
}

/***********************************************/

func (d Date) Day() (day uint8) {
	_, _, day = ord2ymd(d.ord)
	return
}

/***********************************************/

func (d Date) YearDoy() (year int32, doy uint16) {
	var month, day uint8
	year, month, day = ord2ymd(d.ord)
	doy = _DAYS_BEFORE_MONTH[month-1]

	if isLeapYear(year) && month > 2 {
		doy++
	}

	doy += uint16(day)
	return
}

/***********************************************/

func (d Date) DayOfYear() (doy uint16) {
	var year int32
	var month, day uint8
	year, month, day = ord2ymd(d.ord)
	doy = _DAYS_BEFORE_MONTH[month-1]

	if isLeapYear(year) && month > 2 {
		doy++
	}

	doy += uint16(day)
	return
}

/***********************************************/

func (d Date) Mjd() int32 {
	return d.ord - 1 + _MJD_ORD1
}

/***********************************************/
