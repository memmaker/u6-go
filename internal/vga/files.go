package vga

import (
	"encoding/binary"
	"fmt"
	"io/fs"
)

const charSize = 8

// Font is an 8x8 character set: 8 rows per character, leftmost pixel in the high bit (U6.CH).
type Font [256][charSize]byte

// LoadFont reads U6.CH.
func LoadFont(files fs.FS) (*Font, error) {
	raw, err := fs.ReadFile(files, "U6.CH")
	if err != nil {
		return nil, err
	}
	font := &Font{}
	if len(raw) < len(font)*charSize {
		return nil, fmt.Errorf("U6.CH: short font")
	}
	for ch := range font {
		copy(font[ch][:], raw[ch*charSize:])
	}
	return font, nil
}

// LoadPalette reads U6PAL: 256 colours of three 6-bit components.
func LoadPalette(files fs.FS) (*[paletteBytes]byte, error) {
	raw, err := fs.ReadFile(files, "U6PAL")
	if err != nil {
		return nil, err
	}
	palette := &[paletteBytes]byte{}
	if len(raw) < paletteBytes {
		return nil, fmt.Errorf("U6PAL: short palette")
	}
	copy(palette[:], raw)
	return palette, nil
}

// Pixmap is a picture drawn by DrawPixmap (tPixmap in gr.h).
type Pixmap struct {
	Width, Height int
	Pixels        []byte
}

// LoadPixmap reads a packed picture: a 16-bit width and height, then the pixels row by row
// (PAPER.BMP, NEWMAGIC.BMP, WORLDMAP.BMP, ...).
func LoadPixmap(files fs.FS, name string) (*Pixmap, error) {
	raw, err := readPacked(files, name)
	if err != nil {
		return nil, err
	}
	const headerBytes = 4
	if len(raw) < headerBytes {
		return nil, fmt.Errorf("%s: short picture", name)
	}
	p := &Pixmap{Width: int(binary.LittleEndian.Uint16(raw)), Height: int(binary.LittleEndian.Uint16(raw[2:]))}
	if len(raw) < headerBytes+p.Width*p.Height {
		return nil, fmt.Errorf("%s: short picture", name)
	}
	p.Pixels = raw[headerBytes : headerBytes+p.Width*p.Height]
	return p, nil
}
