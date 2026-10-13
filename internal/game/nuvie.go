package game

import (
	"embed"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// The UI's art, from ScummVM's ultima.dat (devtools/create_ultima/files/ultima6,
// GPL; see nuvie/COPYING): Nuvie's dialog border, gump backgrounds and buttons, its scroll
// and sign, and its gump and sign fonts. The pictures are PNG copies of Nuvie's BMPs; the gump
// font is Nuvie's compiled-in font_gump_data in its doll colours.
//
//go:embed nuvie/*.png nuvie/*.bin
var nuvieArt embed.FS

// nuvieColourKey is the colour Nuvie's pictures leave transparent.
var nuvieColourKey = color.RGBA{R: 0, G: 0x70, B: 0xfc, A: 0xff}

// art is a picture of Nuvie's in the game's colours.
type art struct {
	width, height int
	pixels        []byte
	opaque        []bool
}

func (a *art) size() image.Point { return image.Pt(a.width, a.height) }

// art returns Nuvie's picture nuvie/name.png, mapped onto the game's palette once.
func (g *Game) art(name string) *art {
	if a := g.arts[name]; a != nil {
		return a
	}
	f, err := nuvieArt.Open("nuvie/" + name + ".png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	a := &art{width: b.Dx(), height: b.Dy(), pixels: make([]byte, b.Dx()*b.Dy()), opaque: make([]bool, b.Dx()*b.Dy())}
	nearest := map[color.RGBA]byte{}
	for y := range a.height {
		for x := range a.width {
			c := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
			if c == nuvieColourKey {
				continue
			}
			i, seen := nearest[c]
			if !seen {
				i = g.nearestColour(c)
				nearest[c] = i
			}
			a.pixels[y*a.width+x], a.opaque[y*a.width+x] = i, true
		}
	}
	if g.arts == nil {
		g.arts = map[string]*art{}
	}
	g.arts[name] = a
	return a
}

// nearestColour is the index of the game's palette closest to c.
func (g *Game) nearestColour(c color.RGBA) byte {
	const sixBitShift = 2 // 8-bit components to the DAC's 6 bits
	want := [3]int{int(c.R >> sixBitShift), int(c.G >> sixBitShift), int(c.B >> sixBitShift)}
	best, bestDistance := 0, -1
	for i := range len(g.basePalette) / 3 {
		distance := 0
		for k := range want {
			d := int(g.basePalette[i*3+k]) - want[k]
			distance += d * d
		}
		if bestDistance < 0 || distance < bestDistance {
			best, bestDistance = i, distance
		}
	}
	return byte(best)
}

// drawArt draws the part src of a with its corner at at, clipped to clip and the screen.
func (g *Game) drawArt(a *art, src image.Rectangle, at image.Point, clip image.Rectangle) {
	clip = clip.Intersect(image.Rect(0, 0, g.screenWidth, g.screenHeight))
	for y := src.Min.Y; y < src.Max.Y; y++ {
		for x := src.Min.X; x < src.Max.X; x++ {
			pt := at.Add(image.Pt(x-src.Min.X, y-src.Min.Y))
			if i := y*a.width + x; a.opaque[i] && pt.In(clip) {
				g.shown[pt.Y*g.screenWidth+pt.X] = a.pixels[i]
			}
		}
	}
}

// drawWhole draws all of a at at.
func (g *Game) drawWhole(a *art, at image.Point) {
	g.drawArt(a, image.Rectangle{Max: a.size()}, at, image.Rect(0, 0, g.screenWidth, g.screenHeight))
}

// Nuvie's dialog border (BorderU6_1..8): corners and edges going clockwise from the top
// left corner. The edges repeat along the sides.
const (
	borderTopLeft = iota
	borderTop
	borderTopRight
	borderRight
	borderBottomRight
	borderBottom
	borderBottomLeft
	borderLeft
)

// borderWidth is how thick the border is: the scroll's padding.
const borderWidth = scrollPadding

// borderArt is the border's piece.
func (g *Game) borderArt(piece int) *art {
	return g.art("border" + string(rune('1'+piece)))
}

// drawBorder draws Nuvie's border around r, as GUI_Dialog::Display does.
func (g *Game) drawBorder(r image.Rectangle) {
	put := func(piece int, at image.Point, clip image.Rectangle) {
		a := g.borderArt(piece)
		g.drawArt(a, image.Rectangle{Max: a.size()}, at, clip)
	}
	inner := r.Inset(borderWidth)
	top := image.Rect(inner.Min.X, r.Min.Y, inner.Max.X, inner.Min.Y)
	bottom := image.Rect(inner.Min.X, inner.Max.Y, inner.Max.X, r.Max.Y)
	for x := inner.Min.X; x < inner.Max.X; x += g.borderArt(borderTop).width {
		put(borderTop, image.Pt(x, r.Min.Y), top)
		put(borderBottom, image.Pt(x, inner.Max.Y), bottom)
	}
	left := image.Rect(r.Min.X, inner.Min.Y, inner.Min.X, inner.Max.Y)
	right := image.Rect(inner.Max.X, inner.Min.Y, r.Max.X, inner.Max.Y)
	for y := inner.Min.Y; y < inner.Max.Y; y += g.borderArt(borderLeft).height {
		put(borderLeft, image.Pt(r.Min.X, y), left)
		put(borderRight, image.Pt(inner.Max.X, y), right)
	}
	put(borderTopLeft, r.Min, r)
	put(borderTopRight, image.Pt(inner.Max.X, r.Min.Y), r)
	put(borderBottomRight, inner.Max, r)
	put(borderBottomLeft, image.Pt(r.Min.X, inner.Max.Y), r)
}

// drawNuvieScroll fills r with Nuvie's U6 scroll (U6_WOU_Scroll_bg, the conversation
// scroll of its original+ style): its ragged edges kept and its inside repeated.
func (g *Game) drawNuvieScroll(r image.Rectangle) {
	a := g.art("u6_wou_scroll_bg")
	const edge = scrollPadding
	from := func(p, lo, hi, size int) int {
		switch {
		case p-lo < edge:
			return p - lo
		case hi-1-p < edge:
			return size - (hi - p)
		}
		return edge + (p-lo-edge)%(size-2*edge)
	}
	clip := r.Intersect(image.Rect(0, 0, g.screenWidth, g.screenHeight))
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			if i := from(y, r.Min.Y, r.Max.Y, a.height)*a.width + from(x, r.Min.X, r.Max.X, a.width); a.opaque[i] {
				g.shown[y*g.screenWidth+x] = a.pixels[i]
			}
		}
	}
}

