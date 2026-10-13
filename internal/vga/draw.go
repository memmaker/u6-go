package vga

// target is the picture PutPixel and FillRect draw on, and its width.
func (d *Driver) target() ([]byte, int) {
	if d.Offscreen {
		return d.BackBuffer[:], BackBufferSize
	}
	return d.Screen[:], ScreenWidth
}

// plot sets or (in XorMode) flips one pixel; pixels outside the picture are dropped.
// fix: the original wrote past the picture.
func (d *Driver) plot(pixels []byte, width, x, y int) {
	if x < 0 || x >= width || y < 0 {
		return
	}
	i := y*width + x
	if i >= len(pixels) {
		return
	}
	if d.XorMode {
		pixels[i] ^= d.color
	} else {
		pixels[i] = d.color
	}
}

// SetColor sets the colour of PutPixel and FillRect.
//
// GR_06 (driver offset 6).
// C: U6MCGA.DRV:2A72
func (d *Driver) SetColor(color byte) { d.color = color }

// PutPixel draws one pixel in the current colour on the screen or, with Offscreen, the back
// buffer.
//
// GR_09 (driver offset 9).
// C: U6MCGA.DRV:2A82
func (d *Driver) PutPixel(x, y int) {
	pixels, width := d.target()
	d.plot(pixels, width, x, y)
}

// FillRect fills the rectangle between two corners (inclusive, in either order) in the
// current colour on the screen or, with Offscreen, the back buffer.
//
// GR_12 (driver offset 18).
// C: U6MCGA.DRV:2ADF
func (d *Driver) FillRect(x0, y0, x1, y1 int) {
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	pixels, width := d.target()
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			d.plot(pixels, width, x, y)
		}
	}
}

// ScrollRectVertically moves the screen's pixels inside the rectangle (inclusive corners, in
// either order) up by -dy rows when dy is negative; the last -dy rows keep their pixels.
// The game only scrolls its text windows up by one 8 pixel line.
//
// GR_15 (driver offset 21).
// fix: for dy > 0 the original copied from a wrong row; here it moves the pixels down.
// C: U6MCGA.DRV:2B8D
func (d *Driver) ScrollRectVertically(x0, y0, x1, y1, dy int) {
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	copyRow := func(to, from int) {
		copy(d.Screen[to*ScreenWidth+x0:to*ScreenWidth+x1+1], d.Screen[from*ScreenWidth+x0:])
	}
	if dy < 0 {
		for y := y0; y <= y1+dy; y++ {
			copyRow(y, y-dy)
		}
	} else {
		for y := y1; y >= y0+dy; y-- {
			copyRow(y, y-dy)
		}
	}
}

// DrawPixmap draws a picture on the screen with its top left corner at (x, y).
//
// GR_18 (driver offset 24).
// C: U6MCGA.DRV:21C0
func (d *Driver) DrawPixmap(p *Pixmap, x, y int) {
	for row := 0; row < p.Height; row++ {
		at := (y+row)*ScreenWidth + x
		copy(d.Screen[at:at+p.Width], p.Pixels[row*p.Width:])
	}
}

// DrawChar draws character ch of the font on the screen with its top left corner at (x, y),
// set bits in TextForeground and clear bits in TextBackground.
//
// GR_1E (driver offset 30).
// C: U6MCGA.DRV:2A10
func (d *Driver) DrawChar(ch byte, x, y int) {
	for row, bits := range d.Font[ch] {
		for col := 0; col < charSize; col++ {
			c := d.TextBackground
			if bits&(highBit>>col) != 0 {
				c = d.TextForeground
			}
			d.Screen[(y+row)*ScreenWidth+x+col] = c
		}
	}
}

const highBit = 128

// CopyBackBufferToScreen copies the back buffer's rectangle between (x0, y0) and (x1, y1)
// (inclusive) to the screen at (toX, toY). The copy works on pixel pairs: x0 and toX are
// rounded down and x1 up to whole pairs.
//
// GR_45 (driver offset 69).
// C: U6MCGA.DRV:290E
func (d *Driver) CopyBackBufferToScreen(x0, y0, x1, y1, toX, toY int) {
	x0 &^= 1
	x1 |= 1
	toX &^= 1
	width := x1 - x0 + 1
	for row := 0; row <= y1-y0; row++ {
		from := (y0+row)*BackBufferSize + x0
		to := (toY+row)*ScreenWidth + toX
		copy(d.Screen[to:to+width], d.BackBuffer[from:from+width])
	}
}

