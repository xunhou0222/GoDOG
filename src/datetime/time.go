package datetime

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
Time representation.
Like the Date class, the Time class measures time using ordinal. One slight
difference is that the ordinal consists of an integer part and a decimal
part. The ordinal is the continuous and uniform count of days, considering
01-Jan-1601 as day 1. One ordinal is strictly equal to 86400 seconds.
*/
type Time struct {
	sys    TimeSys // time system
	ordInt int32   // integer part of ordinal
	ordDec float64 // decimal part of ordinal
}

/***** FUNCTION ********************************/

// The default constructor.
func NewTime() Time {
	return Time{TIME_SYS_NONE, 1, 0}
}

/***********************************************/

func DateTime2Time(sys TimeSys, year int32, month, day, hour, minute uint8, second float64) Time {
	if sys == TIME_SYS_NONE {
		panic(fmt.Sprintf("time system cannot be '%s'", TimeSys2Name[TIME_SYS_NONE]))
	}

	if hour > 23 {
		panic("hour must be in 0..23")
	}

	if minute > 59 {
		panic("minute must be in 0..59")
	}

	ordInt := ymd2ord(year, month, day)
	mjd := ordInt - 1 + _MJD_ORD1
	var leapVal int8 = 0
	var leapTot int16 = 0

	switch sys {
	case TIME_SYS_UTC:
		leapVal, leapTot = getLeapSec(mjd)

		if hour != 23 || minute != 59 {
			leapVal = 0
		}
	case TIME_SYS_GLONASST:
		if hour >= 3 {
			leapVal, leapTot = getLeapSec(mjd)
		} else {
			leapVal, leapTot = getLeapSec(mjd - 1)
		}

		if hour != 2 || minute != 59 {
			leapVal = 0
		}
	}

	var maxSec int8 = int8(MINUTE2SECOND) + leapVal

	if second < 0 || second >= float64(maxSec) {
		panic(fmt.Sprintf("second must be in [0, %d)", maxSec))
	}

	var ordDec float64 = float64(hour)*HOUR2DAY + float64(minute)*MINUTE2DAY + (second+float64(leapTot))*SECOND2DAY
	return Time{sys, ordInt, ordDec}
}

/***********************************************/

func YearDoySod2Time(sys TimeSys, year int32, doy uint16, sod float64) Time {
	var maxDoy uint16 = 365

	if isLeapYear(year) {
		maxDoy++
	}

	if doy < 1 || doy > maxDoy {
		panic(fmt.Sprintf("day of year must be in 1..%d", maxDoy))
	}

	ordInt := ymd2ord(year, 1, 1)
	ordInt += int32(doy) - 1
	mjd := ordInt - 1 + _MJD_ORD1
	var leapVal int8 = 0
	var leapTot int16 = 0

	switch sys {
	case TIME_SYS_UTC:
		leapVal, leapTot = getLeapSec(mjd)
	case TIME_SYS_GLONASST:
		leapVal, leapTot = getLeapSec(mjd - 1)
	}

	maxSod := int32(DAY2SECOND) + int32(leapVal)

	if sod < 0 || sod >= float64(maxSod) {
		panic(fmt.Sprintf("sod must be in [0, %d)", maxSod))
	}

	return Time{sys, ordInt, (sod + float64(leapTot)) * SECOND2DAY}
}

/***********************************************/

func WeekSow2Time(sys TimeSys, week int32, sow float64) Time {
	dt := Time{TIME_SYS_NONE, week * int32(WEEK2DAY), sow * SECOND2DAY}

	switch sys {
	case TIME_SYS_NONE:
		return dt
	case TIME_SYS_GPST:
		return TIME_GPST0.Add(dt)
	case TIME_SYS_BDT:
		return TIME_BDT0.Add(dt)
	case TIME_SYS_GST:
		return TIME_GST0.Add(dt)
	default:
		panic(fmt.Sprintf("week/sow are not defined for '%s'", TimeSys2Name[sys]))
	}
}

/***********************************************/

func Now2Time(sys TimeSys) Time {
	tNow := time.Now().UTC()
	year := int32(tNow.Year())
	month := uint8(tNow.Month())
	day := uint8(tNow.Day())
	hour := uint8(tNow.Hour())
	minute := uint8(tNow.Minute())
	second := float64(tNow.Second())
	second += float64(tNow.Nanosecond()) * 1.0e-9
	return DateTime2Time(sys, year, month, day, hour, minute, second)
}

