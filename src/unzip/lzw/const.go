package lzw

const (
	magicByte1 = 0x1F
	magicByte2 = 0x9D
	otherByte  = 0x60
	bitMask    = 0x1F
	minBits    = 9
	// minBits2   = 10
	maxBits    = 16
	blockMode  = 0x80
	iBufSize   = 0x40000
	iBufExtra  = 64
	oBufSize   = 0x40000
	oBufExtra  = 2048
	blockClear = 256
	blockFirst = 257
)
