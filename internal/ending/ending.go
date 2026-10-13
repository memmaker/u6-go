// Package ending is END.EXE: the endgame sequence. Lord British and Nystul step out of a
// red moongate, the Codex appears in the vortex through the concave lens, King Draxinusom
// arrives through a second gate, the two lenses together open the book, and the player is
// congratulated with the time the quest took.
//
// Only the VGA (MCGA) path of the original is ported; see README.md for the deviations.
package ending

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/memmaker/u6-go/internal/data"
	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
	"github.com/memmaker/u6-go/internal/vga"
)

// Args are END.EXE's four command line arguments as GAME.EXE passes them (execl("end.exe",
// param_str, y_str, m_str, d_str)).
type Args struct {
	// Mouse: param_str's 'M' (a mouse button counts as a key press) or 'K' (keyboard only).
	Mouse bool
	// Music: param_str's sound driver letter is not 'N'. The video letter is ignored: the
	// port is VGA only. CONFIG.U6, which END.EXE read first, is not needed: GAME.EXE's
	// param_str always names video, input and sound, overriding it.
	Music bool
	// Years, Months and Days the quest took: the game date minus 4 July 161 (GAME.EXE's
	// C_0903_0A1E; a year has 13 months of 28 days).
	Years, Months, Days int
}

// ending is END.EXE's state: its globals.
type ending struct {
	fsys fs.FS
	args Args

	screen surface // g_screen, the VGA memory
	back   surface // g_1ec0, the picture scenes are drawn on

	// clip rectangles on the back picture
	leftHalf   surface // g_1778
	rightHalf  surface // g_1786
	leftFloor  surface // g_1794, the left half above the floor line, where a gate rises
	rightFloor surface // g_17a2, the same on the right
	vortexBack surface // g_1eae, the vortex on the back picture
	vortex     surface // g_1e9e, the vortex on the screen

	// showAroundVortex copies these three back picture areas to the screen
	aroundBack, aroundScreen [3]surface // g_17da g_17f6 g_17be, g_17cc g_17e8 g_17b0

	// the text scroll (showText, showWindow)
	textPicture surface // g_textSurf, the text is printed here
	textDest    surface // g_1ede, where it goes on the back picture
	scroll      scroll

	dac          [colours * rgbBytes]byte
	savedPalette [colours]rgb // g_palette, ENDPAL.LBM's colours
	fade         fader

	shapes []*vga.Shape // END.SHP
	font   *font        // U6.SET
	win    textWindow

	years, months, days string // g_years, g_months, g_days

	scene      int  // g_1831, the text shown
	effectDone int  // g_1812, the last vortex effect played (1 Codex, 2 opened, 3 gone)
	moving     bool // g_0254, a gate is rising or sinking

	stars          starField
	tickHandler    func() // g_tickFn
	timerHooked    bool   // g_timerOn
	curMusic       int    // g_curMusic
	pendingScan    int    // what getch returns after the 0 of an extended key
	synced         bool
	mouseL, mouseR bool
}

// Run plays the ending. fsys holds the game's files. When it returns, END.EXE had switched
// to text mode and exited to DOS: it starts no other program.
func Run(fsys fs.FS, args Args) error {
	e := &ending{fsys: fsys, args: args}
	if err := e.load(); err != nil {
		return err
	}
	sound.Start(fsys)
	host.ShowFrame = e.frame
	defer func() { activeTimer = nil }()
	e.main()
	return nil
}

func (e *ending) readFile(name string) ([]byte, error) {
	b, err := fs.ReadFile(e.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		b, err = fs.ReadFile(e.fsys, strings.ToLower(name))
	}
	return b, err
}

// load reads END.EXE's files: the font, END.SHP and the palette.
// C: seg_00A9:0BF0 (main), the part after the video set up
func (e *ending) load() error {
	e.screen = newSurface()
	e.back = newSurface()
	e.textPicture = newSurface()
	packed, err := e.readFile(fontFile)
	if err != nil {
		return err
	}
	raw, err := data.Decompress(packed)
	if err != nil {
		return err
	}
	e.font = &font{raw: raw}
	if e.shapes, err = vga.LoadShapes(e.fsys, shapeFile); errors.Is(err, fs.ErrNotExist) {
		e.shapes, err = vga.LoadShapes(e.fsys, strings.ToLower(shapeFile))
	}
	if err != nil {
		return err
	}
	return e.loadPalette()
}