/***********************************************/

func Mjd2Time(sys TimeSys, mjd float64) Time {
	var dec float64

	if mjd < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	mjdInt := int32(math.Floor(mjd) + dec) // to prevent trunction
	ordInt := mjdInt - _MJD_ORD1 + 1
	var leapVal int8 = 0
	var leapTot int16 = 0

	switch sys {
	case TIME_SYS_UTC:
		leapVal, leapTot = getLeapSec(mjdInt)
	case TIME_SYS_GLONASST:
		leapVal, leapTot = getLeapSec(mjdInt - 1)
	}

	maxSod := int32(DAY2SECOND) + int32(leapVal)
	ordDec := ((mjd-float64(mjdInt))*float64(maxSod) + float64(leapTot)) * SECOND2DAY
	return Time{sys, ordInt, ordDec}
}

/***********************************************/

func Ord2Time(sys TimeSys, ordInt int32, ordDec float64) Time {
	return Time{sys, ordInt, ordDec}
}

/***********************************************/

func Seconds2Time(seconds float64) Time {
	var dec float64

	if seconds < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	secTmp := seconds * SECOND2DAY
	ordInt := int32(math.Floor(secTmp) + dec) // to prevent trunction
	return Time{TIME_SYS_NONE, ordInt, secTmp - float64(ordInt)}
}

/***********************************************/

func Str2Time(str string) Time {
	subs := strings.Fields(str)
	var sys TimeSys
	var err error
	var t Time

	if len(subs) == 0 {
		panic("invalid time format")
	}

	sys, ok := Name2TimeSys[strings.ToUpper(subs[0])]

	if !ok {
		panic("invalid time system")
	}

	switch len(subs) {
	case 7:
		var year int64
		var month, day, hour, minute uint64
		var second float64

		year, err = strconv.ParseInt(subs[1], 10, 32)

		if err != nil {
			panic("invalid year")
		}

		month, err = strconv.ParseUint(subs[2], 10, 8)

		if err != nil {
			panic("invalid month")
		}

		day, err = strconv.ParseUint(subs[3], 10, 8)

		if err != nil {
			panic("invalid day")
		}

		hour, err = strconv.ParseUint(subs[4], 10, 8)

		if err != nil {
			panic("invalid hour")
		}

		minute, err = strconv.ParseUint(subs[5], 10, 8)

		if err != nil {
			panic("invalid minute")
		}

		second, err = strconv.ParseFloat(subs[6], 64)

		if err != nil {
			panic("invalid second")
		}

		t = DateTime2Time(sys, int32(year), uint8(month), uint8(day), uint8(hour), uint8(minute), second)
	case 4:
		var year int64
		var doy uint64
		var sod float64

		year, err = strconv.ParseInt(subs[1], 10, 32)

		if err != nil {
			panic("invalid year")
		}

		doy, err = strconv.ParseUint(subs[2], 10, 32)

		if err != nil {
			panic("invalid day of year")
		}

		sod, err = strconv.ParseFloat(subs[3], 64)

		if err != nil {
			panic("invalid second of day")
		}

		t = YearDoySod2Time(sys, int32(year), uint16(doy), sod)
	case 3:
		var week int64
		var sow float64

		week, err = strconv.ParseInt(subs[1], 10, 32)

		if err != nil {
			panic("invalid week")
		}

		sow, err = strconv.ParseFloat(subs[2], 64)

		if err != nil {
			panic("invalid second of week")
		}

		t = WeekSow2Time(sys, int32(week), sow)
	default:
		panic("invalid time format")
	}

	return t
}

/***********************************************/

func (t *Time) ConvertSelf(sys TimeSys) {
	if sys == t.sys {
		return
	}

	if t.sys == TIME_SYS_TAI {
		fromTAI(t, sys)
	} else if sys == TIME_SYS_TAI {
		toTAI(t)
	} else {
		toTAI(t)
		fromTAI(t, sys)
	}
}

/***********************************************/

