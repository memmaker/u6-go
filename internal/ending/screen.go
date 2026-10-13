package ending

import "github.com/memmaker/u6-go/internal/vga"

// The drawing primitives of GRLIB and MCGADRV.BIN as END.EXE uses them. GRLIB is the same
// library as U.EXE's UL.LIB; these follow internal/intro's port of it.

const (
	screenWidth  = 320
	screenHeight = 200
	colours      = 256
	rgbBytes     = 3
	// coloursPerRetrace is how many DAC entries a palette loop writes per retrace.
	coloursPerRetrace = 16
)

// surface is a rectangle of a picture (tSurface).
type surface = vga.Surface

// newSurface allocates a 320x200 picture (allocSurface).
// C: seg_05CD:0000 (allocSurface)
func newSurface() surface {
	return surface{Pixels: make([]byte, screenWidth*screenHeight), Width: screenWidth,
		Right: screenWidth - 1, Bottom: screenHeight - 1}
}

func at(s surface, x, y int) int { return y*s.Width + x }

// fill paints the surface's rectangle.
// C: driver entry 12
func fill(s surface, colour int) {
	for y := s.Top; y <= s.Bottom; y++ {
		for x := s.Left; x <= s.Right; x++ {
			s.Pixels[at(s, x, y)] = byte(colour)
		}
	}
}

// copyRect copies src's rectangle to dst's, as much as fits in both, from their top left
// corners.
// C: driver entry 15
func copyRect(src, dst surface) {
	width := min(src.Right-src.Left, dst.Right-dst.Left) + 1
	height := min(src.Bottom-src.Top, dst.Bottom-dst.Top) + 1
	for row := range height {
		from := at(src, src.Left, src.Top+row)
		to := at(dst, dst.Left, dst.Top+row)
		copy(dst.Pixels[to:to+width], src.Pixels[from:from+width])
	}
}

// The DAC.

type rgb [rgbBytes]int

// setColor sets DAC colour index.
// C: setColor (PALET.ASM)
func (e *ending) setColor(index int, c rgb) {
	for i, v := range c {
		e.dac[index*rgbBytes+i] = byte(v)
	}
}

// getColor reads DAC colour index.
// C: getColor (PALET.ASM)
func (e *ending) getColor(index int) (c rgb) {
	for i := range c {
		c[i] = int(e.dac[index*rgbBytes+i])
	}
	return c
}

const (
	// lbmPaletteAt is where ENDPAL.LBM's CMAP colours start: the IFF and BMHD headers, then
	// the CMAP chunk's header.
	lbmPaletteAt = 48
	// lbmToDAC turns 8-bit colour components into the DAC's 6 bits.
	lbmToDAC = 2
)

// loadPalette sets the DAC to ENDPAL.LBM's colours, 16 per retrace.
// C: loadPalette (PALET.ASM)
func (e *ending) loadPalette() error {
	raw, err := e.readFile(paletteFile)
	if err != nil {
		return err
	}
	if len(raw) < lbmPaletteAt+colours*rgbBytes {
		return errShortPalette
	}
	for i := range colours * rgbBytes {
		e.dac[i] = raw[lbmPaletteAt+i] >> lbmToDAC
	}
	return nil
}

// savePalette reads the DAC into savedPalette; restorePalette writes it back.
// C: seg_00A9:0BBC (fadeIn), seg_00A9:0B88 (fadeOut)
func (e *ending) savePalette() {
	for i := range e.savedPalette {
		if i%coloursPerRetrace == 0 {
			e.waitRetrace()
		}
		e.savedPalette[i] = e.getColor(i)
	}
}

func (e *ending) restorePalette() {
	for i, c := range e.savedPalette {
		if i%coloursPerRetrace == 0 {
			e.waitRetrace()
		}
		e.setColor(i, c)
	}
}

// fader is one Bresenham fade of n values towards a target (fadeStep's static state).
type fader struct {
	running     bool
	err, dist   []int
	dir         []int
	maxDistance int
	steps       int
}