const (
	fontFile    = "U6.SET"
	shapeFile   = "END.SHP"
	paletteFile = "ENDPAL.LBM"
)

// frame is what the host shows.
func (e *ending) frame() *host.Frame { return &host.Frame{Pixels: e.screen.Pixels, Palette: &e.dac} }

// numberWords are the numbers the congratulation spells (g_numbers).
var numberWords = [...]string{
	"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
	"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen",
	"twenty", "twenty-one", "twenty-two", "twenty-three", "twenty-four", "twenty-five", "twenty-six", "twenty-seven", "twenty-eight", "twenty-nine",
}

// spell writes n and a unit ("one year", "two years").
func spell(n int, unit string) string {
	word := ""
	if n >= 0 && n < len(numberWords) {
		word = numberWords[n]
	} else {
		// fix: the original read past g_numbers (garbage) for 30 and more, or a negative
		// day count from GAME.EXE's borrow bug; the port writes the digits.
		word = itoa(n)
	}
	if n != 1 {
		unit += "s"
	}
	return word + " " + unit
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + itoa(n%10)
}

// Layout of the scene (END.SHP shapes and where they go), in screen pixels.
const (
	shapeVortexStars = 0 // the vortex's starry swirl
	shapeCodexClosed = 1
	shapeCodexOpen   = 2
	shapeGate        = 3 // a red moongate
	shapeWall        = 4 // the wall the vortex appears on
	shapeAvatar      = 5 // the Avatar and the lens, left
	shapeLensRight   = 6 // the second lens on the floor, right
	shapeHall        = 7 // the throne room
	shapeLordsArrive = 8 // a half of the room with its lord arrived
	shapeLordsTurned = 9 // the same half once the lord has acted

	hallX, hallY               = 0, 0
	vortexLeft, vortexRight    = 106, 218
	vortexTop, vortexBottom    = 0, 93
	avatarX, avatarY           = 73, 87
	lensRightX, lensRightY     = 218, 87
	codexClosedX, codexClosedY = 151, 29
	codexOpenX, codexOpenY     = 133, 23
	halfSplit                  = 160 // the last column of the left half and the first of the right
	floorLine                  = 125 // a gate rises from below this row
	leftGateX, rightGateX      = 9, 230
	gateBuried                 = floorLine // the gate's y when it is below the floor
	gateRisen                  = -12       // and when it is fully up
	aroundSideBottom           = 80
	aroundLeftRight            = 105
	aroundRightLeft            = 213
	congratulationsScene       = 15
)

// Songs.
const (
	songBrit     = sound.SongBrit
	songGargoyle = sound.SongGargoyle
	songEnd      = sound.SongEnd
)

// Dissolve speeds of the vortex effects.
const (
	dissolveToStars    = 70
	dissolveCodexIn    = 30
	dissolveCodexOpen  = 30
	dissolveStarsAgain = 15
	dissolveWallAgain  = 80
	pauseBeforeEffect  = 5
	pauseBeforeClosing = 30
	pauseBeforeMoons   = 30
)