func (t Time) ConvertNew(sys TimeSys) Time {
	if sys == t.sys {
		return t
	}

	if t.sys == TIME_SYS_TAI {
		fromTAI(&t, sys)
	} else if sys == TIME_SYS_TAI {
		toTAI(&t)
	} else {
		toTAI(&t)
		fromTAI(&t, sys)
	}

	return t
}

/***********************************************/

func Positive(t Time) Time {
	return t
}

/***********************************************/

func Negative(t Time) Time {
	return Time{t.sys, -t.ordInt, -t.ordDec}
}

/***********************************************/

func (t Time) Add(other Time) Time {
	t.AddEq(other)
	return t
}

/***********************************************/

func (t Time) Sub(other Time) Time {
	if other.sys == TIME_SYS_NONE {
		t.SubEq(other)
		return t
	} else {
		if t.sys == TIME_SYS_NONE {
			panic(fmt.Sprintf("when time system of the second operand is not '%s', "+
				"time system of the first operand cannot be '%s'",
				TimeSys2Name[TIME_SYS_NONE],
				TimeSys2Name[TIME_SYS_NONE]))
		}

		t1 := t.ConvertNew(TIME_SYS_TAI)
		t2 := other.ConvertNew(TIME_SYS_TAI)
		t1.ConvertSelf(TIME_SYS_NONE)
		t2.ConvertSelf(TIME_SYS_NONE)
		t1.SubEq(t2)
		return t1
	}
}

/***********************************************/

func (t Time) Mul(c float64) Time {
	t.MulEq(c)
	return t
}

/***********************************************/

func (t Time) Div(c float64) Time {
	t.DivEq(c)
	return t
}

/***********************************************/

func (t *Time) AddEq(other Time) {
	if other.sys != TIME_SYS_NONE {
		panic(fmt.Sprintf("time system of the second operand is not '%s'", TimeSys2Name[TIME_SYS_NONE]))
	}

	t.ordDec += other.ordDec
	t.ordInt += other.ordInt
	var dec float64

	if t.ordDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	ordInt := int32(math.Floor(t.ordDec) + dec)
	t.ordDec -= float64(ordInt)
	t.ordInt += ordInt
}

/***********************************************/

func (t *Time) SubEq(other Time) {
	if other.sys != TIME_SYS_NONE {
		panic(fmt.Sprintf("time system of the second operand is not '%s'", TimeSys2Name[TIME_SYS_NONE]))
	}

	t.ordDec -= other.ordDec
	t.ordInt -= other.ordInt
	var dec float64

	if t.ordDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	ordInt := int32(math.Floor(t.ordDec) + dec)
	t.ordDec -= float64(ordInt)
	t.ordInt += ordInt
}

/***********************************************/

func (t *Time) MulEq(c float64) {
	if t.sys != TIME_SYS_NONE {
		panic(fmt.Sprintf("time system is not '%s'", TimeSys2Name[TIME_SYS_NONE]))
	}

	tmpDec := float64(t.ordInt) * c
	var dec float64

	if tmpDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	t.ordInt = int32(math.Floor(tmpDec) + dec)
	tmpDec -= float64(t.ordInt)

	t.ordDec *= c
	t.ordDec += tmpDec

	if t.ordDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	tmpInt := int32(math.Floor(t.ordDec) + dec)
	t.ordDec -= float64(tmpInt)
	t.ordInt += tmpInt
}

/***********************************************/

func (t *Time) DivEq(c float64) {
	if t.sys != TIME_SYS_NONE {
		panic(fmt.Sprintf("time system is not '%s'", TimeSys2Name[TIME_SYS_NONE]))
	}

	tmpDec := float64(t.ordInt) / c
	var dec float64

	if tmpDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	t.ordInt = int32(math.Floor(tmpDec) + dec)
	tmpDec -= float64(t.ordInt)

	t.ordDec /= c
	t.ordDec += tmpDec

	if t.ordDec < 0 {
		dec = -0.1
	} else {
		dec = 0.1
	}

	tmpInt := int32(math.Floor(t.ordDec) + dec)
	t.ordDec -= float64(tmpInt)
	t.ordInt += tmpInt
}

/***********************************************/

