// Package data reads the game's data files.
package data

import (
	"encoding/binary"
	"errors"
)

// The game's compressed files (LZMAP, LZOBJBLK, LOOK.LZD, ... and the inner parts of the .SHP
// and library files) hold a 4-byte little-endian size of the unpacked data followed by an
// LZW stream of 9 to 12 bit codes, low bit first (INFLATE.ASM, LZ.INC).
const (
	sizeHeaderBytes = 4
	firstCodeBits   = 9
	maxCodeBits     = 12
	clearCode       = 256
	endCode         = 257
	firstFreeCode   = 258
)

var errBadLZW = errors.New("lzw: corrupt stream")

// Decompress unpacks one compressed block.
func Decompress(packed []byte) ([]byte, error) {
	if len(packed) < sizeHeaderBytes {
		return nil, errBadLZW
	}
	size := binary.LittleEndian.Uint32(packed)
	in := packed[sizeHeaderBytes:]
	out := make([]byte, 0, size)

	type entry struct {
		prefix int
		last   byte
	}
	dict := make([]entry, 1<<maxCodeBits)
	codeBits := firstCodeBits
	freeCode := firstFreeCode
	previous := -1
	bitPos := 0
	stack := make([]byte, 0, 1<<maxCodeBits)

	// expand appends the string of code to out and returns its first byte.
	expand := func(code int) byte {
		stack = stack[:0]
		for code >= firstFreeCode {
			stack = append(stack, dict[code].last)
			code = dict[code].prefix
		}
		first := byte(code)
		out = append(out, first)
		for i := len(stack) - 1; i >= 0; i-- {
			out = append(out, stack[i])
		}
		return first
	}

	for {
		byteAt := bitPos / 8
		if byteAt >= len(in) {
			return nil, errBadLZW
		}
		var window int
		for i := 2; i >= 0; i-- {
			window <<= 8
			if byteAt+i < len(in) {
				window |= int(in[byteAt+i])
			}
		}
		code := window >> (bitPos % 8) & (1<<codeBits - 1)
		bitPos += codeBits

		switch {
		case code == clearCode:
			codeBits = firstCodeBits
			freeCode = firstFreeCode
			previous = -1
			continue
		case code == endCode:
			return out, nil
		case previous < 0:
			if code >= clearCode {
				return nil, errBadLZW
			}
			out = append(out, byte(code))
			previous = code
			continue
		}

		var first byte
		switch {
		case code < freeCode:
			first = expand(code)
		case code == freeCode: // the string being defined: previous string + its own first byte
			first = expand(previous)
			out = append(out, first)
		default:
			return nil, errBadLZW
		}
		if freeCode < 1<<maxCodeBits {
			dict[freeCode] = entry{previous, first}
			freeCode++
			if freeCode == 1<<codeBits && codeBits < maxCodeBits {
				codeBits++
			}
		}
		previous = code
	}
}
