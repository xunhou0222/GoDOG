package gnsstime

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

// ---------------------------------------------------------------------------------------
// 1. PyDate methods
/// 1.1 Operators
/*
   Whether pd is less than other
*/
func (pd PyDate) LT(other PyDate) bool {
	return cmpPyDate(&pd, &other) < 0
}

/*
Whether pd is less than or equal to other
*/
func (pd PyDate) LE(other PyDate) bool {
	return cmpPyDate(&pd, &other) <= 0
}

/*
Whether pd is greater than other
*/
func (pd PyDate) GT(other PyDate) bool {
	return cmpPyDate(&pd, &other) > 0
}

/*
Whether pd is greater than or equal to other
*/
func (pd PyDate) GE(other PyDate) bool {
	return cmpPyDate(&pd, &other) >= 0
}

/*
Whether pd is equal to other
*/
func (pd PyDate) EQ(other PyDate) bool {
	return cmpPyDate(&pd, &other) == 0
}

/*
Whether pd is not equal to other
*/
func (pd PyDate) NE(other PyDate) bool {
	return cmpPyDate(&pd, &other) != 0
}

/*
Add several days to pd to get a new "Date"
*/
func (pd PyDate) ADD(days int) (PyDate, error) {
	ord, _ := ymd2ord(pd.year, pd.month, pd.day)

	ord += days
	year, mon, day, err := ord2ymd(ord)

	if err != nil {
		return _PyDate0, err
	}

	return PyDate{year, mon, day}, nil
}

func (pd *PyDate) AddEq(days int) error {
	ord, _ := ymd2ord(pd.year, pd.month, pd.day)
	ord += days
	year, mon, day, err := ord2ymd(ord)

	if err != nil {
		return err
	}

	pd.year = year
	pd.month = mon
	pd.day = day

	return nil
}

/*
Substract several days from pd to get a new "Date"
*/
func (pd PyDate) SUB(days int) (PyDate, error) {
	ord, _ := ymd2ord(pd.year, pd.month, pd.day)
	ord -= days

	year, mon, day, err := ord2ymd(ord)

	if err != nil {
		return _PyDate0, err
	}

	return PyDate{year, mon, day}, nil
}

func (pd *PyDate) SubEq(days int) error {
	ord, _ := ymd2ord(pd.year, pd.month, pd.day)
	ord -= days

	year, mon, day, err := ord2ymd(ord)

	if err != nil {
		return err
	}

	pd.year = year
	pd.month = mon
	pd.day = day

	return nil
}

/*
Substract other (another "Date") from pd to get days between them
*/
func (pd PyDate) SubDate(other PyDate) int {
	ord1, _ := ymd2ord(pd.year, pd.month, pd.day)
	ord2, _ := ymd2ord(other.year, other.month, other.day)

	return ord1 - ord2
}

// / 1.2 Other mothods
func (pd PyDate) StrFormat(template string) string {
	var yy int

	if pd.year < 2000 {
		yy = pd.year - 1900
	} else {
		yy = pd.year - 2000
	}

	template = strings.ReplaceAll(template, "<YYYY>", fmt.Sprintf("%04d", pd.year))
	template = strings.ReplaceAll(template, "<YY>", fmt.Sprintf("%02d", yy))
	template = strings.ReplaceAll(template, "<MM>", fmt.Sprintf("%02d", pd.month))
	template = strings.ReplaceAll(template, "<DD>", fmt.Sprintf("%02d", pd.day))

	return template
}

// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// 2. GNSSTime methods
/// 2.1 Operators
/*
	Whether t is less than other
*/
func (t GNSSTime) LT(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) < 0
}

/*
Whether t is less than or equal to other
*/
func (t GNSSTime) LE(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) <= 0
}

/*
Whether t is equal to other
*/
func (t GNSSTime) EQ(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) == 0
}

/*
Whether t is greater than other
*/
func (t GNSSTime) GT(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) > 0
}

/*
Whether t is greater than or equal to other
*/
func (t GNSSTime) GE(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) >= 0
}

/*
Whether t is not equal to other
*/
func (t GNSSTime) NE(other GNSSTime) bool {
	return cmpGNSSTime(&t, &other) != 0
}

/*
Add several seconds to t to get a new "GNSSTime"
*/
func (t GNSSTime) ADD(seconds float64) (GNSSTime, error) {
	week := t.week
	sow  := t.sow + seconds

	for sow >= 604800 {
		sow -= 604800
		week += 1
	}

	for sow < 0 {
		sow += 604800
		week -= 1
	}

	return FromWeekSow(t.flag, week, sow)
}

/*
Substract several seconds from t to get a new "GNSSTime"
*/
func (t GNSSTime) SUB(seconds float64) (GNSSTime, error) {
	week := t.week
	sow  := t.sow - seconds

	for sow >= 604800 {
		sow -= 604800
		week += 1
	}

	for sow < 0 {
		sow += 604800
		week -= 1
	}

	return FromWeekSow(t.flag, week, sow)
}

