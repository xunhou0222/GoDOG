package datetime

/***** CONSTANT ********************************/

const (
	DAY2HOUR      uint8   = 24
	HOUR2DAY      float64 = 1.0 / float64(DAY2HOUR)
	HOUR2MINUTE   uint8   = 60
	MINUTE2HOUR   float64 = 1.0 / float64(HOUR2MINUTE)
	MINUTE2SECOND uint8   = 60
	SECOND2MINUTE float64 = 1.0 / float64(MINUTE2SECOND)
	DAY2MINUTE    uint16  = uint16(DAY2HOUR) * uint16(HOUR2MINUTE)
	MINUTE2DAY    float64 = 1.0 / float64(DAY2MINUTE)
	DAY2SECOND    uint32  = uint32(DAY2HOUR) * uint32(HOUR2MINUTE) * uint32(MINUTE2SECOND)
	SECOND2DAY    float64 = 1.0 / float64(DAY2SECOND)
	HOUR2SECOND   uint16  = uint16(HOUR2MINUTE) * uint16(MINUTE2SECOND)
	SECOND2HOUR   float64 = 1.0 / float64(HOUR2SECOND)
	WEEK2DAY      uint8   = 7
	DAY2WEEK      float64 = 1.0 / float64(WEEK2DAY)
	WEEK2SECOND   uint32  = uint32(WEEK2DAY) * uint32(DAY2SECOND)
	SECOND2WEEK   float64 = 1.0 / float64(WEEK2SECOND)
)

/***********************************************/

const (
	TIME_EPSILON   float64 = 0.1e-6
	MJD_J2000      float64 = 51544.5
	JD_MJD0        float64 = 2400000.5
	DELTA_TAI_TT   float64 = -32.184
	DELTA_TAI_GPST float64 = 19.0
	DELTA_TAI_BDT  float64 = 33.0
	DELTA_TAI_GST  float64 = 32.0
	DELTA_TAI_UTC  float64 = 10.0
	DELTA_GLOT_UTC float64 = 3 * float64(HOUR2SECOND)
)

/***********************************************/

type LeapSecond struct {
	Value int8
	Total int16
	Mjd   int32
}

// leap seconds table of UTC (value, total value, mjd).
var UTC_LEAP_SEC []LeapSecond = []LeapSecond{
	{1, 27, 57754}, // 2017-01-01
	{1, 26, 57204}, // 2015-07-01
	{1, 25, 56109}, // 2012-07-01
	{1, 24, 54832}, // 2009-01-01
	{1, 23, 53736}, // 2006-01-01
	{1, 22, 51179}, // 1999-01-01
	{1, 21, 50630}, // 1997-07-01
	{1, 20, 50083}, // 1996-01-01
	{1, 19, 49534}, // 1994-07-01
	{1, 18, 49169}, // 1993-07-01
	{1, 17, 48804}, // 1992-07-01
	{1, 16, 48257}, // 1991-01-01
	{1, 15, 47892}, // 1990-01-01
	{1, 14, 47161}, // 1988-01-01
	{1, 13, 46247}, // 1985-07-01
	{1, 12, 45516}, // 1983-07-01
	{1, 11, 45151}, // 1982-07-01
	{1, 10, 44786}, // 1981-07-01
	{1, 9, 44239},  // 1980-01-01
	{1, 8, 43874},  // 1979-01-01
	{1, 7, 43509},  // 1978-01-01
	{1, 6, 43144},  // 1977-01-01
	{1, 5, 42778},  // 1976-01-01
	{1, 4, 42413},  // 1975-01-01
	{1, 3, 42048},  // 1974-01-01
	{1, 2, 41683},  // 1973-01-01
	{1, 1, 41499},  // 1972-07-01
}

/***********************************************/

type TimeSys uint8

const (
	TIME_SYS_NONE     TimeSys = iota // used when Time represents a time interval rather than an epoch
	TIME_SYS_TAI                     // international atomic time
	TIME_SYS_TT                      // terrestrial time
	TIME_SYS_UTC                     // coordinate universal time
	TIME_SYS_GPST                    // GPS time
	TIME_SYS_GLONASST                // GLONASS time
	TIME_SYS_BDT                     // BDS time
	TIME_SYS_GST                     // Galileo time
)

var TimeSys2Name map[TimeSys]string = map[TimeSys]string{
	TIME_SYS_NONE:     "NONE",
	TIME_SYS_TAI:      "TAI",
	TIME_SYS_TT:       "TT",
	TIME_SYS_UTC:      "UTC",
	TIME_SYS_GPST:     "GPST",
	TIME_SYS_GLONASST: "GLONASST",
	TIME_SYS_BDT:      "BDT",
	TIME_SYS_GST:      "GST",
}

var Name2TimeSys map[string]TimeSys = map[string]TimeSys{
	"NONE":     TIME_SYS_NONE,
	"TAI":      TIME_SYS_TAI,
	"TT":       TIME_SYS_TT,
	"UTC":      TIME_SYS_UTC,
	"GPST":     TIME_SYS_GPST,
	"GLONASST": TIME_SYS_GLONASST,
	"BDT":      TIME_SYS_BDT,
	"GST":      TIME_SYS_GST,
}

/***********************************************/

var (
	_DAYS_IN_MONTH     [12]uint8  = [12]uint8{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	_DAYS_BEFORE_MONTH [12]uint16 = [12]uint16{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
)

const (
	_MJD_ORD1     int32 = -94187 // mjd of 01-Jan-1601
	_ORD_REF_YEAR int32 = 1601
)

/***********************************************/

var (
	TIME_GPST0 = DateTime2Time(TIME_SYS_GPST, 1980, 1, 6, 0, 0, 0)
	TIME_BDT0  = DateTime2Time(TIME_SYS_BDT, 2006, 1, 1, 0, 0, 0)
	TIME_GST0  = DateTime2Time(TIME_SYS_GST, 1999, 8, 21, 23, 59, 47.0)
)

/***********************************************/