// main is END.EXE's main.
// C: seg_00A9:0BF0 (main)
func (e *ending) main() {
	if e.args.Years != 0 {
		e.years = spell(e.args.Years, "year")
	}
	if e.args.Months != 0 {
		e.months = spell(e.args.Months, "month")
	}
	if e.args.Days != 0 || e.args.Years == 0 && e.args.Months == 0 {
		e.days = spell(e.args.Days, "day")
	}
	e.playMusic(songBrit)

	e.leftHalf, e.rightHalf, e.leftFloor, e.rightFloor = e.back, e.back, e.back, e.back
	e.vortexBack, e.vortex = e.back, e.screen
	e.vortexBack.Left, e.vortexBack.Right = vortexLeft, vortexRight
	e.vortexBack.Top, e.vortexBack.Bottom = vortexTop, vortexBottom
	e.vortex.Left, e.vortex.Right, e.vortex.Top, e.vortex.Bottom = vortexLeft, vortexRight, vortexTop, vortexBottom
	e.leftHalf.Right, e.rightHalf.Left = halfSplit, halfSplit
	e.leftFloor.Right, e.rightFloor.Left = halfSplit, halfSplit
	e.leftFloor.Bottom, e.rightFloor.Bottom = floorLine, floorLine
	for i := range e.aroundBack {
		e.aroundBack[i], e.aroundScreen[i] = e.back, e.screen
	}
	for _, s := range []*surface{&e.aroundBack[0], &e.aroundScreen[0]} {
		s.Right, s.Bottom = aroundLeftRight, aroundSideBottom
	}
	for _, s := range []*surface{&e.aroundBack[1], &e.aroundScreen[1]} {
		s.Left, s.Bottom = aroundRightLeft, aroundSideBottom
	}
	for _, s := range []*surface{&e.aroundBack[2], &e.aroundScreen[2]} {
		s.Top = aroundSideBottom + 1
	}

	e.savePalette()
	e.calibrate()
	e.draw(e.back, vortexLeft, vortexTop, shapeWall)
	e.draw(e.back, hallX, hallY, shapeHall)
	e.draw(e.back, avatarX, avatarY, shapeAvatar)
	e.draw(e.back, lensRightX, lensRightY, shapeLensRight)
	e.showText(0)
	e.fadeIn(e.back, e.screen)
	copyRect(e.back, e.screen)
	e.showWindow()
	e.stars.init()
	e.waitTicks(pauseBeforeMoons)
	e.showAroundVortex()
	e.timerOn()
	e.tickHandler = e.starsTick

	// The gate rises with Lord British; a key brings it up at once.
	step := calcSpeed(3, 6)
	y := gateBuried
	for done := false; !done; {
		done = y == gateRisen
		e.draw(e.back, hallX, hallY, shapeHall)
		e.draw(e.leftFloor, leftGateX, y, shapeGate)
		e.draw(e.back, avatarX, avatarY, shapeAvatar)
		e.draw(e.back, lensRightX, lensRightY, shapeLensRight)
		e.showWindow()
		e.showAroundVortex()
		if e.checkKey() != 0 {
			y = step + gateRisen
		}
		y = max(y-step, gateRisen)
	}
	e.showText(1)
	for e.checkKey() == 0 {
	}

	for done := false; !done; {
		e.moving = false
		e.draw(e.back, hallX, hallY, shapeHall)
		if e.scene < 2 && y < gateBuried {
			// Lord British's gate sinks once he stepped out
			e.draw(e.leftFloor, leftGateX, y, shapeGate)
			if e.scene == 1 {
				y += step
				e.moving = true
			}
		}
		if e.scene == 6 || e.scene == 7 {
			// Draxinusom's gate rises
			e.moving = true
			e.draw(e.rightFloor, rightGateX, y, shapeGate)
			y = max(y-step, gateRisen)
		}
		if e.scene == 8 && y < gateBuried {
			e.moving = true
			y += step
			e.draw(e.rightFloor, rightGateX, y, shapeGate)
		}
		if e.scene >= 7 && e.scene <= 10 {
			e.draw(e.rightHalf, hallX, hallY, shapeLordsArrive)
		}
		if e.scene > 10 {
			e.draw(e.rightHalf, hallX, hallY, shapeLordsTurned)
		} else {
			e.draw(e.back, lensRightX, lensRightY, shapeLensRight)
		}
		if e.scene < 4 {
			e.draw(e.leftHalf, hallX, hallY, shapeLordsArrive)
			e.draw(e.back, avatarX, avatarY, shapeAvatar)
		} else {
			e.draw(e.leftHalf, hallX, hallY, shapeLordsTurned)
		}
		if e.scene != 14 {
			e.showWindow()
		}
		e.showAroundVortex()

		if e.scene == 4 && e.effectDone != 1 {
			// the Codex appears in the vortex
			e.waitTicks(pauseBeforeEffect)
			e.draw(e.vortexBack, vortexLeft, vortexTop, shapeVortexStars)
			e.dissolve(e.vortexBack, e.vortex, dissolveToStars)
			e.draw(e.vortexBack, codexClosedX, codexClosedY, shapeCodexClosed)
			e.dissolve(e.vortexBack, e.vortex, dissolveCodexIn)
			e.flushKeys()
			e.effectDone = 1
		}
		if e.scene == 12 && e.effectDone != 2 {
			// the Codex opens
			e.waitTicks(pauseBeforeEffect)
			e.playMusic(songEnd)
			e.draw(e.vortexBack, codexOpenX, codexOpenY, shapeCodexOpen)
			e.dissolve(e.vortexBack, e.vortex, dissolveCodexOpen)
			e.flushKeys()
			e.effectDone = 2
		}
		if e.scene == 14 && e.effectDone != 3 {
			// the vision fades and the wall comes back
			e.waitTicks(pauseBeforeClosing)
			e.draw(e.vortexBack, vortexLeft, vortexTop, shapeVortexStars)
			e.dissolve(e.vortexBack, e.vortex, dissolveStarsAgain)
			e.draw(e.vortexBack, vortexLeft, vortexTop, shapeWall)
			e.dissolve(e.vortexBack, e.vortex, dissolveWallAgain)
			e.tickHandler = nil
			e.effectDone = 3
			done = true
		}
		if !e.moving {
			for e.checkKey() == 0 {
			}
		}
		if e.checkKey() != 0 || !e.moving {
			e.scene++
			switch e.scene {
			case 6:
				e.showText(7)
				e.playMusic(songGargoyle)
				y = gateBuried
			case 7:
				e.showText(8)
				y = gateRisen
			case congratulationsScene:
				done = true
			case 14:
			default:
				e.showText(e.scene + 1)
			}
		}
	}

	e.timerOff()
	e.fadeOut(e.screen, 0)
	fill(e.screen, 0)
	e.restorePalette()
	e.showText(congratulationsScene)
	fill(e.back, 0)
	e.showWindow()
	e.fadeIn(e.back, e.screen)
	e.flushKeys()
	for e.checkKey() == 0 {
	}
	e.playMusic(sound.SongStop)
}