func (t Time) Gt(other Time) bool {
	if t.sys == other.sys {
		return float64(t.ordInt-other.ordInt)+(t.ordDec-other.ordDec) > TIME_EPSILON/86400.0
	} else if t.sys == TIME_SYS_NONE || other.sys == TIME_SYS_NONE {
		panic(fmt.Sprintf("time of time system '%s' cannot be compared with time of other time systems",
			TimeSys2Name[TIME_SYS_NONE]))
	} else {
		other.ConvertSelf(t.sys)
		return float64(t.ordInt-other.ordInt)+(t.ordDec-other.ordDec) > TIME_EPSILON/86400.0
	}
}

/***********************************************/

func (t Time) Lt(other Time) bool {
	if t.sys == other.sys {
		return float64(t.ordInt-other.ordInt)+(t.ordDec-other.ordDec) < -TIME_EPSILON/86400.0
	} else if t.sys == TIME_SYS_NONE || other.sys == TIME_SYS_NONE {
		panic(fmt.Sprintf("time of time system '%s' cannot be compared with time of other time systems",
			TimeSys2Name[TIME_SYS_NONE]))
	} else {
		other.ConvertSelf(t.sys)
		return float64(t.ordInt-other.ordInt)+(t.ordDec-other.ordDec) < -TIME_EPSILON/86400.0
	}
}

/***********************************************/

func (t Time) Eq(other Time) bool {
	return !t.Gt(other) && !t.Lt(other)
}

/***********************************************/

func (t Time) Ne(other Time) bool {
	return !t.Eq(other)
}

/***********************************************/

func (t Time) Ge(other Time) bool {
	return !t.Lt(other)
}

/***********************************************/

func (t Time) Le(other Time) bool {
	return !t.Gt(other)
}

/***********************************************/

func (t Time) Format(format string) string {
	precision := 0
	re := regexp.MustCompile(`\{[\+\- 0]*\d*\.(\d+)[osS]\}`)

	for _, matched := range re.FindAllStringSubmatch(format, -1) {
		precTmp, _ := strconv.Atoi(matched[1])

		if precTmp > precision {
			precision = precTmp
		}
	}

	var (
		year                     int32
		doy                      uint16
		month, day, hour, minute uint8
		sod, second              float64
	)

	year, doy, sod = t.YearDoySod()
	sod = math.Round(sod*math.Pow10(precision)) / math.Pow10(precision)
	t = YearDoySod2Time(t.sys, year, doy, sod)
	year, month, day, hour, minute, second = t.DateTime()

	var (
		flag                   string
		typer                  byte
		width                  int
		fmtStr, resStr, result string
	)

	re = regexp.MustCompile(`\{([\+\- 0]*)(\d*)\.?(\d*)([YymdHhMSDTOoWws])\}`)
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
		case 'H': // hour
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, hour)
		case 'h': // hour represented with letters, e.g. 'a' or 'A' for 0 hour, 'b' or 'B' or 1 hour
			hourByte := 'a' + hour
			fmtStr = strings.ReplaceAll(fmtStr, "+", "")
			fmtStr = strings.ReplaceAll(fmtStr, "-", "")
			fmtStr += "c"

			if strings.IndexByte(flag, '+') >= 0 { // uppercase
				hourByte = 'A' + hour
			}
			resStr = fmt.Sprintf(fmtStr, hourByte)
		case 'M': // minute
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, minute)
		case 'S': // second
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "02.0f"
			} else {
				fmtStr += "f"
			}

			resStr = fmt.Sprintf(fmtStr, second)
		case 'D':
			resStr = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		case 'T':
			resStr = fmt.Sprintf("%02d:%02d:%02.0f", hour, minute, second)
		case 'O': // day of year
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "03d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, doy)
		case 'o': // second of day
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "05.0f"
			} else {
				fmtStr += "f"
			}

			resStr = fmt.Sprintf(fmtStr, sod)
		case 'W': // week
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "04d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, t.Week())
		case 'w': // day of week
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "1d"
			} else {
				fmtStr += "d"
			}

			resStr = fmt.Sprintf(fmtStr, t.DayOfWeek())
		case 's': // second of week
			if flag == "" && width < 0 && precision < 0 { // default format
				fmtStr += "06.0f"
			} else {
				fmtStr += "f"
			}

			resStr = fmt.Sprintf(fmtStr, t.SecondOfWeek())
		default:
			panic(fmt.Sprintf("unknown formater '%s'", flag))
		}

		result = strings.ReplaceAll(result, matched[0], resStr)
	}

	return result
}