// step moves cur one step towards target; false when the fade is done.
// C: seg_06A2:000E (fadeStep)
func (f *fader) step(cur, target []int) bool {
	n := len(cur)
	if !f.running {
		f.err, f.dist, f.dir = make([]int, n), make([]int, n), make([]int, n)
		f.maxDistance = 0
		for i := range n {
			d := cur[i] - target[i]
			f.dir[i] = -1
			if d < 0 {
				d = -d
				f.dir[i] = 1
			}
			f.dist[i] = d
			f.maxDistance = max(f.maxDistance, d)
		}
		for i := range n {
			f.err[i] = f.maxDistance / 2
		}
		f.steps = f.maxDistance
		f.running = true
	}
	if f.steps == 0 {
		f.running = false
		return false
	}
	f.steps--
	for i := range n {
		if f.err[i] += f.dist[i]; f.err[i] > f.maxDistance {
			f.err[i] -= f.maxDistance
			cur[i] += f.dir[i]
		}
	}
	return true
}

// usedColours lists the colours in the surface's rectangle.
func usedColours(s surface) []int {
	var used [colours]bool
	for y := s.Top; y <= s.Bottom; y++ {
		for x := s.Left; x <= s.Right; x++ {
			used[s.Pixels[at(s, x, y)]] = true
		}
	}
	var list []int
	for c, ok := range used {
		if ok {
			list = append(list, c)
		}
	}
	return list
}

func (e *ending) fadeTo(list []int, from, to []int) {
	for e.fade.step(from, to) {
		e.waitRetrace()
		for i, c := range list {
			e.setColor(c, rgb{from[i*rgbBytes], from[i*rgbBytes+1], from[i*rgbBytes+2]})
		}
	}
}

// fadeIn sets the colours used in s to the colour of dst's top left pixel, copies s to dst
// and fades the colours back.
// C: seg_06A2:0157 (fadeInSurface)
func (e *ending) fadeIn(s, dst surface) {
	list := usedColours(s)
	start := e.getColor(int(dst.Pixels[at(dst, dst.Left, dst.Top)]))
	from, to := make([]int, len(list)*rgbBytes), make([]int, len(list)*rgbBytes)
	e.waitRetrace()
	for i, c := range list {
		c0 := e.getColor(c)
		copy(to[i*rgbBytes:], c0[:])
		e.setColor(c, start)
		copy(from[i*rgbBytes:], start[:])
	}
	copyRect(s, dst)
	e.fadeTo(list, from, to)
}

// fadeOut fades the colours used in s to colour.
// C: seg_06A2:0350 (fadeOutSurface)
func (e *ending) fadeOut(s surface, colour int) {
	list := usedColours(s)
	target := e.getColor(colour)
	from, to := make([]int, len(list)*rgbBytes), make([]int, len(list)*rgbBytes)
	e.waitRetrace()
	for i, c := range list {
		c0 := e.getColor(c)
		copy(from[i*rgbBytes:], c0[:])
		copy(to[i*rgbBytes:], target[:])
	}
	e.fadeTo(list, from, to)
}

// feedbackTaps are maximal length LFSR taps for registers of 2..32 bits.
var feedbackTaps = [...]uint32{3, 6, 12, 20, 48, 96, 184, 272, 576, 1280, 3232, 6912, 13568, 24576,
	46080, 73728, 132096, 466944, 589824, 1310720, 3145728, 4325376, 14155776, 18874368,
	59244544, 119537664, 150994944, 335544320, 847249408, 1207959552, 2734686208}

// dissolve copies src's rectangle to dst's pixel by pixel in the order of a linear feedback
// shift register, in about 7*speed+1 retraces.
// C: F_07286 (TRANS.ASM)
func (e *ending) dissolve(src, dst surface, speed int) {
	width, height := src.Right-src.Left+1, src.Bottom-src.Top+1
	register, xMask, xBits, bits := uint32(3), uint32(1), 1, 2
	for w := width >> 1; w != 0; w >>= 1 {
		register = register<<1 | 1
		xMask = xMask<<1 | 1
		xBits++
		bits++
	}
	for h := height >> 1; h != 0; h >>= 1 {
		register = register<<1 | 1
		bits++
	}
	count := uint16(width*height - 1)
	perWait := count / uint16(speed*7+1)
	left := perWait
	taps := feedbackTaps[bits-2]
	copyPixel := func(x, y int) {
		dst.Pixels[at(dst, dst.Left+x, dst.Top+y)] = src.Pixels[at(src, src.Left+x, src.Top+y)]
	}
	for {
		carry := register & 1
		register >>= 1
		if carry != 0 {
			register ^= taps
		}
		x, y := int(register&xMask), int(uint16(register>>xBits))
		if x >= width || y >= height {
			continue
		}
		copyPixel(x, y)
		if left--; left == 0 {
			e.waitRetrace()
			left = perWait
		}
		if count--; count == 0 {
			break
		}
	}
	copyPixel(0, 0)
}
