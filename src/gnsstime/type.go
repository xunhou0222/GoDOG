/*
	GNSSTime is a golang package used to represent several time systems that are frequently

encountered in GNSS field. In this package, time is expressed in different format, e.g.,
"Y-M-D H:M:S", "week/sow", and "year/doy/sod", and in different systems such as TAI, TT,
UTC, GPST, GLONASS-UTC, BDT and GST.

	The origins of time systems:
		TAI		1958-01-01 00:00:00 (TAI)		1958-01-01 00:00:00 (TAI)		1958-12-31 23:59:50 (UTC)
		TT		 1977-01-01 00:00:00 (TT)		 1976-12-31 23:59:27.816 (TAI)	1976-12-31 23:59:13.816 (UTC)
		UTC		1972-01-01 00:00:00 (UTC)		1972-01-01 00:00:10 (TAI)		1972-01-01 00:00:00 (UTC)
		GPST	   1980-01-06 00:00:00 (GPST)	   1980-01-06 00:00:19 (TAI)		1980-01-06 00:00:00 (UTC)
		GLO-UTC	1972-01-01 00:00:00 (GLO-UTC)	1972-01-01 03:00:10 (TAI)		1972-01-01 03:00:00 (UTC)
		BDT		2006-01-01 00:00:00 (BDT)		2006-01-01 00:00:33 (TAI)		2006-01-01 00:00:00 (UTC)
		GST		1999-08-21 23:59:47 (GST)		1999-08-22 00:00:19 (TAI)		1999-08-21 23:59:47 (UTC)
	Among the time systems this package concerns, there are some suttle problems, which are

mainly found in UTC. Due to leap seconds, UTC is not a consecutive time system like TAI
and GPST. That means, there would be 86401 or 86399 seconds in some days. In this library,
UTC is handled as below:

	In year/doy/sod format, sod can be 86401 or 86399 seconds, but in week/sow format, sow

must not exceed 604800.

	Since GNSSTime is implemented on the basis of pydate, which can only represent year from

1 to 9999, there is a lowwer/upper limit for each GNSS time system (see "const.go").
*/
package gnsstime

import (
	"fmt"
	"time"
)

// -----------------------------------------------------------------------------------
// 1. The structure "PyDate".

/*
	As indicated by its name, PyDate imitatea the date class in a python

module named datetime.
*/
type PyDate struct {
	year  int
	month int
	day   int
}

// 2. Constructor for "PyDate".

func InitPyDate() PyDate {
	return PyDate{1, 1, 1}
}

func NewPyDate(year, month, day int) (PyDate, error) {
	dim, err := days_in_month(year, month)

	if err != nil {
		return _PyDate0, err
	}

	if day < 1 || day > dim {
		err := fmt.Errorf("day must be in 1...%d, %d", dim, day)
		return _PyDate0, err
	}

	return PyDate{year, month, day}, nil
}

// -----------------------------------------------------------------------------------

// 3. The structure "GNSSTime".

type GNSSTime struct {
	flag   byte

	year   int
	month  int
	day    int
	hour   int
	minute int
	second float64

	doy    int
	sod    float64

	week   int
	dow    int
	sow    float64
}

// 4. Constructors of the structure "GNSSTime".

func InitGNSSTime() GNSSTime {
	return GNSSTime{'G', 1, 1, 1, 0, 0, 0, 1, 0, -103260, 1, 86400}
}

func FromDateTime(flag byte, year, month, day, hour, minute int, second float64) (GNSSTime, error) {
	week, dow, err := Date2Week(flag, year, month, day)

	if err != nil {
		return _GNSSTime0, err
	}

	if hour < 0 || hour > 23 {
		err = fmt.Errorf("hour must be in 0...23, but %d is given", hour)
		return _GNSSTime0, err
	} else if minute < 0 || minute > 59 {
		err = fmt.Errorf("minute must be in 0...59, but %d is given", minute)
		return _GNSSTime0, err
	}

	sec_limit := 60
	d, _ := NewPyDate(year, month, day)

	if flag == 'U' || flag == 'u' || flag == 'R' || flag == 'r' {
		sec_limit += utc_flag(d)
	} else {
		if second < 0 || second >= float64(sec_limit) {
			err = fmt.Errorf("second must be in 0...%d (excluded), but %f is given",
				sec_limit, second)
			return _GNSSTime0, err
		}
	}

	if second < 0 || second >= float64(sec_limit) {
		err = fmt.Errorf("second must be in 0...%d (excluded) for this date, but %f is given",
			sec_limit, second)
		return _GNSSTime0, err
	}

	doy, _ := Date2Doy(year, month, day)
	sod := float64(hour*3600+minute*60) + second
	sow := float64(dow*86400) + sod

	if flag == 'U' || flag == 'u' {
		sow += float64(leap_seconds(d) - 10)
	} else if flag == 'R' || flag == 'r' {
		if hour < 3 && year >= 1972 {
			d.SubEq(1)
		}

		sow += float64(leap_seconds(d) - 10)
	} else if flag == 'E' || flag == 'e' {
		sow += 13
	}

	if sow >= 604800 {
		week += 1
		sow -= 604800
	} else if sow < 0 {
		week -= 1
		sow += 604800
	}

	dow = int(sow / 86400)

	return GNSSTime{flag, year, month, day, hour, minute, second, doy, sod, week, dow, sow}, nil
}