/***********************************************/

func (t Time) Sys() TimeSys {
	return t.sys
}

/***********************************************/

func (t Time) Ordinal() (ordInt int32, ordDec float64) {
	ordInt = t.ordInt
	ordDec = t.ordDec

	if math.Abs(ordDec*float64(DAY2SECOND)) < TIME_EPSILON {
		ordDec = 0.0
	}

	return
}

/***********************************************/

func (t Time) OrdInt() int32 {
	return t.ordInt
}

/***********************************************/

func (t Time) OrdDec() float64 {
	ordDec := t.ordDec

	if math.Abs(ordDec*float64(DAY2SECOND)) < TIME_EPSILON {
		ordDec = 0.0
	}

	return ordDec
}

/***********************************************/

func (t Time) OrdTotal() float64 {
	ordDec := t.ordDec

	if math.Abs(ordDec*float64(DAY2SECOND)) < TIME_EPSILON {
		ordDec = 0.0
	}

	return float64(t.ordInt) + ordDec
}

/***********************************************/

func (t Time) DateTime() (year int32, month, day, hour, minute uint8, second float64) {
	mjd := t.ordInt - 1 + _MJD_ORD1
	var leapVal int8 = 0
	var leapTot int16 = 0
	ordInt := t.ordInt
	ordDec := t.ordDec
	var sod float64
	var maxSod int32

	for {
		switch t.sys {
		case TIME_SYS_UTC:
			leapVal, leapTot = getLeapSec(mjd)
		case TIME_SYS_GLONASST:
			leapVal, leapTot = getLeapSec(mjd - 1)
		}

		maxSod = int32(DAY2SECOND) + int32(leapVal)
		sod = ordDec*float64(DAY2SECOND) - float64(leapTot)

		if sod < -TIME_EPSILON {
			mjd--
			ordInt--
			ordDec += 1
		} else if sod > float64(maxSod)-TIME_EPSILON {
			mjd++
			ordInt++
			ordDec -= 1
		} else {
			break
		}
	}

	year, month, day = ord2ymd(ordInt)

	hour = 0
	var i uint8

	for i = 0; i < DAY2HOUR; i++ {
		leapTot = 0

		if t.sys == TIME_SYS_UTC && i == 23 {
			leapTot = int16(leapVal)
		} else if t.sys == TIME_SYS_GLONASST && i == 2 {
			leapTot = int16(leapVal)
		}

		hour = i

		if sod-float64(HOUR2SECOND)-float64(leapTot) < -TIME_EPSILON {
			break
		}

		sod -= float64(HOUR2SECOND) + float64(leapTot)
	}

	minute = 0

	for i = 0; i < HOUR2MINUTE; i++ {
		leapTot = 0

		if t.sys == TIME_SYS_UTC && hour == 23 && i == 59 {
			leapTot = int16(leapVal)
		} else if t.sys == TIME_SYS_GLONASST && hour == 2 && i == 59 {
			leapTot = int16(leapVal)
		}

		minute = i

		if sod-float64(MINUTE2SECOND)-float64(leapTot) < -TIME_EPSILON {
			break
		}

		sod -= float64(MINUTE2SECOND) + float64(leapTot)
	}

	second = sod

	if math.Abs(second) < TIME_EPSILON {
		second = 0.0
	}

	return
}

/***********************************************/

func (t Time) Date() (year int32, month, day uint8) {
	year, month, day, _, _, _ = t.DateTime()
	return
}

/***********************************************/

func (t Time) Time() (hour, minute uint8, second float64) {
	_, _, _, hour, minute, second = t.DateTime()
	return
}

/***********************************************/

func (t Time) Year() (year int32) {
	year, _, _, _, _, _ = t.DateTime()
	return year
}

/***********************************************/

func (t Time) Month() (month uint8) {
	_, month, _, _, _, _ = t.DateTime()
	return
}

/***********************************************/

func (t Time) Day() (day uint8) {
	_, _, day, _, _, _ = t.DateTime()
	return
}

/***********************************************/

func (t Time) Hour() (hour uint8) {
	_, _, _, hour, _, _ = t.DateTime()
	return
}

/***********************************************/

