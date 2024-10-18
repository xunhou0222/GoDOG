package gnsstime

// -----------------------------------------------------------------------------------
// 1. PyDate properties (some special methods that get values of private member variables)
func (pd PyDate) Year() int {
	return pd.year
}

func (pd PyDate) Month() int {
	return pd.month
}

func (pd PyDate) Day() int {
	return pd.day
}

// -----------------------------------------------------------------------------------

// 2. GNSSTime properties (some special methods that get values of private member variables).
func (t GNSSTime) Flag() byte {
	return t.flag
}

func (t GNSSTime) Year() int {
	return t.year
}

func (t GNSSTime) Month() int {
	return t.month
}

func (t GNSSTime) Day() int {
	return t.day
}

func (t GNSSTime) Hour() int {
	return t.hour
}

func (t GNSSTime) Minute() int {
	return t.minute
}

func (t GNSSTime) Second() float64 {
	return t.second
}

func (t GNSSTime) Doy() int {
	return t.doy
}

func (t GNSSTime) Sod() float64 {
	return t.sod
}

func (t GNSSTime) Week() int {
	return t.week
}

func (t GNSSTime) Dow() int {
	return t.dow
}

func (t GNSSTime) Sow() float64 {
	return t.sow
}

// -----------------------------------------------------------------------------------
