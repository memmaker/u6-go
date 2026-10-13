package intro

import (
	"time"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
)

// The drawing primitives of MCGADRV.BIN (U.EXE's own VGA driver, entries GR(n)) and of UL.LIB,
// on 320x200 paletted pictures.

const (
	screenWidth  = host.ScreenWidth
	screenHeight = host.ScreenHeight
	colours      = 256
	rgbBytes     = 3
	// coloursPerRetrace is how many DAC entries the palette loops write between two waits
	// for the vertical retrace.
	coloursPerRetrace = 16
)

// surface is a rectangle of a picture (tSurface): pixels is the picture, width its row
// length; left..right and top..bottom are inclusive picture coordinates.
type surface struct {
	pixels                   []byte
	width                    int
	right, left, top, bottom int
}

// newPicture allocates a picture for the surface's rectangle; the rectangle is kept and
// addresses the new picture from its row and column 0.
// C: seg_1277:000C (SURF_alloc)
func newPicture(s *surface) {
	s.width = s.right - s.left + 1
	s.pixels = make([]byte, s.width*(s.bottom-s.top+1))
}

// samePicture makes every surface of dst draw on src's picture, keeping their rectangles.
// C: seg_12EA:000A (REC_copyto)
func samePicture(src surface, dst ...*surface) {
	for _, d := range dst {
		d.pixels, d.width = src.pixels, src.width
	}
}

func (s *surface) at(x, y int) int { return y*s.width + x }

// inPicture reports whether (x, y) lies in the picture (the original wrote past its edges
// in a few places; the port leaves those pixels out).
func (s *surface) inPicture(x, y int) bool {
	return x >= 0 && x < s.width && y >= 0 && s.at(x, y) < len(s.pixels)
}

// fill paints the surface's rectangle with a colour.
// C: MCGADRV GR(0x0C)
func fill(s surface, colour int) {
	for y := s.top; y <= s.bottom; y++ {
		for x := s.left; x <= s.right; x++ {
			if s.inPicture(x, y) {
				s.pixels[s.at(x, y)] = byte(colour)
			}
		}
	}
}

// putPixel sets one pixel inside the rectangle.
// C: MCGADRV GR(0x18)
func putPixel(s surface, x, y, colour int) {
	if x >= s.left && x <= s.right && y >= s.top && y <= s.bottom {
		s.pixels[s.at(x, y)] = byte(colour)
	}
}

// getPixel reads one pixel inside the rectangle, -1 outside.
// C: seg_12B7:022F (surf_getpixel)
func getPixel(s surface, x, y int) int {
	if x < s.left || x > s.right || y < s.top || y > s.bottom {
		return -1
	}
	return int(s.pixels[s.at(x, y)])
}

// shapes is a loaded shape library (.SHP, INTRO.PTR).
type shapes []*vga.Shape

// draw draws shape index of the library with its hot spot at (x, y), clipped to the surface.
// C: MCGADRV GR(0x03)
func (lib shapes) draw(s surface, x, y, index int) {
	if index < 0 || index >= len(lib) {
		return
	}
	vga.Surface{Pixels: s.pixels, Width: s.width, Right: s.right, Left: s.left, Top: s.top,
		Bottom: s.bottom}.DrawShape(x, y, lib[index])
}

// copyRect copies src's rectangle to dst's, as much as fits in both, from their top left
// corners. Copying to the screen is a page flip: the host shows the frame.
// C: MCGADRV GR(0x0F)
func (u *intro) copyRect(src, dst surface) {
	width := min(src.right-src.left, dst.right-dst.left) + 1
	height := min(src.bottom-src.top, dst.bottom-dst.top) + 1
	for row := 0; row < height; row++ {
		from := src.at(src.left, src.top+row)
		to := dst.at(dst.left, dst.top+row)
		copy(dst.pixels[to:to+width], src.pixels[from:from+width])
	}
	if u.isScreen(dst) {
		u.present()
	}
}

func (u *intro) isScreen(s surface) bool {
	return len(s.pixels) > 0 && &s.pixels[0] == &u.screen.pixels[0]
}

// Frame pacing. The original drew as fast as the machine allowed and waited for the
// vertical retrace in some places. Here every wait for the retrace is one host tick; a page
// flip waits for the next tick when it is the second in the same tick, and a keyboard poll
// waits for it when nothing else waited since the previous poll, so every loop runs at most
// once per tick.

// yield ends the host tick.
func (u *intro) yield() {
	host.Yield()
	u.synced = true
}

// wait lets d pass.
func (u *intro) wait(d time.Duration) {
	host.Wait(d)
	u.synced = true
}

// waitRetrace waits for the vertical retrace.
// C: seg_1368:0015 (wait_vretrace_start), seg_1368:0002 (wait_vretrace)
func (u *intro) waitRetrace() { u.yield() }

