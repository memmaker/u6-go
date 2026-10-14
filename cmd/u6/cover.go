package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/memmaker/u6-go/internal/vga"
)

const (
	coverWidth  = 320
	coverHeight = 200
	// the title logo and its tagline ("The False Prophet"): TITLES.SHP's first two shapes,
	// placed as the main menu places them, on its palette (PALETTES.INT's first)
	titleLibrary  = "TITLES.SHP"
	titleShape    = 0
	subtitleShape = 1
	titleX        = 19 // the menu's hot spots of the two (internal/intro)
	titleY        = 0
	subtitleX     = 59
	subtitleY     = 47
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
// Ultima VI title logo with its tagline, centred on black.
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
		level := func(c int) uint8 { return uint8(components[i*3+c] * math.MaxUint8 / vgaMaxLevel) }
		palette[i] = color.RGBA{level(0), level(1), level(2), math.MaxUint8}
	}

	// the two as the menu shows them, then that picture's drawn part centred
	menu := make([]byte, coverWidth*coverHeight)
	surface := vga.Surface{Pixels: menu, Width: coverWidth, Right: coverWidth - 1, Bottom: coverHeight - 1}
	surface.DrawShape(titleX, titleY, shapes[titleShape])
	if subtitleShape < len(shapes) {
		surface.DrawShape(subtitleX, subtitleY, shapes[subtitleShape])
	}
	drawn := image.Rectangle{}
	for at, c := range menu {
		if c != 0 {
			drawn = drawn.Union(image.Rect(at%coverWidth, at/coverWidth, at%coverWidth+1, at/coverWidth+1))
		}
	}
	pixels := make([]byte, coverWidth*coverHeight)
	dx, dy := (coverWidth-drawn.Dx())/2-drawn.Min.X, (coverHeight-drawn.Dy())/2-drawn.Min.Y
	for y := drawn.Min.Y; y < drawn.Max.Y; y++ {
		copy(pixels[(y+dy)*coverWidth+drawn.Min.X+dx:], menu[y*coverWidth+drawn.Min.X:y*coverWidth+drawn.Max.X])
	}

	img := &image.Paletted{Pix: pixels, Stride: coverWidth, Rect: image.Rect(0, 0, coverWidth, coverHeight), Palette: palette}
	f, err := os.Create(filepath.Join(dir, "COVER.PNG"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
