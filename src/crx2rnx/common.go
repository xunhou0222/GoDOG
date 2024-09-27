package crx2rnx

const (
	maxDiffOrder = 5
	maxSatNum    = 100
	maxTypeNum   = 100
)

type TypePRN [3]byte

// structure for fields of clock offset
type ClockFormat struct {
	upper [maxDiffOrder + 1]int64 // upper X digits for each difference order
	lower [maxDiffOrder + 1]int64 // lower 8 digits
}

// structure for fields of observation records
type DataFormat struct {
	upper    [maxDiffOrder + 1]int64 // upper X digits for each difference order
	lower    [maxDiffOrder + 1]int64 // lower 5 digits
	order    int
	arcOrder int
}

type SatInfo struct {
	TypeNum int // number of observation types
	OldIdx  int // The index in the old satellite list (-1 if is a new satellite)
}
