package lzw

import (
	"errors"
	"io"
)

type Reader struct {
	reader  io.Reader //valid after NewReader or Reader.Reset
	MaxBits byte      // set when reading the header of the .Z file
	IfBlock bool      // set when reading the header of the .Z file
	err     error

	iBuf     [iBufSize + iBufExtra]byte // input buffer
	iBufLen  int32                      // length of iBuf
	iPosBit  int32
	iBufBits int32

	oBuf    [oBufSize + oBufExtra]byte // output buffer
	oBufLen int32                      // length of oBuf
	oPos    int32                      // index of the next character to be output

	prefix [1 << maxBits]uint16
	suffix [1 << maxBits]byte

	bits       uint32
	mask       uint32
	maxCode    int64
	maxMaxCode int64
	entry      int64
	prev       int64
	final      int64
	flag       bool
}

func (z *Reader) readHeader() error {
	var n int
	var err error

	if n, err = io.ReadFull(z.reader, z.iBuf[:iBufSize]); n < 3 || (err != nil && err != io.EOF && err != io.ErrUnexpectedEOF) {
		return errors.New("lzc: invalid length of header")
	}

	z.iBufLen, z.iPosBit = int32(n), (3 << 3)
	z.flag = true

	if z.iBuf[0] != magicByte1 || z.iBuf[1] != magicByte2 {
		return errors.New("lzc: invalid magic bytes in the header")
	}

	if (z.iBuf[2] & otherByte) != 0 {
		return errors.New("lzc: invalid flag byte in the header")
	}

	z.MaxBits = z.iBuf[2] & bitMask

	if z.MaxBits < minBits || z.MaxBits > maxBits {
		return errors.New("lzc: max bits of code size in the header is out-of-range")

	}

	// if z.MaxBits == minBits { // 9 doesn't really mean 9
	// 	z.MaxBits = minBits2
	// }
	z.maxMaxCode = 1 << z.MaxBits
	z.IfBlock = (z.iBuf[2] & blockMode) != 0 // true if block compressed
	return nil
}

func (z *Reader) clearPrefix() {
	for i := 0; i < 256; i++ {
		z.prefix[i] = 0
	}
}

func (z *Reader) isOtherErr() bool {
	if z.err != nil && z.err != io.EOF {
		return true
	} else {
		return false
	}
}

func (z *Reader) unlzw() {
	var o, e int32
	var code, tmpCode, tmpFinal int64
	var stack []byte

loop:
	for n := int(z.iBufLen); n > 0; {
		if z.flag {
			// move tailing characters in z.iBuf forwards
			o = z.iPosBit >> 3

			if o <= z.iBufLen {
				e = z.iBufLen - o
			} else {
				e = 0
			}

			for i := int32(0); i < e; i++ {
				z.iBuf[i] = z.iBuf[i+o]
			}

			z.iBufLen = e
			z.iPosBit = 0

			if z.iBufLen < iBufExtra {
				n, z.err = z.reader.Read(z.iBuf[z.iBufLen : z.iBufLen+iBufSize])

				if z.isOtherErr() {
					return
				}

				z.iBufLen += int32(n)
			}

			if n != 0 {
				z.iBufBits = (z.iBufLen - z.iBufLen%int32(z.bits)) << 3
			} else {
				z.iBufBits = (z.iBufLen << 3) - int32(z.bits-1)
			}
		}

		for z.iPosBit < z.iBufBits {
			if z.entry > z.maxCode {
				z.iPosBit = (z.iPosBit - 1) + (int32(z.bits<<3) - (z.iPosBit-1+int32(z.bits<<3))%int32(z.bits<<3))
				z.bits++

				if z.bits == uint32(z.MaxBits) {
					z.maxCode = z.maxMaxCode
				} else {
					z.maxCode = (1 << z.bits) - 1
				}

				z.mask = (1 << z.bits) - 1
				z.flag = true
				continue loop
			}

			o = z.iPosBit >> 3
			code = ((int64(z.iBuf[o]) | (int64(z.iBuf[o+1]) << 8) | (int64(z.iBuf[o+2]) << 16)) >>
				int64(z.iPosBit&0x7)) & int64(z.mask)
			z.iPosBit += int32(z.bits)

			if z.prev == -1 {
				if code >= 256 {
					z.err = errors.New("lzc: the first code is not a literal")
					return
				}

				z.prev = code
				z.final = code
				z.oBuf[z.oBufLen] = byte(code)
				z.oBufLen++
				continue
			}

			if code == blockClear && z.IfBlock {
				z.clearPrefix()
				z.entry = blockFirst - 1
				z.iPosBit = (z.iPosBit - 1) + int32(z.bits<<3) - (z.iPosBit-1+int32(z.bits<<3))%int32(z.bits<<3)
				z.bits = minBits
				z.mask = (1 << z.bits) - 1
				z.maxCode = (1 << z.bits) - 1
				z.flag = true
				continue loop
			}

			// process LZW code
			tmpCode = code     // save the current code
			stack = stack[0:0] // buffer for reversed match - empty stack

			// special string like "aBaBa", where "a" represents a single character,
			// and "B" represents a string of arbitrary length
			if code >= z.entry {
				if code > z.entry {
					z.err = errors.New("lzc: invalid LZW code")
					return
				}

				stack = append(stack, byte(z.final))
				code = z.prev
			}

			// walk through linked list to generate output in reverse order
			for code >= 256 {
				stack = append(stack, z.suffix[code])
				code = int64(z.prefix[code])
			}

			tmpFinal = z.final
			z.final = int64(z.suffix[code])
			stack = append(stack, byte(z.final))

			if z.oBufLen+int32(len(stack)) >= oBufSize {
				z.iPosBit -= int32(z.bits)
				z.final = tmpFinal
				z.flag = false
				return
			}

			// link new table entry
			if z.entry < z.maxMaxCode {
				z.prefix[z.entry] = uint16(z.prev)
				z.suffix[z.entry] = byte(z.final)
				z.entry++
			}

			// set previous code for next iteration
			z.prev = tmpCode

			// Write stack to output in forward order
			for i := len(stack) - 1; i >= 0; i-- {
				z.oBuf[z.oBufLen] = stack[i]
				z.oBufLen++
			}
		}

		z.flag = true
	}
}

func NewReader(r io.Reader) (*Reader, error) {
	z := new(Reader)
	z.reader = r
	z.err = z.readHeader() // Read the header and set z.MaxBits and z.IfBlock

	if z.err != nil {
		return nil, z.err
	}

	z.oBufLen, z.oPos = 0, 0

	z.clearPrefix()

	for i := 0; i < 256; i++ {
		z.suffix[i] = byte(i)
	}

	z.bits = minBits
	z.mask = (1 << z.bits) - 1
	z.maxCode = (1 << z.bits) - 1

	if z.IfBlock {
		z.entry = blockFirst
	} else {
		z.entry = blockFirst - 1
	}

	z.prev = -1
	return z, nil
}

func (z *Reader) Read(p []byte) (n int, err error) {
	if z.isOtherErr() {
		return 0, z.err
	}

	lenP := len(p)

	for n < lenP {
		for z.oPos < z.oBufLen && n < lenP {
			p[n] = z.oBuf[z.oPos]
			n++
			z.oPos++
		}

		if z.isOtherErr() || (z.err == io.EOF && z.iPosBit >= z.iBufBits) {
			break
		}

		if n < lenP {
			z.oBufLen, z.oPos = 0, 0
			z.unlzw()
		}
	}

	if z.oBufLen-z.oPos > 0 && z.err == io.EOF {
		return n, nil
	}

	return n, z.err
}