// present ends a frame that put a new picture on the screen.
func (u *intro) present() {
	if host.Ticks == u.presentTick {
		u.yield()
	}
	u.presentTick = host.Ticks
	u.synced = true
}

// idle ends the frame of a loop that polls the keyboard, unless something else waited.
func (u *intro) idle() {
	if !u.synced {
		u.yield()
	}
	u.synced = false
}

// The VGA DAC.

type rgb [rgbBytes]int

// setDAC sets colour index of the DAC.
// C: seg_1475:0099 (set_dac)
func (u *intro) setDAC(index int, c rgb) {
	index &= colours - 1
	for i, v := range c {
		u.dac[index*rgbBytes+i] = byte(v)
	}
}

// getDAC reads colour index of the DAC.
// C: seg_1475:00B8 (get_dac)
func (u *intro) getDAC(index int) (c rgb) {
	index &= colours - 1
	for i := range c {
		c[i] = int(u.dac[index*rgbBytes+i])
	}
	return c
}

// setPalette writes a whole palette, waiting for the retrace every 16 colours.
// C: seg_00E1:04F9 (pal_fade_4A40), seg_00E1:052D (pal_fade_505F)
func (u *intro) setPalette(p *[colours]rgb) {
	for i, c := range p {
		if i%coloursPerRetrace == 0 {
			u.waitRetrace()
		}
		u.setDAC(i, c)
	}
}

// savePalette reads the whole DAC into p.
// C: seg_00E1:0561 (pal_set_505F)
func (u *intro) savePalette(p *[colours]rgb) {
	for i := range p {
		if i%coloursPerRetrace == 0 {
			u.waitRetrace()
		}
		p[i] = u.getDAC(i)
	}
}

const (
	paletteFile = "PALETTES.INT"
	// packedPaletteColours colours of 4 six-bit components packed in 3 bytes each
	packedPaletteColours = colours * rgbBytes / 4
	packedPaletteBytes   = packedPaletteColours * rgbBytes
	sixBits              = 6
	sixBitMask           = 1<<sixBits - 1
	componentsPerPack    = 4
)

// loadPalette sets the DAC to palette n of PALETTES.INT: 768 six-bit components packed four
// to three bytes, the first in the low bits.
// C: seg_0BC1:03D2 (LoadPalette)
func (u *intro) loadPalette(n int) {
	raw, err := u.readFile(paletteFile)
	if err != nil || len(raw) < (n+1)*packedPaletteBytes {
		return
	}
	raw = raw[n*packedPaletteBytes:]
	var components [colours * rgbBytes]int
	for i := range packedPaletteColours {
		bits := int(raw[i*3]) | int(raw[i*3+1])<<8 | int(raw[i*3+2])<<16
		for k := range componentsPerPack {
			components[i*componentsPerPack+k] = bits >> (k * sixBits) & sixBitMask
		}
	}
	for i := range colours {
		if i%coloursPerRetrace == 0 {
			u.waitRetrace()
		}
		u.setDAC(i, rgb{components[i*3], components[i*3+1], components[i*3+2]})
	}
}

// fader is one Bresenham fade of n values towards a target (fade_step's static state).
type fader struct {
	running     bool
	err, dist   []int
	dir         []int
	maxDistance int
	steps       int
}

