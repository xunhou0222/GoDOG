package gnsstime

import (
	"bytes"
	"fmt"
	"strings"
)

func ParseTimeSys(TimeSys string) byte {
	TimeSys = strings.ToUpper(TimeSys)

	switch TimeSys {
	case "GPST":
		return 'G'
	case "GLONASS-UTC":
		return 'R'
	case "BDT":
		return 'C'
	case "GST":
		return 'E'
	case "UTC":
		return 'U'
	case "TAI":
		return 'A'
	case "TT":
		return 'T'
	default:
		return 'G'
	}
}

/*
Determine whether the year is a leap year or not.
*/
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

/*
Get the number of days in the given month in a specific year.
*/
func days_in_month(year, month int) (int, error) {
	// Validation check.
	if year < MINYEAR || year > MAXYEAR {
		err := fmt.Errorf("year must be in %d...%d, but %d is given", MINYEAR, MAXYEAR, year)
		return 0, err
	} else if month < 1 || month > 12 {
		err := fmt.Errorf("month must be in 1...12, but %d is given", month)
		return 0, err
	}

	if month == 2 && isLeapYear(year) {
		return 29, nil
	}

	return _DAYS_IN_MONTH[month - 1], nil
}

/*
Number of days preceding the first day of a given month of a specific year.
*/
func days_before_month(year, month int) (int, error) {
	if year < MINYEAR || year > MAXYEAR {
		err := fmt.Errorf("year must be in %d...%d, but %d is given", MINYEAR, MAXYEAR, year)
		return 0, err
	} else if month < 1 || month > 12 {
		err := fmt.Errorf("month must be in 1...12, but %d is given", month)
		return 0, err
	}

	if month > 2 && isLeapYear(year) {
		return _DAYS_BEFORE_MONTH[month - 1] + 1, nil
	} else {
		return _DAYS_BEFORE_MONTH[month - 1], nil
	}
}

/*
The number of days before January 1st of a given year.
*/
func days_before_year(year int) (int, error) {
	if year < MINYEAR || year > MAXYEAR {
		err := fmt.Errorf("year must be in %d...%d, but %d is given", MINYEAR, MAXYEAR, year)
		return 0, err
	}

	return (year - 1)*365 + (year - 1)/4 - (year - 1)/100 + (year-1)/400, nil
}

/*
year, month, day -> ordinal, considering 01-Jan-0001 as ordinal 1.
*/
func ymd2ord(year, month, day int) (int, error) {
	dim, err := days_in_month(year, month)

	if err != nil {
		return 0, err
	}

	if day < 1 || day > dim {
		err = fmt.Errorf("day must be in 1...%d, but %d is given", dim, day)
		return 0, err
	}

	d_b_y, _ := days_before_year(year)
	d_b_m, _ := days_before_month(year, month)

	return d_b_y + d_b_m + day, nil
}

/*
ordinal -> (year, month, day), considering 01-Jan-0001 as ordinal 1.
*/
func ord2ymd(ord int) (int, int, int, error) {
	if ord < 1 || ord > MAXORDINAL {
		err := fmt.Errorf("ordinal must be in 1...%d, but %d is given", MAXORDINAL, ord)
		return 0, 0, 0, err
	}

	DI4Y := 4*365 + 1
	DI100Y := 25*DI4Y - 1
	DI400Y := 4*DI100Y + 1
	n := ord - 1
	n400 := n / DI400Y
	n = n % DI400Y
	n100 := n / DI100Y
	n = n % DI100Y
	n4 := n / DI4Y
	n = n % DI4Y
	n1 := n / 365
	n = n % 365

	var year, month, day int

	year = 400*n400 + 100*n100 + 4*n4 + n1 + 1

	if (n1 == 4 || n100 == 4) && n == 0 {
		year -= 1
		month = 12
		day = 31
		return year, month, day, nil
	}

	month = (n + 50) >> 5
	preceding, _ := days_before_month(year, month)

	if preceding > n {
		month -= 1

		if month == 2 && isLeapYear(year) {
			preceding -= _DAYS_IN_MONTH[month-1] + 1
		} else {
			preceding -= _DAYS_IN_MONTH[month-1]
		}
	}

	day = n - preceding + 1

	return year, month, day, nil
}