// draw draws a shape of END.SHP with its hot spot at (x, y), clipped to s.
// C: driver entry 3
func (e *ending) draw(s surface, x, y, shape int) {
	if shape < len(e.shapes) {
		s.DrawShape(x, y, e.shapes[shape])
	}
}

// showAroundVortex copies the back picture to the screen except for the vortex, which the
// star field and the dissolves own.
// C: seg_00A9:0B26 (showMoons)
func (e *ending) showAroundVortex() {
	for i := range e.aroundBack {
		copyRect(e.aroundBack[i], e.aroundScreen[i])
	}
}

// calcSpeed picks a step between a and b by the machine's speed: a on a fast machine, b on a
// slow one. The port takes a fast machine (calibratedLoops), as the intro does.
// C: seg_00A9:1D9C (calcSpeed)
func calcSpeed(a, b int) int {
	const slowest, fastest = 1500, 20000
	loops := min(max(calibratedLoops, slowest), fastest) - slowest
	span := abs(a - b)
	step := min(loops/((fastest-slowest)/(span+1)), span)
	if a > b {
		return b + step
	}
	return b - step
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// playMusic starts song n (sound.Songs; 0 stops) unless the sound is off.
// C: seg_00A9:1FC8 (playMusic)
func (e *ending) playMusic(n int) {
	if !e.args.Music || n == e.curMusic {
		return
	}
	if e.curMusic != 0 {
		_ = sound.Play(sound.SongStop)
		e.curMusic = 0
	}
	if n != 0 {
		_ = sound.Play(n)
		e.curMusic = n
	}
}

// Input.

// keyWaiting reports a key in the BIOS buffer.
func (e *ending) keyWaiting() bool { return e.pendingScan != 0 || len(host.KeyQueue) > 0 }

// kbhit reports a key; a loop polling it runs once per host tick.
func (e *ending) kbhit() bool {
	e.idle()
	return e.keyWaiting()
}

// getch reads a key: its character, or 0 and then its scan code.
func (e *ending) getch() int {
	for !e.keyWaiting() {
		e.yield()
	}
	if e.pendingScan != 0 {
		scan := e.pendingScan
		e.pendingScan = 0
		return scan
	}
	k := host.KeyQueue[0]
	host.KeyQueue = host.KeyQueue[1:]
	if k.Char == 0 {
		e.pendingScan = int(k.Scan)
	}
	return int(k.Char)
}

func (e *ending) flushKeys() {
	for e.kbhit() {
		e.getch()
	}
}

// pollMouse reads the buttons (never pressed without a mouse).
func (e *ending) pollMouse() {
	e.mouseL = e.args.Mouse && host.MouseLeft
	e.mouseR = e.args.Mouse && host.MouseRight
}

// checkKey returns a key pressed, 1 for a mouse click (after the buttons are released) or 0.
// C: seg_00A9:09E5 (checkKey)
func (e *ending) checkKey() int {
	e.pollMouse()
	if e.mouseL || e.mouseR {
		for e.mouseL || e.mouseR {
			e.yield()
			e.pollMouse()
		}
		return 1
	}
	if !e.kbhit() {
		return 0
	}
	return e.getch()
}

// waitTicks waits n 400ths of a timer tick, or until a key.
// C: seg_00A9:0A25 (waitTicks), seg_00A9:0A49 (delay)
func (e *ending) waitTicks(n int) {
	for ; n > 0; n-- {
		e.wait(timerTick / loopsPerDelayUnit)
		if e.checkKey() != 0 {
			break
		}
	}
}

// Timing.

const (
	pitHz      = 1193182
	pitDivisor = 65536
	timerHz    = float64(pitHz) / pitDivisor
	// timerTick is the length of a timer tick (int 1Ch).
	timerTick = time.Second * pitDivisor / pitHz
	// calibratedLoops is g_cpuLoops, the busy loops per timer tick of a fast machine.
	calibratedLoops   = 20000
	loopsPerDelayUnit = 400
)

// activeTimer is the hooked timer interrupt; host.AddTimer can not be undone, so it is
// installed once and calls whichever ending is running.
var (
	activeTimer func()
	timerAdded  bool
	timerDue    float64
)

// timerOn hooks the timer interrupt with an empty handler.
// C: seg_079D:0053 (timerOn)
func (e *ending) timerOn() {
	if !e.timerHooked {
		e.timerHooked = true
		e.tickHandler = nil
		activeTimer = func() {
			if e.tickHandler != nil {
				e.tickHandler()
			}
		}
	}
	if !timerAdded {
		timerAdded = true
		host.AddTimer(func() {
			timerDue += timerHz / host.TicksPerSecond
			for ; timerDue >= 1; timerDue-- {
				if activeTimer != nil {
					activeTimer()
				}
			}
		})
	}
}

// timerOff unhooks it.
// C: seg_079D:009A (timerOff)
func (e *ending) timerOff() {
	e.timerHooked = false
	activeTimer = nil
}

// calibrate measured the machine's speed over a timer tick.
// C: seg_079D:00BC (calibrate)
func (e *ending) calibrate() {
	const ticksWaited = 3
	e.wait(ticksWaited * timerTick)
}

// Frame pacing, as in the intro: every wait for the retrace is one host tick, and a loop
// polling the keyboard runs at most once per tick.

func (e *ending) yield() {
	host.Yield()
	e.synced = true
}

func (e *ending) wait(d time.Duration) {
	host.Wait(d)
	e.synced = true
}

func (e *ending) waitRetrace() { e.yield() }

func (e *ending) idle() {
	if !e.synced {
		e.yield()
	}
	e.synced = false
}

// random is Turbo C's rand range (0..32767), seeded from the clock.
func random() int { return rand.IntN(randMax + 1) }

const randMax = 32767
