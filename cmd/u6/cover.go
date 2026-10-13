package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/memmaker/u6-go/internal/vga"
)

const (
	coverWidth  = 320
	coverHeight = 200
	// the title logo: TITLES.SHP's first shape, on the main menu's palette (PALETTES.INT's first)
	titleLibrary = "TITLES.SHP"
	titleShape   = 0
	paletteFile  = "PALETTES.INT"
	paletteSize  = 256
	// PALETTES.INT packs four 6-bit components into three bytes
	componentBits    = 6
	componentMask    = 1<<componentBits - 1
	componentsInPack = 4
	bytesInPack      = 3
	vgaMaxLevel      = componentMask
)

// writeCover writes dir/COVER.PNG, the picture goplay shows on the game's floppy label: the
// Ultima VI title logo, centred on black.
func writeCover(dir string) error {
	files := os.DirFS(dir)
	shapes, err := vga.LoadShapes(files, titleLibrary)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(dir, paletteFile))
	if err != nil {
		return err
	}
	var components []int
	for at := 0; at+bytesInPack <= len(raw) && len(components) < paletteSize*3; at += bytesInPack {
		bits := int(raw[at]) | int(raw[at+1])<<8 | int(raw[at+2])<<16
		for k := range componentsInPack {
			components = append(components, bits>>(k*componentBits)&componentMask)
		}
	}
	palette := make(color.Palette, paletteSize)
	for i := range palette {
		level := func(c int) uint8 { return uint8(components[i*3+c] * 255 / vgaMaxLevel) }
		palette[i] = color.RGBA{level(0), level(1), level(2), 255}
	}

	logo := shapes[titleShape]
	pixels := make([]byte, coverWidth*coverHeight)
	surface := vga.Surface{Pixels: pixels, Width: coverWidth, Right: coverWidth - 1, Bottom: coverHeight - 1}
	x := (coverWidth-logo.Width())/2 + logo.Left
	y := (coverHeight-logo.Height())/2 + logo.Up
	surface.DrawShape(x, y, logo)

	img := &image.Paletted{Pix: pixels, Stride: coverWidth, Rect: image.Rect(0, 0, coverWidth, coverHeight), Palette: palette}
	f, err := os.Create(filepath.Join(dir, "COVER.PNG"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