func sys_week_dow_check(flag byte, week, dow int) (PyDate, error) {
	switch flag {
	case 'A', 'a':
		if week < MINTAIWEEK || week > MAXTAIWEEK {
			err := fmt.Errorf("TAI week must be in %d...%d, but %d is given", MINTAIWEEK, MAXTAIWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("TAI dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINTAIWEEK && dow < MINTAIWEEK_MINDOW {
			err := fmt.Errorf("when TAI week is equal to %d, dow must not be less than %d, but %d is given",
				              MINTAIWEEK, MINTAIWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXTAIWEEK && dow > MAXTAIWEEK_MAXDOW {
			err := fmt.Errorf("when TAI week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXTAIWEEK, MAXTAIWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1958, 1, 1)
	case 'T', 't':
		if week < MINTTWEEK || week > MAXTTWEEK {
			err := fmt.Errorf("TT week must be in %d...%d, but %d is given", MINTTWEEK, MAXTTWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("TT dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINTTWEEK && dow < MINTTWEEK_MINDOW {
			err := fmt.Errorf("when TT week is equal to %d, dow must not be less than %d, but %d is given",
				              MINTAIWEEK, MINTAIWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXTTWEEK && dow > MAXTTWEEK_MAXDOW {
			err := fmt.Errorf("when TT week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXTTWEEK, MAXTTWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1977, 1, 1)
	case 'U', 'u':
		if week < MINUTCWEEK || week > MAXUTCWEEK {
			err := fmt.Errorf("UTC week must be in %d...%d, but %d is given", MINUTCWEEK, MAXUTCWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("UTC dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINUTCWEEK && dow < MINUTCWEEK_MINDOW {
			err := fmt.Errorf("when UTC week is equal to %d, dow must not be less than %d, but %d is given",
				              MINUTCWEEK, MINUTCWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXUTCWEEK && dow > MAXUTCWEEK_MAXDOW {
			err := fmt.Errorf("when UTC week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXUTCWEEK, MAXUTCWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1972, 1, 1)
	case 'G', 'g':
		if week < MINGPSWEEK || week > MAXGPSWEEK {
			err := fmt.Errorf("GPS week must be in %d...%d, but %d is given", MINGPSWEEK, MAXGPSWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("GPS dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINGPSWEEK && dow < MINGPSWEEK_MINDOW {
			err := fmt.Errorf("when GPS week is equal to %d, dow must not be less than %d, but %d is given",
				              MINGPSWEEK, MINGPSWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXGPSWEEK && dow > MAXGPSWEEK_MAXDOW {
			err := fmt.Errorf("when GPS week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXGPSWEEK, MAXGPSWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1980, 1, 6)
	case 'R', 'r':
		if week < MINGLOWEEK || week > MAXGLOWEEK {
			err := fmt.Errorf("GLO week must be in %d...%d, but %d is given", MINGLOWEEK, MAXGLOWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("GLO dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINGLOWEEK && dow < MINGLOWEEK_MINDOW {
			err := fmt.Errorf("when GLO week is equal to %d, dow must not be less than %d, but %d is given",
				              MINGLOWEEK, MINGLOWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXGLOWEEK && dow > MAXGLOWEEK_MAXDOW {
			err := fmt.Errorf("when GLO week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXGLOWEEK, MAXGLOWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1972, 1, 1)
	case 'C', 'c':
		if week < MINBDSWEEK || week > MAXBDSWEEK {
			err := fmt.Errorf("BDS week must be in %d...%d, but %d is given", MINBDSWEEK, MAXBDSWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("BDS dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINBDSWEEK && dow < MINBDSWEEK_MINDOW {
			err := fmt.Errorf("when BDS week is equal to %d, dow must not be less than %d, but %d is given",
				              MINBDSWEEK, MINBDSWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXBDSWEEK && dow > MAXBDSWEEK_MAXDOW {
			err := fmt.Errorf("when BDS week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXBDSWEEK, MAXBDSWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(2006, 1, 1)
	case 'E', 'e':
		if week < MINGALWEEK || week > MAXGALWEEK {
			err := fmt.Errorf("GAL week must be in %d...%d, but %d is given", MINGALWEEK, MAXGALWEEK, week)
			return _PyDate0, err
		} else if dow < 0 || dow > 6 {
			err := fmt.Errorf("GAL dow must be in 0...6, but %d is given", dow)
			return _PyDate0, err
		} else if week == MINGALWEEK && dow < MINGALWEEK_MINDOW {
			err := fmt.Errorf("when GAL week is equal to %d, dow must not be less than %d, but %d is given",
				              MINGALWEEK, MINGALWEEK_MINDOW, dow)
			return _PyDate0, err
		} else if week == MAXGALWEEK && dow > MAXGALWEEK_MAXDOW {
			err := fmt.Errorf("when GAL week is equal to %d, dow must not be greater than %d, but %d is given",
				              MAXGALWEEK, MAXGALWEEK_MAXDOW, dow)
			return _PyDate0, err
		}

		return NewPyDate(1999, 8, 22)
	default:
		err := fmt.Errorf("unsupported GNSS time system: '%c'\n"+
			              "Supported: \n"+
			              "    'A' or 'a' for TAI\n"+
			              "    'T' or 't' for TT\n"+
			              "    'U' or 'u' for UTC\n"+
			              "    'G' or 'g' for GPST\n"+
			              "    'R' or 'r' for GLONASS UTC\n"+
			              "    'C' or 'c' for BDT\n"+
			              "    'E' or 'e' for GST", flag)
		return _PyDate0, err
	}
}

func year_doy_check(year, doy int) error {
	if year < MINYEAR || year > MAXYEAR {
		return fmt.Errorf("year must be in %d...%d, but %d id given", MINYEAR, MAXYEAR, year)
	} else {
		var max_doy int = 365

		if isLeapYear(year) {
			max_doy++
		}

		if doy < 1 || doy > max_doy {
			return fmt.Errorf("for year %d, doy must be in 1...%d, but %d is given", year, max_doy, doy)
		}
	}

	return nil
}

/*
Convert year/month/day to GNSS week/dow.

Arguments:

	flag	GNSS system indicator.
	year, month, day

Return:

	week, dow
	err	 Error message.
*/
func Date2Week(flag byte, year, month, day int) (int, int, error) {
	date00, err := sys_week_dow_check(flag, 0, 0)

	if err != nil {
		return 0, 0, err
	}

	date, err := NewPyDate(year, month, day)

	if err != nil {
		return 0, 0, err
	}

	days := date.SubDate(date00)
	week := days / 7
	dow  := days % 7

	if dow < 0 {
		week -= 1
		dow += 7
	}

	return week, dow, err
}

/*
Convert GNSS week/dow to year/month/day.

Arguments:

	flag	GNSS system indicator, which could be 'G' or 'C' for now.
	week, dow

Return:

	year, month, day
	err	 Error message.
*/
func Week2Date(flag byte, week, dow int) (int, int, int, error) {
	date, err := sys_week_dow_check(flag, week, dow)

	if err != nil {
		return 0, 0, 0, err
	}

	err = date.AddEq(week*7 + dow)

	if err != nil {
		return 0, 0, 0, err
	}

	return date.Year(), date.Month(), date.Day(), nil
}

/*
Convert year/month/day to year/doy.

Arguments:

	year, month, day

Return:

	doy
	err	Error message.
*/
func Date2Doy(year, month, day int) (int, error) {
	date, err := NewPyDate(year, month, day)

	if err != nil {
		return 0, err
	}

	date00, _ := NewPyDate(year, 1, 1)

	return date.SubDate(date00) + 1, nil
}

/*
Convert year/doy to year/month/day.

Arguments:

	year, doy

Return:

	month, day
	err	Error message.
*/
func Doy2Date(year, doy int) (int, int, error) {
	err := year_doy_check(year, doy)

	if err != nil {
		return 0, 0, err
	}

	date, _ := NewPyDate(year, 1, 1)
	err = date.AddEq(doy - 1)

	if err != nil {
		return 0, 0, err
	}

	return date.Month(), date.Day(), nil
}

/*
Convert GNSS week/dow to year/doy.

Arguments:

	flag	GNSS system indicator.
	week, dow

Return:

	year, doy
	err	 Error message.
*/
func Week2Doy(flag byte, week, dow int) (int, int, error) {
	date00, err := sys_week_dow_check(flag, week, dow)

	if err != nil {
		return 0, 0, err
	}

	date, err := date00.ADD(week*7 + dow)

	if err != nil {
		return 0, 0, err
	}

	year := date.Year()
	date0, _ := NewPyDate(year, 1, 1)

	return year, date.SubDate(date0) + 1, nil
}

/*
Convert year/doy to GNSS week/dow.

Arguments:

	flag	GNSS system indicator.
	year, doy

Return:

	week, dow
	err	 Error message.
*/
func Doy2Week(flag byte, year, doy int) (int, int, error) {
	date00, err := sys_week_dow_check(flag, 0, 0)

	if err != nil {
		return 0, 0, err
	}

	err = year_doy_check(year, doy)

	if err != nil {
		return 0, 0, err
	}

	date0, _ := NewPyDate(year, 1, 1)

	date, err := date0.ADD(doy - 1)

	if err != nil {
		return 0, 0, err
	}

	var days int = date.SubDate(date00)

	var week int = days / 7
	var dow int = days % 7

	if dow < 0 {
		week -= 1
		dow += 7
	}

	return week, dow, nil
}

/*
Given a UTC date, determine which there is a leap second in that day.

Arguments:

	d	The date.

Return:

	0 or 1 (or -1)
*/
func utc_flag(d PyDate) int {
	if d.EQ(_LeapSecD1) || d.EQ(_LeapSecD2) || d.EQ(_LeapSecD3) || d.EQ(_LeapSecD4) ||
		d.EQ(_LeapSecD5) || d.EQ(_LeapSecD6) || d.EQ(_LeapSecD7) || d.EQ(_LeapSecD8) ||
		d.EQ(_LeapSecD9) || d.EQ(_LeapSecD10) || d.EQ(_LeapSecD11) || d.EQ(_LeapSecD12) ||
		d.EQ(_LeapSecD13) || d.EQ(_LeapSecD14) || d.EQ(_LeapSecD15) || d.EQ(_LeapSecD16) ||
		d.EQ(_LeapSecD17) || d.EQ(_LeapSecD18) || d.EQ(_LeapSecD19) || d.EQ(_LeapSecD20) ||
		d.EQ(_LeapSecD21) || d.EQ(_LeapSecD22) || d.EQ(_LeapSecD23) || d.EQ(_LeapSecD24) ||
		d.EQ(_LeapSecD25) || d.EQ(_LeapSecD26) || d.EQ(_LeapSecD27) {
		return 1
	} else {
		return 0
	}
}

/*
Calculate the value of leap seconds given a UTC date.

Arguments:

	d

Return:

	The value of leap seconds.
*/
func leap_seconds(d PyDate) int {
	var leapsec int

	if d.LE(_LeapSecD1) {
		leapsec = 10
	} else if d.GT(_LeapSecD1) && d.LE(_LeapSecD2) {
		leapsec = 11
	} else if d.GT(_LeapSecD2) && d.LE(_LeapSecD3) {
		leapsec = 12
	} else if d.GT(_LeapSecD3) && d.LE(_LeapSecD4) {
		leapsec = 13
	} else if d.GT(_LeapSecD4) && d.LE(_LeapSecD5) {
		leapsec = 14
	} else if d.GT(_LeapSecD5) && d.LE(_LeapSecD6) {
		leapsec = 15
	} else if d.GT(_LeapSecD6) && d.LE(_LeapSecD7) {
		leapsec = 16
	} else if d.GT(_LeapSecD7) && d.LE(_LeapSecD8) {
		leapsec = 17
	} else if d.GT(_LeapSecD8) && d.LE(_LeapSecD9) {
		leapsec = 18
	} else if d.GT(_LeapSecD9) && d.LE(_LeapSecD10) {
		leapsec = 19
	} else if d.GT(_LeapSecD10) && d.LE(_LeapSecD11) {
		leapsec = 20
	} else if d.GT(_LeapSecD11) && d.LE(_LeapSecD12) {
		leapsec = 21
	} else if d.GT(_LeapSecD12) && d.LE(_LeapSecD13) {
		leapsec = 22
	} else if d.GT(_LeapSecD13) && d.LE(_LeapSecD14) {
		leapsec = 23
	} else if d.GT(_LeapSecD14) && d.LE(_LeapSecD15) {
		leapsec = 24
	} else if d.GT(_LeapSecD15) && d.LE(_LeapSecD16) {
		leapsec = 25
	} else if d.GT(_LeapSecD16) && d.LE(_LeapSecD17) {
		leapsec = 26
	} else if d.GT(_LeapSecD17) && d.LE(_LeapSecD18) {
		leapsec = 27
	} else if d.GT(_LeapSecD18) && d.LE(_LeapSecD19) {
		leapsec = 28
	} else if d.GT(_LeapSecD19) && d.LE(_LeapSecD20) {
		leapsec = 29
	} else if d.GT(_LeapSecD20) && d.LE(_LeapSecD21) {
		leapsec = 30
	} else if d.GT(_LeapSecD21) && d.LE(_LeapSecD22) {
		leapsec = 31
	} else if d.GT(_LeapSecD22) && d.LE(_LeapSecD23) {
		leapsec = 32
	} else if d.GT(_LeapSecD23) && d.LE(_LeapSecD24) {
		leapsec = 33
	} else if d.GT(_LeapSecD24) && d.LE(_LeapSecD25) {
		leapsec = 34
	} else if d.GT(_LeapSecD26) && d.LE(_LeapSecD26) {
		leapsec = 35
	} else if d.GT(_LeapSecD26) && d.LE(_LeapSecD27) {
		leapsec = 36
	} else {
		leapsec = 37
	}

	return leapsec
}

/*
Convert TAI (in week/sow format) to TT/UTC/GPST/GLONASS-UTC/BDT/GST.

Arguments:

	week	The TAI week.
	sow	 The TAI sow.
	flag	The target time system.

Return:

	week/sow in the target time system.
*/
func fromTAI(week int, sow float64, flag byte) (int, float64) {
	if flag == 'T' || flag == 't' { // From TAI to TT.
		week -= 991
		sow -= 259167.816 // 259200 - 32.184, 1977-01-01 00:00:00 (TAI) is TAI week 991, sow 259200.
	} else if flag == 'U' || flag == 'u' { // From TAI to UTC.
		week -= 730
		sow -= 259210 // 259200 + 10, 1972-01-01 00:00:00 (UTC) is TAI week 730, sow 259210.
	} else if flag == 'G' || flag == 'g' { // From TAI to GPST.
		week -= 1148
		sow -= 345619 // 345600 + 19, 1980-01-06 00:00:00 (UTC) is TAI week 1148, sow 345619.
	} else if flag == 'R' || flag == 'r' { // From TAI to GLONASS-UTC.
		week -= 730
		sow -= 248410 // 259200 + 10 - 3*3600, 1971-12-31 21:00:00 (UTC) is TAI week 730, sow 248410.
	} else if flag == 'C' || flag == 'c' { // From TAI to BDT.
		week -= 2504
		sow -= 345633 // 345600 + 33, 2006-01-01 00:00:00 (UTC) is TAI week 2504, sow 345633.
	} else if flag == 'E' || flag == 'e' { // From TAI to GST.
		week -= 2172
		sow -= 345619 // 345600 + 19, 1999-08-21 23:59:47 (UTC) is TAI week 2172, sow 345619.
	}

	if sow < 0 {
		sow += 604800
		week--
	}

	return week, sow
}

/*
Convert TT/UTC/GPST/GLONASS-UTC/BDT/GST (in week/sow format) to TAI.

Arguments:

	flag	The original time system.
	week	Week in the original time system.
	sow	 Sow in the original time system.

Return:

	week/sow in TAI.
*/
func toTAI(flag byte, week int, sow float64) (int, float64) {
	if flag == 'T' || flag == 't' { // From TT to TAI.
		week += 991
		sow += 259167.816 // 259200 - 32.184, 1977-01-01 00:00:00 (TAI) is TAI week 991, sow 259200.
	} else if flag == 'U' || flag == 'u' { // From UTC to TAI.
		week += 730
		sow += 259210 // 259200 + 10, 1972-01-01 00:00:00 (UTC) is TAI week 730, sow 259210.
	} else if flag == 'G' || flag == 'g' { // From GPST to TAI.
		week += 1148
		sow += 345619
	} else if flag == 'R' || flag == 'r' { // From GLONASS-UTC to TAI.
		week += 730
		sow += 248410 // 259200 + 10, 1972-01-01 03:00:00 (UTC) is TAI week 730, sow 248410.
	} else if flag == 'C' || flag == 'c' { // From BDT to TAI.
		week += 2504
		sow += 345633
	} else if flag == 'E' || flag == 'e' { // From TAI to GST.
		week += 2172
		sow += 345619 // 345600 + 19, 1999-08-21 23:59:47 (UTC) is TAI week 2172, sow 345619.
	}

	if sow >= 604800 {
		sow -= 604800
		week++
	}

	return week, sow
}

/*
A function that compares two PyDate objects.
*/
func cmpPyDate(pd1 *PyDate, pd2 *PyDate) int {
	y1 := pd1.year
	m1 := pd1.month
	d1 := pd1.day
	y2 := pd2.year
	m2 := pd2.month
	d2 := pd2.day

	if y1 < y2 || (y1 == y2 && m1 < m2) || (y1 == y2 && m1 == m2 && d1 < d2) {
		return -1
	} else if y1 == y2 && m1 == m2 && d1 == d2 {
		return 0
	} else { // ( y1 > y2 || (y1 == y2 && m1 > m2) || (y1 == y2 && m1 == m2 && d1 > d2) )
		return 1
	}
}

/*
A function that compares two GNSSTime objects.
*/
func cmpGNSSTime(t1, t2 *GNSSTime) int {
	flags1 := []byte{t1.flag}
	flags2 := []byte{t2.flag}
	week1 := t1.week
	sow1  := t1.sow

	var week2 int
	var sow2 float64

	if bytes.Equal(bytes.ToUpper(flags2), bytes.ToUpper(flags1)) {
		week2 = t2.week
		sow2 = t2.sow
	} else {
		var t2_new, _ = t2.NewConvert(t1.flag)
		week2 = t2_new.week
		sow2 = t2_new.sow
	}

	if week1 < week2 || (week1 == week2 && sow1 < sow2) {
		return -1
	} else if week1 == week2 && sow1 == sow2 {
		return 0
	} else { // ( week1 > week2 || (week1 == week2 && sow1 > sow2) )
		return 1
	}
}