// step moves cur one step towards target; false when the fade is done (and the next call
// starts a new one).
// C: seg_136D:0004 (fade_step)
func (f *fader) step(cur, target []int) bool {
	n := len(cur)
	if !f.running {
		f.err, f.dist, f.dir = make([]int, n), make([]int, n), make([]int, n)
		f.maxDistance = 0
		for i := range n {
			d := cur[i] - target[i]
			if d < 0 {
				d = -d
				f.dir[i] = 1
			} else {
				f.dir[i] = -1
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

// usedColours lists the colours that appear in the surface's rectangle.
func usedColours(s surface) []int {
	var used [colours]bool
	for y := s.top; y <= s.bottom; y++ {
		for x := s.left; x <= s.right; x++ {
			used[s.pixels[s.at(x, y)]] = true
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

// fadeTo fades the colours in list from their values in from to those in to.
func (u *intro) fadeTo(list []int, from, to []int) {
	for u.fade.step(from, to) {
		u.waitRetrace()
		for i, c := range list {
			u.setDAC(c, rgb{from[i*3], from[i*3+1], from[i*3+2]})
		}
	}
}

// fadeIn sets the colours used in s to the colour of bg's top left pixel, copies s to bg and
// fades the colours back to their values.
// C: seg_136D:014D (fade_in)
func (u *intro) fadeIn(s, bg surface) {
	list := usedColours(s)
	start := u.getDAC(getPixel(bg, bg.left, bg.top))
	from, to := make([]int, len(list)*3), make([]int, len(list)*3)
	u.waitRetrace()
	for i, c := range list {
		c0 := u.getDAC(c)
		copy(to[i*3:], c0[:])
		u.setDAC(c, start)
		copy(from[i*3:], start[:])
	}
	u.copyRect(s, bg)
	u.fadeTo(list, from, to)
}

// fadeOut fades the colours used in s to the colour colour.
// C: seg_136D:0346 (fade_out)
func (u *intro) fadeOut(s surface, colour int) {
	list := usedColours(s)
	target := u.getDAC(colour)
	from, to := make([]int, len(list)*3), make([]int, len(list)*3)
	u.waitRetrace()
	for i, c := range list {
		c0 := u.getDAC(c)
		copy(from[i*3:], c0[:])
		copy(to[i*3:], target[:])
	}
	u.fadeTo(list, from, to)
}

// feedbackTaps are maximal length LFSR taps for registers of 2..32 bits.
var feedbackTaps = [...]uint32{3, 6, 12, 20, 48, 96, 184, 272, 576, 1280, 3232, 6912, 13568, 24576,
	46080, 73728, 132096, 466944, 589824, 1310720, 3145728, 4325376, 14155776, 18874368,
	59244544, 119537664, 150994944, 335544320, 847249408, 1207959552, 2734686208}

// dissolve copies src's rectangle to dst's pixel by pixel in the pseudo random order of a
// linear feedback shift register, in about 7*speed+1 frames (speed 0: at once).
// C: seg_13F2:000C (dissolve)
func (u *intro) dissolve(src, dst surface, speed int) {
	width, height := src.right-src.left+1, src.bottom-src.top+1
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
		dst.pixels[dst.at(dst.left+x, dst.top+y)] = src.pixels[src.at(src.left+x, src.top+y)]
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
			u.waitRetrace()
			left = perWait
		}
		if count--; count == 0 {
			break
		}
	}
	copyPixel(0, 0)
}

// The timer interrupt (int 1Ch, 18.2 times a second) with its callback.

const (
	pitHz      = 1193182
	pitDivisor = 65536
	timerHz    = float64(pitHz) / pitDivisor
	// timerTick is the length of a timer tick.
	timerTick = time.Second * pitDivisor / pitHz
)

var (
	// timerISR runs on every timer tick while set (the hooked int 1Ch).
	timerISR   func()
	timerDue   float64
	timerAdded bool
)

// installTimer hooks the timer interrupt; the callback starts empty the first time.
// C: seg_1482:0055 (timer_install)
func (u *intro) installTimer() {
	if !u.timerInstalled {
		u.timerInstalled = true
		u.timerCallback = nil
		timerISR = func() {
			if u.timerCallback != nil {
				u.timerCallback()
			}
		}
	}
	if !timerAdded {
		timerAdded = true
		host.AddTimer(func() {
			timerDue += timerHz / host.TicksPerSecond
			for ; timerDue >= 1; timerDue-- {
				if timerISR != nil {
					timerISR()
				}
			}
		})
	}
}

// removeTimer unhooks the timer interrupt.
// C: seg_1482:009C (timer_remove)
func (u *intro) removeTimer() {
	u.timerInstalled = false
	timerISR = nil
}

// setTimerCallback makes f run on every timer tick (nil: nothing).
// C: seg_1482:000F (timer_set_callback), seg_1482:001F (timer_clear_callback)
func (u *intro) setTimerCallback(f func()) { u.timerCallback = f }

// calibratedLoops is Timer.loops, the busy-wait iterations per timer tick timer_calibrate
// measured. It sets how fast the intros scroll (randomBetween); the port takes the value of
// a fast machine, the most randomBetween counts.
const calibratedLoops = 20000

// calibrateTimer measured the machine's speed during the third timer tick.
// C: seg_1482:00BE (timer_calibrate)
func (u *intro) calibrateTimer() {
	const ticksWaited = 3
	u.installTimer()
	u.wait(ticksWaited * timerTick)
	u.removeTimer()
}

const loopsPerDelayUnit = 400

// busyDelay waits n 400ths of a timer tick (a busy loop of n*loops/400 iterations).
// C: seg_06E9:2B34 (INTRO_delay)
func (u *intro) busyDelay(n int) { u.wait(time.Duration(n) * timerTick / loopsPerDelayUnit) }

// delayMilliseconds waits ms milliseconds in whole timer ticks (55 ms).
// C: seg_1482:0113 (delay_ms)
func (u *intro) delayMilliseconds(ms int) {
	const msPerTick = 55
	u.wait(time.Duration(ms/msPerTick) * timerTick)
}