/*
Substract other (another "GNSSTime") from t to get seconds between them
*/
func (t GNSSTime) SubTime(other GNSSTime) float64 {
	if t.flag == other.flag {
		return float64((t.week-other.week)*604800) + (t.sow - other.sow)
	} else {
		var other_s, _ = other.NewConvert(t.flag)
		return float64((t.week-other_s.week)*604800) + (t.sow - other_s.sow)
	}
}

/*
Add several seconds to t
*/
func (t *GNSSTime) AddEq(seconds float64) error {
	week := t.week
	sow  := t.sow + seconds

	for sow >= 604800 {
		sow -= 604800
		week += 1
	}

	for sow < 0 {
		sow += 604800
		week -= 1
	}

	t_new, err := FromWeekSow(t.flag, week, sow)

	if err != nil {
		return err
	} else {
		*t = t_new
		return nil
	}
}

/*
Substract several seconds from t
*/
func (t *GNSSTime) SubEq(seconds float64) error {
	week := t.week
	sow  := t.sow - seconds

	for sow >= 604800 {
		sow -= 604800
		week += 1
	}

	for sow < 0 {
		sow += 604800
		week -= 1
	}
	
	t_new, err := FromWeekSow(t.flag, week, sow)

	if err != nil {
		return err
	} else {
		*t = t_new
		return nil
	}
}

// / 2.2 Convertion methods.
func (t GNSSTime) NewConvert(flag byte) (GNSSTime, error) {
	flags0 := []byte{t.flag}
	flags  := []byte{flag}

	_, err := sys_week_dow_check(flag, 0, 0)

	if err != nil {
		return _GNSSTime0, err
	}

	if bytes.Equal(bytes.ToUpper(flags), bytes.ToUpper(flags0)) {
		return t, nil
	}

	week := t.week
	sow  := t.sow

	if t.flag == 'A' || t.flag == 'a' { // From TAI to others.
		week, sow = fromTAI(week, sow, flag)
	} else if flag == 'A' || flag == 'a' { // From others to TAI.
		week, sow = toTAI(t.flag, week, sow)
	} else { // From others to others.
		week, sow = toTAI(t.flag, week, sow)
		week, sow = fromTAI(week, sow, flag)
	}

	return FromWeekSow(flag, week, sow)
}

func (t *GNSSTime) SelfConvert(flag byte) error {
	t_new, err := t.NewConvert(flag)

	if err != nil {
		return err
	} else {
		*t = t_new
		return nil
	}
}

// / 2.3 Other methods.
func (t GNSSTime) Date() PyDate {
	d, _ := NewPyDate(t.year, t.month, t.day)

	return d
}

/*
Calculate MJD of the GNSSTime.

Return:

	MJD_int, MJD_frac	The integer and fractional part of MJD.
*/
func (t GNSSTime) MJD() (int, float64) {
	date0, _ := NewPyDate(1858, 11, 17)
	MJD_int := t.Date().SubDate(date0)
	MJD_frac := t.sod / 86400

	return MJD_int, MJD_frac
}

func (t GNSSTime) StrFormat(template string, n int) string {
	week := t.week
	sow := t.sow

	if n < 0 {
		n = 0
	}

	if 1.0-sow+float64(int(sow)) < 0.5/math.Pow10(n) {
		sow = float64(int(sow)) + 1.0

		if sow >= 604800 {
			week++
			sow -= 604800
		}
	}

	tt, _ := FromWeekSow(t.flag, week, sow)

	var yy, w, p int

	if tt.year < 2000 {
		yy = tt.year - 1900
	} else {
		yy = tt.year - 2000
	}

	template = strings.ReplaceAll(template, "<YYYY>", fmt.Sprintf("%04d", tt.year))
	template = strings.ReplaceAll(template, "<YY>", fmt.Sprintf("%02d", yy))
	template = strings.ReplaceAll(template, "<MM>", fmt.Sprintf("%02d", tt.month))
	template = strings.ReplaceAll(template, "<DD>", fmt.Sprintf("%02d", tt.day))
	template = strings.ReplaceAll(template, "<HH>", fmt.Sprintf("%02d", tt.hour))
	template = strings.ReplaceAll(template, "<mm>", fmt.Sprintf("%02d", tt.minute))
	template = strings.ReplaceAll(template, "<DOY>", fmt.Sprintf("%03d", tt.doy))
	template = strings.ReplaceAll(template, "<WEEK>", fmt.Sprintf("%04d", tt.week))
	template = strings.ReplaceAll(template, "<d>", fmt.Sprintf("%01d", tt.dow))

	if n <= 0 {
		w = 2
		p = 0
	} else {
		w = n + 3
		p = n
	}

	template = strings.ReplaceAll(template, "<SS>", fmt.Sprintf("%0*.*f", w, p, tt.second))

	if n <= 0 {
		w = 5
		p = 0
	} else {
		w = n + 6
		p = n
	}

	template = strings.ReplaceAll(template, "<SOD>", fmt.Sprintf("%0*.*f", w, p, tt.sod))

	if n <= 0 {
		w = 6
		p = 0
	} else {
		w = n + 7
		p = n
	}

	template = strings.ReplaceAll(template, "<SOWSSS>", fmt.Sprintf("%0*.*f", w, p, tt.sow))

	return template
}

// ---------------------------------------------------------------------------------------
