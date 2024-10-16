package gnsstime

// -----------------------------------------------------------------------------------
// 1. PyDate-related constants

const (
	MINYEAR    = 1
	MAXYEAR    = 9999
	MAXORDINAL = 3652059
)

var _DAYS_IN_MONTH = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
var _DAYS_BEFORE_MONTH = [12]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

var _PyDate0 PyDate = InitPyDate()

// Dates with leap seconds.
var (
	_LeapSecD1,  _ = NewPyDate(1972, 6, 30)
	_LeapSecD2,  _ = NewPyDate(1972, 12, 31)
	_LeapSecD3,  _ = NewPyDate(1973, 12, 31)
	_LeapSecD4,  _ = NewPyDate(1974, 12, 31)
	_LeapSecD5,  _ = NewPyDate(1975, 12, 31)
	_LeapSecD6,  _ = NewPyDate(1976, 12, 31)
	_LeapSecD7,  _ = NewPyDate(1977, 12, 31)
	_LeapSecD8,  _ = NewPyDate(1978, 12, 31)
	_LeapSecD9,  _ = NewPyDate(1979, 12, 31)
	_LeapSecD10, _ = NewPyDate(1981, 6, 30)
	_LeapSecD11, _ = NewPyDate(1982, 6, 30)
	_LeapSecD12, _ = NewPyDate(1983, 6, 30)
	_LeapSecD13, _ = NewPyDate(1985, 6, 30)
	_LeapSecD14, _ = NewPyDate(1987, 12, 31)
	_LeapSecD15, _ = NewPyDate(1989, 12, 31)
	_LeapSecD16, _ = NewPyDate(1990, 12, 31)
	_LeapSecD17, _ = NewPyDate(1992, 6, 30)
	_LeapSecD18, _ = NewPyDate(1993, 6, 30)
	_LeapSecD19, _ = NewPyDate(1994, 6, 30)
	_LeapSecD20, _ = NewPyDate(1995, 12, 31)
	_LeapSecD21, _ = NewPyDate(1997, 6, 30)
	_LeapSecD22, _ = NewPyDate(1998, 12, 31)
	_LeapSecD23, _ = NewPyDate(2005, 12, 31)
	_LeapSecD24, _ = NewPyDate(2008, 12, 31)
	_LeapSecD25, _ = NewPyDate(2012, 6, 30)
	_LeapSecD26, _ = NewPyDate(2015, 6, 30)
	_LeapSecD27, _ = NewPyDate(2016, 12, 31)
)

// -----------------------------------------------------------------------------------

// -----------------------------------------------------------------------------------
// 2. GNSSTime-related constants

const (
	MINTAIWEEK        = -102112 // 1-1-1
	MINTAIWEEK_MINDOW = 5       // If week == MINTAIWEEK, dow must >= MINTAIWEEK_MINDOW
	MAXTAIWEEK        = 419611  // 9999/12/31
	MAXTAIWEEK_MAXDOW = 2       // If week == MAXTAIWEEK, dow must <= MAXTAIWEEK_MAXDOW

	MINTTWEEK        = -103103 // 1-1-1
	MINTTWEEK_MINDOW = 2       // If week == MINTTWEEK, dow must >= MINTTWEEK_MINDOW
	MAXTTWEEK        = 418619  // 9999/12/31
	MAXTTWEEK_MAXDOW = 6       // If week == MAXTTWEEK, dow must <= MAXTTWEEK_MAXDOW

	MINUTCWEEK        = -102842 // 1-1-1
	MINUTCWEEK_MINDOW = 2       // 1-1-1 // If week == MINUTCWEEK, dow must >= MINUTCWEEK_MINDOW
	MAXUTCWEEK        = 418881  // 9999/12/31
	MAXUTCWEEK_MAXDOW = 0       // If week == MAXUTCWEEK, dow must <= MAXUTCWEEK_MAXDOW

	MINGPSWEEK        = -103260 // 1-1-1
	MINGPSWEEK_MINDOW = 1       // If week == MINGPSWEEK, dow must >= MINGPSWEEK_MINDOW
	MAXGPSWEEK        = 418462  // 9999/12/31
	MAXGPSWEEK_MAXDOW = 5       // If week == MAXGPSWEEK, dow must <= MAXGPSWEEK_MAXDOW

	MINGLOWEEK        = -102842 // 1-1-1
	MINGLOWEEK_MINDOW = 2       // If week == MINGLOWEEK, dow must >= MINGLOWEEK_MINDOW
	MAXGLOWEEK        = 418881  // 9999/12/31
	MAXGLOWEEK_MAXDOW = 0       // If week == MAXGLOWEEK, dow must <= MAXGLOWEEK_MAXDOW

	MINBDSWEEK        = -104616 // 1-1-1
	MINBDSWEEK_MINDOW = 1       // If week == MINBDSWEEK, dow must >= MINBDSWEEK_MINDOW
	MAXBDSWEEK        = 417106  // 9999/12/31
	MAXBDSWEEK_MAXDOW = 5       // If week == MAXBDSWEEK, dow must <= MAXBDSWEEK_MAXDOW

	MINGALWEEK        = -104284 // 1-1-1
	MINGALWEEK_MINDOW = 1       // If week == MINGALWEEK, dow must >= MINGALWEEK_MINDOW
	MAXGALWEEK        = 417438  // 9999/12/31
	MAXGALWEEK_MAXDOW = 5       // If week == MAXGALWEEK, dow must <= MAXGALWEEK_MAXDOW
)

var SUPPORTED_TIME_SYS [14]byte = [14]byte{'A', 'a', 'T', 't', 'U', 'u', 'G', 'g', 'R', 'r', 'C', 'c', 'E', 'e'}

var _GNSSTime0 GNSSTime = InitGNSSTime()

// -----------------------------------------------------------------------------------
