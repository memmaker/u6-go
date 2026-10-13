package ending

// The star field of the vortex: 64 stars fly out from its centre on every timer tick,
// drawn straight on the screen where it is black (the swirl's background).

const (
	starCount    = 64
	fieldWidth   = 113 // the vortex, in pixels from (vortexLeft, vortexTop)
	fieldHeight  = 94
	fieldCentreX = 56
	fieldCentreY = 47
	// speedShift: a star moves 1 + (distance from the centre)/16 pixels per tick
	speedShift = 4
	// starColours are the VGA star colours: 224..239
	firstStarColour = 224
	starColours     = 16
	// lfsrTaps of the 16-bit register starsMCGA picks new star positions with
	lfsrTaps     = 46080
	lowSevenBits = 127
	// cycled colours: 16..31 rotate up one every second tick while scene < cycleScenes
	firstCycled = 16
	lastCycled  = 31
	cycleScenes = 9
)

// starField is the stars' positions (in the vortex) and colours, and the shift registers
// that place new ones.
type starField struct {
	x, y, colour [starCount]uint8 // g_starX, g_starY, g_starC
	randX, randY uint16           // g_1833, g_1835
	cycleTick    bool             // g_palTick
}

// init places the stars at random.
// C: seg_00A9:0A90 (initStars)
func (f *starField) init() {
	for i := range starCount {
		f.x[i] = uint8(random() % fieldWidth)
		f.y[i] = uint8(random() % fieldHeight)
		f.colour[i] = uint8(random()%starColours + firstStarColour)
	}
	for f.randX == 0 {
		f.randX = uint16(random())
	}
	for f.randY == 0 {
		f.randY = uint16(random())
	}
}

// next steps a 16-bit shift register and takes a number below limit from it.
// fix: the original addressed g_1833/g_1835 through CS, so it shifted bytes of its own code
// segment instead of the values initStars seeded; the port uses the seeded registers.
func next(register *uint16, limit int) uint8 {
	carry := *register & 1
	*register >>= 1
	if carry != 0 {
		*register ^= lfsrTaps
	}
	n := int(*register & lowSevenBits)
	if n >= limit {
		n %= limit
	}
	return uint8(n)
}

// starsTick is the timer handler: it cycles colours 16..31 every second tick (until scene
// 9) and moves the stars.
// C: seg_02AE:0001 (starsTick)
func (e *ending) starsTick() {
	f := &e.stars
	if f.cycleTick = !f.cycleTick; f.cycleTick && e.scene < cycleScenes {
		saved := e.getColor(lastCycled)
		for i := lastCycled - 1; i >= firstCycled; i-- {
			e.setColor(i+1, e.getColor(i))
		}
		e.setColor(firstCycled, saved)
	}
	e.moveStars()
}

// moveStars erases each star (if it is still there), moves it away from the centre, puts a
// new one in when it left the field, and draws it where the screen is black.
// C: starsMCGA (STARFLD.ASM)
func (e *ending) moveStars() {
	f := &e.stars
	pixel := func(i int) *byte {
		return &e.screen.Pixels[at(e.screen, vortexLeft+int(f.x[i]), vortexTop+int(f.y[i]))]
	}
	for i := starCount - 1; i >= 0; i-- {
		if p := pixel(i); *p == f.colour[i] {
			*p = 0
		}
		deltaX, deltaY := int(int8(f.x[i]-fieldCentreX)), int(int8(f.y[i]-fieldCentreY))
		f.y[i] = uint8(int(f.y[i]) + sign(deltaY)*(abs(deltaY)>>speedShift+1))
		f.x[i] = uint8(int(f.x[i]) + sign(deltaX)*(abs(deltaX)>>speedShift+1))
		if f.y[i] >= fieldHeight || f.x[i] >= fieldWidth {
			f.x[i] = next(&f.randX, fieldWidth)
			f.y[i] = next(&f.randY, fieldHeight)
		}
		if p := pixel(i); *p == 0 {
			*p = f.colour[i]
		}
	}
}

// sign is 1 for n >= 0 (a star on the centre line moves right or down), -1 below.
func sign(n int) int {
	if n < 0 {
		return -1
	}
	return 1
}