func FromWeekSow(flag byte, week int, sow float64) (GNSSTime, error) {
	dow := int(sow / 86400)
	_, err := sys_week_dow_check(flag, week, dow)

	if err != nil {
		return _GNSSTime0, err
	}

	if sow < 0 || sow >= 604800 {
		err = fmt.Errorf("sow must be in 0...606800 (excluded), but %f is given", sow)
		return _GNSSTime0, err
	}

	year, month, day, _ := Week2Date(flag, week, dow)
	sod := sow - float64(dow*86400)
	d, _ := NewPyDate(year, month, day)

	if flag == 'U' || flag == 'u' {
		sod -= float64(leap_seconds(d) - 10)

		if sod < 0 {
			d.SubEq(1)
			year = d.Year()
			month = d.Month()
			day = d.Day()
			sod += float64(86400 + utc_flag(d))
		}
	} else if flag == 'R' || flag == 'r' {
		if sod < 10800 && year >= 1972 {
			d.SubEq(1)
		}

		sod -= float64(leap_seconds(d) - 10)

		if sod < 0 {
			d.SubEq(1)
			year = d.Year()
			month = d.Month()
			day = d.Day()
			sod += float64(86400 + utc_flag(d))
		}
	} else if flag == 'E' || flag == 'e' {
		if int(sow)%86400 < 13 {
			d.SubEq(1)
			year = d.Year()
			month = d.Month()
			day = d.Day()
		}

		sod -= 13

		if sod < 0 {
			sod += 86400
		}
	}

	hour := int(sod / 3600)

	if hour == 24 {
		hour--
	}

	minute := int((sod - float64(hour*3600)) / 60)

	if minute == 60 {
		minute--
	}

	second := sod - float64(hour*3600+minute*60)
	doy, _ := Date2Doy(year, month, day)

	return GNSSTime{flag, year, month, day, hour, minute, second, doy, sod, week, dow, sow}, nil
}

func FromDoySod(flag byte, year, doy int, sod float64) (GNSSTime, error) {
	err := year_doy_check(year, doy)

	if err != nil {
		return _GNSSTime0, err
	}

	week, dow, _ := Doy2Week(flag, year, doy)
	month, day, _ := Doy2Date(year, doy)

	sod_limit := 86400
	d, _ := NewPyDate(year, month, day)

	if flag == 'U' || flag == 'u' {
		sod_limit += utc_flag(d)
	} else if flag == 'R' || flag == 'r' {
		if sod < 10800 && year >= 1972 {
			d.SubEq(1)
		}

		sod_limit += utc_flag(d)
	}

	if sod < 0 || sod >= float64(sod_limit) {
		err = fmt.Errorf("sod (second of day) must be in 0...%d (excluded) for that day, "+
			"but %f is given", sod_limit, sod)
		return _GNSSTime0, err
	}

	hour := int(sod / 3600)

	if hour == 24 {
		hour--
	}

	minute := int((sod - float64(hour*3600)) / 60)

	if minute == 60 {
		minute--
	}

	second := sod - float64(hour*3600+minute*60)
	sow := sod + float64(dow*86400)

	if flag == 'U' || flag == 'u' {
		sow += float64(leap_seconds(d) - 10)
	} else if flag == 'R' || flag == 'r' {
		if hour < 3 && year >= 1972 {
			d.SubEq(1)
		}

		sow += float64(leap_seconds(d) - 10)
	} else if flag == 'E' || flag == 'e' {
		sow += 13
	}

	if sow >= 604800 {
		sow -= 604800
		week++
	} else if sow < 0 {
		sow += 604800
		week--
	}

	dow = int(sow / 86400)

	return GNSSTime{flag, year, month, day, hour, minute, second, doy, sod, week, dow, sow}, nil
}

func Now() (t_now GNSSTime) {
	t := time.Now().UTC()
	t_now, _ = FromDateTime('U', t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), float64(t.Second()+t.Nanosecond()/1.0e9))

	return t_now
}

func FromStr(str string) (GNSSTime, error) {
	var ep [6]float64
	var flag byte

	num, err := fmt.Sscanf(str, "%c %f %f %f %f %f %f",
		                   &flag, &ep[0], &ep[1], &ep[2], &ep[3], &ep[4], &ep[5])

	if num == 0 {
		return _GNSSTime0, err
	}

	if num == 7 {
		return FromDateTime(flag, int(ep[0]), int(ep[1]), int(ep[2]), int(ep[3]), int(ep[4]), ep[5])
	} else if num == 3 {
		return FromWeekSow(flag, int(ep[0]), ep[1])
	} else if num == 4 {
		return FromDoySod(flag, int(ep[0]), int(ep[1]), ep[2])
	}

	err = fmt.Errorf("invalid format in the string")

	return _GNSSTime0, err
}

// -----------------------------------------------------------------------------------