func (t Time) Minute() (minute uint8) {
	_, _, _, _, minute, _ = t.DateTime()
	return
}

/***********************************************/

func (t Time) Second() (second float64) {
	_, _, _, _, _, second = t.DateTime()
	return
}

/***********************************************/

func (t Time) YearDoySod() (year int32, doy uint16, sod float64) {
	mjd := t.ordInt - 1 + _MJD_ORD1
	var leapVal int8 = 0
	var leapTot int16 = 0
	ordInt := t.ordInt
	ordDec := t.ordDec
	var maxSod int32

	for {
		switch t.sys {
		case TIME_SYS_UTC:
			leapVal, leapTot = getLeapSec(mjd)
		case TIME_SYS_GLONASST:
			leapVal, leapTot = getLeapSec(mjd - 1)
		}

		maxSod = int32(DAY2SECOND) + int32(leapVal)
		sod = ordDec*float64(DAY2SECOND) - float64(leapTot)

		if sod < -TIME_EPSILON {
			mjd--
			ordInt--
			ordDec += 1
		} else if sod > float64(maxSod)-TIME_EPSILON {
			mjd++
			ordInt++
			ordDec -= 1
		} else {
			break
		}
	}

	year, _, _ = ord2ymd(ordInt)
	doy = uint16(ordInt - ymd2ord(year, 1, 1) + 1)

	if math.Abs(sod) < TIME_EPSILON {
		sod = 0.0
	}

	return
}

/***********************************************/

func (t Time) DayOfYear() (doy uint16) {
	_, doy, _ = t.YearDoySod()
	return
}

/***********************************************/

func (t Time) SecondOfDay() (sod float64) {
	_, _, sod = t.YearDoySod()
	return
}

/***********************************************/

func (t Time) WeekSow() (week int32, sow float64) {
	var dt Time

	switch t.sys {
	case TIME_SYS_NONE:
		dt = t
	case TIME_SYS_GPST:
		dt = t.Sub(TIME_GPST0)
	case TIME_SYS_BDT:
		dt = t.Sub(TIME_BDT0)
	case TIME_SYS_GST:
		dt = t.Sub(TIME_GST0)
	default:
		panic(fmt.Sprintf("week/sow are not defined for '%s'", TimeSys2Name[t.sys]))
	}

	week = dt.ordInt / int32(WEEK2DAY)
	sow = (float64(dt.ordInt%int32(WEEK2DAY)) + dt.ordDec) * float64(DAY2SECOND)

	if math.Abs(sow) < TIME_EPSILON {
		sow = 0.0
	}

	return
}

/***********************************************/

func (t Time) Week() (week int32) {
	week, _ = t.WeekSow()
	return
}

/***********************************************/

func (t Time) DayOfWeek() (dow uint8) {
	_, sow := t.WeekSow()
	dow = uint8(sow * float64(SECOND2DAY))
	return
}

/***********************************************/

func (t Time) SecondOfWeek() (sow float64) {
	_, sow = t.WeekSow()
	return
}

/***********************************************/

func (t Time) Mjd() (mjdInt int32, mjdDec float64) {
	var leapVal int8 = 0
	var leapTot int16 = 0
	ordDec := t.ordDec
	var maxSod int32
	var sod float64

	mjdInt = t.ordInt - 1 + _MJD_ORD1

	for {
		switch t.sys {
		case TIME_SYS_UTC:
			leapVal, leapTot = getLeapSec(mjdInt)
		case TIME_SYS_GLONASST:
			leapVal, leapTot = getLeapSec(mjdInt - 1)
		}

		sod = ordDec*float64(DAY2SECOND) - float64(leapTot)
		maxSod = int32(DAY2SECOND) + int32(leapVal)

		if sod < -TIME_EPSILON {
			mjdInt--
			ordDec += 1
		} else if sod > float64(maxSod)-TIME_EPSILON {
			mjdInt++
			ordDec -= 1
		} else {
			break
		}
	}

	if math.Abs(sod) < TIME_EPSILON {
		sod = 0.0
	}

	mjdDec = sod / float64(maxSod)
	return
}

/***********************************************/

func (t Time) MjdTotal() (mjd float64) {
	mjdInt, mjdDec := t.Mjd()
	mjd = float64(mjdInt) + mjdDec
	return
}

/***********************************************/