// nuvieFont is a font of Nuvie's: 16 by 16 characters of a picture, each with its width.
type nuvieFont struct {
	glyphs string // the picture
	widths string // nuvie/<widths>.bin
}

var (
	gumpFont = nuvieFont{"gump_font", "gump_font_widths"}
	signFont = nuvieFont{"sign_font", "sign_font_widths"}
)

// fontGrid is how many characters a font picture holds across and down.
const fontGrid = 16

func (f nuvieFont) width(c byte) int {
	widths, err := nuvieArt.ReadFile("nuvie/" + f.widths + ".bin")
	if err != nil {
		panic(err)
	}
	return int(widths[c])
}

// textWidth is how wide s is in f.
func textWidth(f nuvieFont, s string) int {
	w := 0
	for i := range len(s) {
		w += f.width(s[i])
	}
	return w
}

// drawNuvieText writes s in f with its corner at at.
func (g *Game) drawNuvieText(f nuvieFont, s string, at image.Point) {
	a := g.art(f.glyphs)
	cellWidth, cellHeight := a.width/fontGrid, a.height/fontGrid
	for i := range len(s) {
		c := int(s[i])
		src := image.Rect(0, 0, cellWidth, cellHeight).Add(image.Pt(c%fontGrid*cellWidth, c/fontGrid*cellHeight))
		g.drawArt(a, src, at, image.Rect(0, 0, g.screenWidth, g.screenHeight))
		at.X += f.width(s[i])
	}
}

// signTextLimit is the longest text Nuvie shows on its sign; longer ones go to the scroll.
const signTextLimit = 20

// signText is a sign's text without the runic font's marks and the line breaks.
func signText(text string) string {
	return strings.TrimSpace(strings.NewReplacer("<", "", ">", "", "\n", " ").Replace(text))
}

// showsOnSign says that object n's text is shown on Nuvie's sign.
func (g *Game) showsOnSign(n int, text string) bool {
	return g.object(n).Type() == ObjSign && len(signText(text)) <= signTextLimit
}

// drawSign draws Nuvie's sign in the middle of the screen with text on it (SignViewGump).
func (g *Game) drawSign(text string) {
	a := g.art("sign_bg")
	at := image.Pt((g.screenWidth-a.width)/2, (g.screenHeight-a.height)/2)
	g.drawWhole(a, at)
	glyphHeight := g.art(signFont.glyphs).height / fontGrid
	g.drawNuvieText(signFont, text, at.Add(image.Pt((a.width-textWidth(signFont, text))/2, (a.height-glyphHeight)/2)))
}
