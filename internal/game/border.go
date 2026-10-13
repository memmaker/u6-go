package game

import (
	"embed"
	"fmt"
	"image"
	"image/png"

	"github.com/memmaker/u6-go/internal/vga"
)

// Nuvie's new style border (ScummVM's ultima.dat, ultima6/BorderU6_1..8.bmp, GPL; see
// nuvie/COPYING): corners and edges going clockwise from the top left corner. The edges
// repeat along the sides.
//
//go:embed nuvie/*.png
var nuvieArt embed.FS

const (
	borderTopLeft = iota
	borderTop
	borderTopRight
	borderRight
	borderBottomRight
	borderBottom
	borderBottomLeft
	borderLeft
	borderPieces
)

// borderWidth is how thick the border is: the scroll's padding.
const borderWidth = scrollPadding

// loadBorder maps the border's pieces onto the nearest colours of the game's palette.
func (g *Game) loadBorder() {
	for i := range g.border {
		f, err := nuvieArt.Open(fmt.Sprintf("nuvie/border%d.png", i+1))
		if err != nil {
			panic(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			panic(err)
		}
		b := img.Bounds()
		piece := &vga.Pixmap{Width: b.Dx(), Height: b.Dy(), Pixels: make([]byte, b.Dx()*b.Dy())}
		for y := range b.Dy() {
			for x := range b.Dx() {
				r, gr, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				piece.Pixels[y*b.Dx()+x] = g.nearestColour(r, gr, bl)
			}
		}
		g.border[i] = piece
	}
}

// nearestColour is the palette index closest to the 16-bit colour r, gr, bl.
func (g *Game) nearestColour(r, gr, bl uint32) byte {
	const sixBitShift = 10 // 16-bit components to the DAC's 6 bits
	want := [3]int{int(r >> sixBitShift), int(gr >> sixBitShift), int(bl >> sixBitShift)}
	best, bestDistance := 0, -1
	for i := range len(g.basePalette) / 3 {
		distance := 0
		for c := range want {
			d := int(g.basePalette[i*3+c]) - want[c]
			distance += d * d
		}
		if bestDistance < 0 || distance < bestDistance {
			best, bestDistance = i, distance
		}
	}
	return byte(best)
}

// drawBorder draws Nuvie's border around r, as GUI_Dialog::Display does.
func (g *Game) drawBorder(r image.Rectangle) {
	if g.border[0] == nil {
		g.loadBorder()
	}
	put := func(piece int, at image.Point, clip image.Rectangle) {
		p := g.border[piece]
		for y := range p.Height {
			for x := range p.Width {
				pt := at.Add(image.Pt(x, y))
				if pt.In(clip) && pt.In(image.Rect(0, 0, g.screenWidth, g.screenHeight)) {
					g.shown[pt.Y*g.screenWidth+pt.X] = p.Pixels[y*p.Width+x]
				}
			}
		}
	}
	inner := r.Inset(borderWidth)
	top := image.Rect(inner.Min.X, r.Min.Y, inner.Max.X, inner.Min.Y)
	bottom := image.Rect(inner.Min.X, inner.Max.Y, inner.Max.X, r.Max.Y)
	for x := inner.Min.X; x < inner.Max.X; x += g.border[borderTop].Width {
		put(borderTop, image.Pt(x, r.Min.Y), top)
		put(borderBottom, image.Pt(x, inner.Max.Y), bottom)
	}
	left := image.Rect(r.Min.X, inner.Min.Y, inner.Min.X, inner.Max.Y)
	right := image.Rect(inner.Max.X, inner.Min.Y, r.Max.X, inner.Max.Y)
	for y := inner.Min.Y; y < inner.Max.Y; y += g.border[borderLeft].Height {
		put(borderLeft, image.Pt(r.Min.X, y), left)
		put(borderRight, image.Pt(inner.Max.X, y), right)
	}
	put(borderTopLeft, r.Min, r)
	put(borderTopRight, image.Pt(inner.Max.X, r.Min.Y), r)
	put(borderBottomRight, inner.Max, r)
	put(borderBottomLeft, image.Pt(r.Min.X, inner.Max.Y), r)
}