// FadeIn copies the back buffer's rectangle between (x0, y0) and (x1, y1) (inclusive) to the
// same place on the screen pixel by pixel in a scattered order, so the new picture dissolves
// in. The order comes from a maximal-length linear feedback shift register, which visits every
// number from 1 to 2^n-1 once; numbers past the rectangle are skipped and pixel 0 comes last.
//
// GR_48 (driver offset 72).
// C: U6MCGA.DRV:29A1
func (d *Driver) FadeIn(x0, y0, x1, y1 int) {
	width, height := x1-x0+1, y1-y0+1
	bits := 0
	for n := width*height - 1; n != 0; n >>= 1 {
		bits++
	}
	taps := feedbackTaps[bits-minFeedbackBits]
	copyPixel := func(i int) {
		x, y := x0+i%width, y0+i/width
		d.Screen[y*ScreenWidth+x] = d.BackBuffer[y*BackBufferSize+x]
	}
	for state, step := 1, 0; ; step++ {
		if state/width < height {
			copyPixel(state)
		}
		if state&1 != 0 {
			state = state>>1 ^ taps
		} else {
			state >>= 1
		}
		if state == 1 {
			break
		}
		if step%fadePixelsPerPresent == 0 {
			d.present()
		}
	}
	copyPixel(0)
}

// fadePixelsPerPresent paces FadeIn when Present is set.
const fadePixelsPerPresent = 1024

// feedbackTaps are the shift register taps for 2 to 17 bit states.
const minFeedbackBits = 2

var feedbackTaps = [...]int{3, 6, 12, 20, 48, 96, 184, 272, 576, 1280, 3232, 6912, 13568, 24576, 46080, 35669}

// Vortex swirls the map view: eight spiral arms turn around the view's centre, each dragging
// the pixels it passes along, in 22 rounds of shrinking reach. Pixels near the edge are taken
// from the back buffer.
//
// GR_4E (driver offset 78).
// C: U6MCGA.DRV:2D80
func (d *Driver) Vortex() {
	const (
		centre      = 88  // of the 160x160 map view, in screen and back buffer pixels
		startReach  = 88  // spiral length in the first round
		reachStep   = 4   // how much shorter each round's spiral is
		maxReach    = 76  // the swirl stays within this distance of the centre
		turns       = 12  // passes per round
		turnStep    = 8   // angle the arms turn per pass
		trail       = 8   // steps a pixel is carried along an arm
		viewStart   = 16  // the screen part of the view; outside it, pixels come from the back buffer
		viewEnd     = 160 //
		unitShift   = 6   // the triangle wave is 64 times the radius
		quarterTurn = 128 // of the triangle wave's 512 step period
	)
	angle := 0
	for reach := startReach; reach > 0; reach -= reachStep {
		half := min(reach, maxReach)
		for range turns {
			var arms [len(armAngles)]int
			for i := range arms {
				arms[i] = armAngles[i] - angle
			}
			// where each arm put its pixels in the last steps, as screen indexes (-1: nowhere)
			var placed [trail][len(armAngles)]int
			for i := range placed {
				for j := range placed[i] {
					placed[i][j] = -1
				}
			}
			newest, oldest := 0, 1
			for radius := range half {
				for arm := len(arms) - 1; arm >= 0; arm-- {
					a := arms[arm]
					arms[arm]++
					y := triangleWave(a-quarterTurn)*radius>>unitShift + centre
					x := triangleWave(a)*radius>>unitShift + centre
					var pair [2]byte
					if x >= centre-half && x < centre+half && y >= centre-half && y < centre+half {
						at := y*ScreenWidth + x&^1
						placed[newest][arm] = at
						if x >= viewStart && x < viewEnd && y >= viewStart && y < viewEnd {
							copy(pair[:], d.Screen[at:])
						} else {
							copy(pair[:], d.BackBuffer[y*BackBufferSize+x:])
						}
					}
					if to := placed[oldest][arm]; to != -1 {
						copy(d.Screen[to:to+2], pair[:])
					}
				}
				newest = (newest + 1) % trail
				oldest = (oldest + 1) % trail
			}
			angle += turnStep
			d.present()
		}
	}
}

// armAngles are the starting angles of the vortex arms (512 steps per turn).
var armAngles = [...]int{0, 61, 129, 191, 260, 322, 382, 451}

// triangleWave goes 128, 0, -128, 0 at a = 0, 128, 256, 384, repeating every 512 steps.
func triangleWave(a int) int {
	a &= 511
	if a >= 256 {
		return a - 384
	}
	return 128 - a
}
