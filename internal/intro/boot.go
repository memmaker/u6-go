package intro

import (
	"time"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// The introduction U.EXE plays when it starts: the avatar at home in front of the
// television, the storm, the moongate rising among the stones (INTRO_1/2/3.SHP).

// Script opcodes of the television (128 and up; smaller values are a picture to show).
const (
	opRepeat      = 128 + iota // n: repeat until opLoop n times
	opLoop                     // end of a repeat; the frame is done
	opNoise                    // draw static; the frame is done
	opBars                     // the rolling bars of a broken picture
	opPick                     // n p1..pn: show one of the pictures at random (not the last one)
	opNextChannel              // switch to the next channel
	opChosen                   // show the picture opChoose chose
	opChoose                   // choose one of tvChoices
	opBlink                    // a blinking light every few frames
	opStatic                   // one of the three static pictures
	opFrame                    // the frame is done
	opCycle                    // cycle the colours of the picture
)

// channels are the television's programmes: scripts of pictures (INTRO_1.SHP 16 and up).
var channels7 = [...][]int{
	{opNoise, opNoise, opRepeat, 3, 2, opFrame, 2, opFrame, 1, opFrame, 1, opFrame, 0, opFrame, 0, opFrame, 1, opFrame, 1, opLoop, opNextChannel},
	{opNoise, opNoise, opRepeat, 40, 3, opCycle, opLoop, opNextChannel},
	{opNoise, opNoise, opRepeat, 4, 4, opLoop, opRepeat, 4, 8, opLoop, opRepeat, 4, 9, opLoop, opRepeat, 4, 10, opLoop, opRepeat, 4, 11, opLoop, opRepeat, 4, 12, opLoop, opNextChannel},
	{opNoise, opNoise, opChoose, opRepeat, 70, 15, opChosen, opPick, 9, 16, 16, 16, 17, 17, 17, 18, 18, 18, opLoop, opNextChannel},
	{opNoise, opNoise, opRepeat, 50, opBars, opLoop, opNextChannel},
	{opNoise, opNoise, opRepeat, 5, 39, opFrame, 40, opFrame, 41, opLoop, opRepeat, 6, 42, opFrame, 42, opFrame, 43, opFrame, 43, opFrame, 44, opFrame, 44, opFrame, 45, opFrame, 45, opLoop, opRepeat, 10, 46, opFrame, 47, opFrame, 48, opFrame, opPick, 9, 46, 46, 46, 47, 47, 47, 48, 48, 48, opFrame, 46, opFrame, 47, opFrame, 48, opLoop, opNextChannel},
	{opNoise, opNoise, opRepeat, 85, 22, 23, opPick, 12, 19, 19, 19, 19, 20, 20, 20, 20, 21, 21, 21, 21, opBlink, opLoop, opRepeat, 15, 22, opPick, 2, 26, 27, opStatic, opBlink, opLoop, opRepeat, 3, 22, 26, opStatic, opBlink, opLoop, opRepeat, 3, 22, 28, opBlink, opLoop, opRepeat, 3, 22, 29, opBlink, opLoop, opRepeat, 3, 22, 30, opBlink, opLoop, opRepeat, 3, 22, 35, opBlink, opLoop, opRepeat, 3, 22, 36, opBlink, opLoop, opRepeat, 50, 22, opBlink, opLoop, opNextChannel},
}

const channels = len(channels7)

// tvChoices are the pictures opChoose picks from.
var tvChoices = [...]int{5, 6, 7, 13, 14, 24, 25, 31, 32}

// tvPictureX and tvPictureY place the television pictures on the screen.
var (
	tvPictureX = [54]int{0, 0, 0, 0, 0, 31, 31, 31, 0, 0, 0, 0, 0, 31, 31, 0, 9, 9, 9, 12, 12, 12, 0, 4, 31, 31, 4, 0, 4, 4, 4, 31, 31, 0, 0, 6, 6, 8, 4}
	tvPictureY = [54]int{0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 2, 2, 0, 7, 7, 7, 3, 3, 3, 0, 2, 2, 2, 0, 0, 0, 0, 1, 2, 2, 0, 0, 3, 8, 29, 28}
)

const (
	tvPictureShapes  = 16 // INTRO_1.SHP index of television picture 0
	barsPicture      = 34
	barsTopPicture   = 33
	barsPeriod       = 14
	barsFirstRow     = 21
	barsSecondRow    = 36
	blinkPicture     = 37
	staticPicture    = 50
	staticPictures   = 3
	repeatRandomFrom = 10
	maxClouds        = 10
	maxRaindrops     = 150
)

// bootState is the state of the boot intro.
type bootState struct {
	// the television
	tv           surface // D_5FBA, the screen of the television on the back page
	tvOffset     int     // D_6066, how far the television has scrolled out of the picture
	noiseDrawn   bool    // D_606E
	pictures     shapes  // D_6060, INTRO_1.SHP
	channel      int     // D_5BE8
	pos          int     // D_5BF8
	repeatPos    int     // D_5B9A
	repeats      int     // D_5B9C
	chosen       int     // D_5B9E
	lastPick     int     // D_1302
	picture      int     // D_1304, shown again every step until changed
	barsPhase    int     // D_1306
	blinkCounter int     // D_1300
	cycleCounter int     // D_12FE
	pictureY     [len(tvPictureY)]int

	moongateCycle bool // D_1308

	// the storm
	clouds                 *surface // D_5BEE
	cloudShapes            shapes   // D_5BEA
	cloudCount             int      // D_5BE6
	cloudX, cloudY, cloudN [maxClouds]int
	skyX, skyY             int // D_5BF4, D_5BF6
	cloudSpeed             int // D_605E
	rain                   *surface
	rainShapes             shapes // D_6068
	rainMax, rainCount     int    // D_606C, D_6070
	rainDelay, rainChance  int    // D_5BF2, D_6064
	dropShape              [maxRaindrops]int
	dropX, dropY, dropWait [maxRaindrops]int
}

func (b *bootState) init() {
	b.pictureY = tvPictureY
	b.lastPick, b.picture = -1, -1
	b.barsPhase = barsPeriod
}

// setChannel switches the television to a channel (the digit keys do it).
// C: seg_06E9:02D5 (INTRO_setLine)
func (u *intro) setChannel(channel int) {
	u.boot.pos = 0
	u.boot.channel = channel
}

// drawTVPicture draws television picture n.
// C: seg_06E9:02E6 (INTRO_drawSprite)
func (u *intro) drawTVPicture(n int) {
	b := &u.boot
	b.pictures.draw(b.tv, tvPictureX[n]+b.tv.left-b.tvOffset, b.pictureY[n]+b.tv.top, n+tvPictureShapes)
}

// noiseBits stands in for the bytes of its own code the original read as random bits: a
// fixed pseudo random stream starting at the offset it chose.
func noiseBits(offset int) func() bool {
	r := borlandRand{seed: uint32(offset)}
	return func() bool { return r.next()&1 != 0 }
}

// drawNoise fills the television with static.
// C: seg_06E9:000E (INTRO_noise)
func (u *intro) drawNoise() {
	const codeBytes = 3584
	b := &u.boot
	bit := noiseBits(u.rand() & codeBytes)
	for y := b.tv.top; y <= b.tv.bottom; y++ {
		for x := b.tv.left; x <= b.tv.right; x++ {
			c := 0
			if bit() {
				c = whiteColour
			}
			if b.tv.inPicture(x, y) {
				b.tv.pixels[b.tv.at(x, y)] = byte(c)
			}
		}
	}
	b.noiseDrawn = true
}

// rotateColours moves colours first..last of the DAC up by one, the last to first.
func (u *intro) rotateColours(first, last int) {
	top := u.getDAC(last)
	for i := last - 1; i >= first; i-- {
		u.setDAC(i+1, u.getDAC(i))
	}
	u.setDAC(first, top)
}

// cycleTVColours cycles colours 224..229 every fourth call.
// C: seg_06E9:0262 (INTRO_cyclePal)
func (u *intro) cycleTVColours() {
	const first, last, every = 224, 229, 3
	b := &u.boot
	if b.cycleCounter == every {
		b.cycleCounter = 0
		u.waitRetrace()
		u.rotateColours(first, last)
		return
	}
	b.cycleCounter++
}

// runScript runs the television's script up to the end of a frame.
// C: seg_06E9:03C3 (INTRO_runScript)
func (u *intro) runScript() {
	b := &u.boot
	b.noiseDrawn = false
	for done := false; !done; {
		script := channels7[b.channel]
		op := script[b.pos]
		b.pos++
		switch {
		case op < opRepeat:
			b.picture = op
		case op == opBars:
			if b.barsPhase == 0 {
				b.barsPhase = barsPeriod
			} else {
				b.barsPhase--
			}
			b.pictureY[barsPicture] = barsFirstRow - b.barsPhase
			u.drawTVPicture(barsPicture)
			b.pictureY[barsPicture] = barsSecondRow - b.barsPhase
			u.drawTVPicture(barsPicture)
			b.picture = barsTopPicture
		case op == opNextChannel:
			b.pos = 0
			b.channel = (b.channel + 1) % channels
		case op == opNoise:
			u.drawNoise()
			b.picture = -1
			done = true
		case op == opRepeat:
			b.repeats = script[b.pos]
			b.pos++
			if b.repeats >= repeatRandomFrom {
				b.repeats = randomBetween(b.repeats, b.repeats/4)
			}
			b.repeatPos = b.pos
		case op == opLoop:
			if b.repeats != 0 {
				b.repeats--
				b.pos = b.repeatPos
			}
			done = true
		case op == opFrame:
			done = true
		case op == opPick:
			n := script[b.pos]
			b.pos++
			k := 0
			for {
				k = u.rand()%n + b.pos
				if k != b.lastPick {
					break
				}
			}
			b.lastPick = k
			b.picture = script[k]
			b.pos += n
		case op == opChoose:
			b.chosen = tvChoices[u.rand()%len(tvChoices)]
		case op == opChosen:
			b.picture = b.chosen
		case op == opCycle:
			u.cycleTVColours()
		case op == opStatic:
			b.picture = u.rand()%staticPictures + staticPicture
		case op == opBlink:
			const lit, phase, mask = 4, 16, 255
			b.blinkCounter = (b.blinkCounter + 1) & mask
			if b.blinkCounter&lit != 0 {
				b.picture = (b.blinkCounter&phase)/phase + blinkPicture
			}
		}
		if b.picture > -1 {
			u.drawTVPicture(b.picture)
		}
	}
}

// cycleMoongateColours cycles colours 144..159 every other timer tick (a timer callback).
// C: seg_06E9:05BC (INTRO_cyclePal2)
func (u *intro) cycleMoongateColours() {
	const first, last = 144, 159
	b := &u.boot
	if !b.moongateCycle {
		u.rotateColours(first, last)
	}
	b.moongateCycle = !b.moongateCycle
}

func (u *intro) randomX(s surface) int { return u.rand()%(s.right-s.left+1) + s.left }
func (u *intro) randomY(s surface) int { return u.rand()%(s.bottom-s.top+1) + s.top }

const (
	cloudRows      = 30
	cloudShape     = 2
	cloudStartBack = 260
	cloudRestart   = 320
	cloudLead      = 140
	skyShape       = 0
	skyStartX      = -400
	skyStartY      = -5
	skyWidth       = 320
)

// initClouds places n clouds of shapes on s.
// C: seg_06E9:066C (INTRO_initClouds)
func (u *intro) initClouds(s *surface, n int, lib shapes) {
	b := &u.boot
	b.clouds, b.cloudShapes, b.cloudCount = s, lib, n
	for i := range n {
		b.cloudX[i] = u.rand()%(s.right-s.left) + s.left - cloudStartBack
		b.cloudY[i] = u.rand() % cloudRows
		b.cloudN[i] = u.rand()&1 + cloudShape
		b.cloudX[i] -= cloudLead
	}
	b.skyX, b.skyY = skyStartX, skyStartY
}

// drawClouds draws and moves the clouds and the sky behind them.
// C: seg_06E9:06FD (INTRO_clouds)
func (u *intro) drawClouds() {
	b := &u.boot
	s := b.clouds
	for i := range b.cloudCount {
		b.cloudShapes.draw(*s, b.cloudX[i], b.cloudY[i], b.cloudN[i])
		if b.cloudX[i] > s.right {
			b.cloudX[i] = u.rand()%(s.right-s.left) + s.left - cloudRestart
			b.cloudY[i] = u.rand() % cloudRows
			b.cloudN[i] = u.rand()&1 + cloudShape
			b.cloudX[i] -= cloudLead
		}
		b.cloudX[i] += b.cloudSpeed
	}
	b.cloudShapes.draw(*s, b.skyX, b.skyY, skyShape)
	b.cloudShapes.draw(*s, b.skyX-skyWidth, b.skyY, skyShape)
	b.skyX += b.cloudSpeed >> 1
	if b.skyX == skyWidth {
		b.skyX = 0
	}
}

// initRain prepares n raindrops falling on s, moving every delay+1 frames.
// C: seg_06E9:081A (INTRO_initRain)
func (u *intro) initRain(s *surface, n, delay int) {
	b := &u.boot
	b.rainMax, b.rainCount, b.rainDelay, b.rain = n-1, 0, delay, s
}

const (
	dropShape   = 4
	dropShapes  = 4
	dropMargin  = 10
	dropRows    = 9
	dropAbove   = 4
	dropFallX   = 2
	dropFallY   = 10
	dropSpreadX = 21
)

// newDrop puts raindrop i at a random place along the top.
func (u *intro) newDropY(s *surface) int { return u.rand()%dropRows + s.top - dropAbove }
func (u *intro) newDropX(s *surface) int {
	return u.rand()%(s.right-s.left+dropSpreadX) + s.left - dropMargin
}

// drawRain starts a raindrop now and then and draws and moves them.
// C: seg_06E9:0838 (INTRO_rain)
func (u *intro) drawRain() {
	b := &u.boot
	s := b.rain
	if b.rainCount != b.rainMax && u.rand()%b.rainChance == 0 {
		i := b.rainCount
		b.dropShape[i] = u.rand()&(dropShapes-1) + dropShape
		b.dropX[i] = u.newDropX(s)
		b.dropY[i] = u.newDropY(s)
		b.dropWait[i] = b.rainDelay
		b.rainShapes.draw(*s, b.dropX[i], b.dropY[i], b.dropShape[i])
		b.rainCount++
	}
	for i := range b.rainCount {
		if b.dropWait[i] != 0 {
			b.dropWait[i]--
		} else {
			b.dropWait[i] = b.rainDelay
			b.dropX[i] += dropFallX
			b.dropY[i] += dropFallY
			if b.dropY[i] > s.bottom {
				b.dropY[i] = u.newDropY(s)
				b.dropX[i] = u.newDropX(s)
				b.dropShape[i] = u.rand()&(dropShapes-1) + dropShape
			}
		}
		b.rainShapes.draw(*s, b.dropX[i], b.dropY[i], b.dropShape[i])
	}
}

// clockHours are the hours of a 12 hour clock.
var clockHours = [24]int{12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}

// drawClock draws the time of day on the video recorder at x.
// C: seg_06E9:09DC (INTRO_clock)
func (u *intro) drawClock(s surface, lib shapes, x int) {
	const row, digitShape, blankShape = 20, 2, 12
	now := time.Now()
	hour, minute := clockHours[now.Hour()], now.Minute()
	if hour < 10 {
		lib.draw(s, x, row, blankShape)
	} else {
		lib.draw(s, x, row, hour/10+digitShape)
	}
	lib.draw(s, x+4, row, hour%10+digitShape)
	lib.draw(s, x+10, row, minute/10+digitShape)
	lib.draw(s, x+14, row, minute%10+digitShape)
}

// restoreScreen shows the back page.
// C: seg_06E9:2AF6 (INTRO_restoreScreen)
//
// On a 386 with an ISA VGA card the scene redraw plus this 64000-byte copy took about four
// retraces, so animations (the TV, the scrolls) step at that rate; the wait yields every
// tick, so input still lands promptly. The port's machine would otherwise run them at 60 Hz.
func (u *intro) restoreScreen() {
	for host.Ticks-u.restoreTick < ticksPerSceneRedraw {
		u.yield()
	}
	u.restoreTick = host.Ticks
	u.copyRect(u.back, u.screen)
}

// ticksPerSceneRedraw is the 386's full scene redraw time in 60 Hz ticks (about 15 fps);
// raise it for a slower machine.
const ticksPerSceneRedraw = 4

// playIntro plays the boot intro, up to the main menu.
// C: seg_06E9:0BA6 (INTRO_play)
func (u *intro) playIntro() {
	u.loadPalette(1)
	u.savePalette(&u.scenePalette)
	u.calibrateTimer()
	u.installTimer()
	u.getKey()
	if !u.escaped && !sound.Settings.SkipIntro {
		u.introScenes()
		u.fadeOut(u.screen, 0)
		fill(u.screen, 0)
		fill(u.back, 0)
	}
	u.setTimerCallback(nil)
	u.removeTimer()
	u.setPalette(&u.menuPalette)
	u.playMusic(sound.SongStop)
}

// Pictures of INTRO_1.SHP.
const (
	roomShape        = 0
	outsideShape     = 1
	deskShape        = 13
	tvGlowShape      = 14
	treeShape        = 15
	originShape      = 69
	presentsShape    = 70
	deskTop          = 63
	tvGlowX, tvGlowY = 143, 91
	clockX           = 221
	logoFrames       = 50
	tvLeft, tvRight  = 229, 285
	tvTop, tvBottom  = 50, 86
	scrollWidth      = 320
	treeFrom         = 135
	stopScriptAt     = 300
)

// introScenes is INTRO_play up to its exit label; it returns at Escape.
func (u *intro) introScenes() {
	b := &u.boot
	room, outside, desk := u.back, u.back, u.back // D_5BAE, D_5BBC, D_5BCA
	u.playMusic(sound.SongBootup)
	lib := u.mustShapes("INTRO_1.SHP")
	b.pictures = lib
	fill(u.back, 0)
	lib.draw(u.back, 0, 0, originShape)
	u.startLoadedSong()
	u.fadeIn(u.back, u.screen)
	for range logoFrames {
		if u.getKey(); u.escaped {
			return
		}
		u.waitRetrace()
	}
	fill(u.back, 0)
	lib.draw(u.back, 0, 0, presentsShape)
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	u.setPalette(&u.scenePalette)
	u.fadeIn(u.back, u.screen)
	for range logoFrames {
		if u.getKey(); u.escaped {
			return
		}
		u.waitRetrace()
	}
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	u.setPalette(&u.scenePalette)
	if u.getKey(); u.escaped {
		return
	}

	// at home, in front of the television
	b.tv = u.back
	b.tv.left, b.tv.top, b.tv.right, b.tv.bottom = tvLeft, tvTop, tvRight, tvBottom
	u.setChannel(2)
	u.showPassage(0)
	lib.draw(room, 0, 0, roomShape)
	lib.draw(desk, 0, deskTop, deskShape)
	u.drawClock(u.back, lib, clockX)
	u.fadeIn(u.back, u.screen)
	if u.getKey(); u.escaped {
		return
	}
	b.tvOffset = 0
	homeFrame := func(passage bool) {
		lib.draw(room, 0, 0, roomShape)
		lib.draw(desk, 0, deskTop, deskShape)
		u.runScript()
		if b.noiseDrawn {
			lib.draw(desk, tvGlowX, tvGlowY, tvGlowShape)
			b.noiseDrawn = false
		}
		if passage {
			u.drawPassage()
		}
		u.drawClock(u.back, lib, clockX)
		u.restoreScreen()
	}
	for range calibratedLoops / loopsPerDelayUnit {
		homeFrame(false)
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}
	for {
		homeFrame(true)
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}
	u.flushKeys()
	u.showPassage(1)
	for {
		homeFrame(true)
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}

	// the view turns to the window
	u.showPassage(2)
	step := randomBetween(2, 10)
	for scroll, last := 0, false; !last; {
		last = scroll == scrollWidth
		if scroll >= treeFrom {
			const treeShift, treeSpan, treeX = 10, 36, 77
			lib.draw(outside, (scroll-treeFrom)*treeShift/treeSpan+(scrollWidth-scroll)+treeX, 0, treeShape)
		}
		lib.draw(room, room.left-scroll, 0, roomShape)
		lib.draw(outside, outside.left+scrollWidth-scroll, 0, outsideShape)
		lib.draw(desk, desk.left-scroll*2, deskTop, deskShape)
		u.drawPassage()
		if b.tv.right > 0 && scroll < stopScriptAt {
			u.runScript()
		}
		b.tv.right -= step
		b.tv.left -= step
		if b.tv.left < 0 {
			b.tvOffset -= b.tv.left
			b.tv.left = 0
		}
		u.drawClock(u.back, lib, clockX-scroll)
		u.waitRetrace()
		u.restoreScreen()
		if u.getKey() != 0 {
			scroll = scrollWidth - step
		}
		if u.escaped {
			return
		}
		scroll = min(scroll+step, scrollWidth)
	}
	if !u.stormScenes(room, outside) {
		return
	}
	u.moongateScenes()
}

// Pictures of INTRO_2.SHP.
const (
	stormSkyShape       = 1
	stormHillShape      = 10
	stormGroundShape    = 8
	stormLightningShape = 11
	stormBoltShape      = 11
	stormStonesShape    = 26
	stormStonesLitShape = 27
	stormFrameShape     = 28
	stormDoorShape      = 24
	stormDoor2Shape     = 25
	stormHillTop        = 76
	stormBoltX          = 158
	stormBoltY          = 114
	stormBoltRadius     = 160
	stormDoor2X         = 253
	stormDoorsOpen      = 75
	stonesLeft          = 57
	stonesRight         = 257
	stonesBottom        = 176
	skyBottom           = 46
	lightningAfter      = 60
	storyAfter          = 40
	framesWrapAt        = 1000
	framesWrapTo        = 900
	flashColourIndex    = 117
)

// flashColours are the colours a lightning flash brightens.
var flashColours = [...]int{88, 90, 92}

// stormScenes plays the storm (INTRO_2.SHP); false at Escape.
func (u *intro) stormScenes(stones, sky surface) bool {
	b := &u.boot
	lib := u.mustShapes("INTRO_2.SHP")
	b.rainShapes = lib
	if u.waitKey(); u.escaped {
		return false
	}
	stones.right, stones.left, stones.top, stones.bottom = stonesRight, stonesLeft, 0, stonesBottom
	sky.top, sky.bottom = 0, skyBottom
	doorStep := randomBetween(1, 5)
	scrollStep := randomBetween(4, 12)
	b.cloudSpeed = randomBetween(2, 10)
	b.rainChance = randomBetween(4, 1)
	u.initRain(&u.back, maxRaindrops, 0)
	u.initClouds(&u.back, 5, lib)
	u.showPassage(3)
	scroll, frames, flash, bolt, stage := 0, 0, 0, 0, 0
	flashShape, flashX, flashY := 0, 0, 0
	paletteLoaded := false
	var flashColour rgb
	var normal [len(flashColours)]rgb
	for done := false; !done; {
		lib.draw(u.back, 0, 0, stormSkyShape)
		u.drawClouds()
		if u.rand()%6 == 0 && frames >= lightningAfter && flash == 0 && paletteLoaded {
			flash = u.rand()&3 + 1
			flashShape = u.rand() & 7
			flashX = u.randomX(sky)
			flashY = u.randomY(sky)
		}
		if flash != 0 {
			y := u.rand()%3 + flashY - 2
			u.drawShapeAt(lib, u.rand()%3+flashX-2, y, flashShape+stormLightningShape)
		}
		if paletteLoaded {
			u.waitRetrace()
			if (flash != 0 || stage == 3) && (u.rand()%3 != 0 || stage == 3) {
				for _, c := range flashColours {
					u.setDAC(c, flashColour)
				}
			} else {
				for i, c := range flashColours {
					u.setDAC(c, normal[i])
				}
			}
		}
		lib.draw(u.back, 0, stormHillTop, stormHillShape)
		lib.draw(u.back, 0, 0, stormGroundShape)
		for previous := bolt; bolt == previous; {
			bolt = u.rand()%5 + 8
		}
		if stage == 3 || stage == 4 && scroll < stormBoltRadius {
			lib.draw(u.back, stormBoltX, stormBoltY, bolt+stormBoltShape)
		}
		u.drawRain()
		if stage < 5 {
			if flash != 0 && u.rand()&1 != 0 || stage == 3 {
				lib.draw(u.back, stones.left-scroll, stones.top, stormStonesLitShape)
			} else {
				lib.draw(u.back, stones.left-scroll, stones.top, stormStonesShape)
			}
		}
		switch stage {
		case 1, 2, 3:
			lib.draw(u.back, 0, 0, stormFrameShape)
			u.drawPassage()
		case 0:
			lib.draw(u.back, 0, 0, stormFrameShape)
		case 4:
			lib.draw(u.back, -scroll, 0, stormFrameShape)
			lib.draw(u.back, scrollWidth-scroll, 0, stormDoorShape)
			lib.draw(u.back, scrollWidth-scroll+stormDoor2X, 0, stormDoor2Shape)
			if scroll += scrollStep; scroll > scrollWidth {
				stage = 5
				scroll = 0
			}
			u.drawPassage()
		case 5:
			lib.draw(u.back, 0, 0, stormDoorShape)
			lib.draw(u.back, stormDoor2X, 0, stormDoor2Shape)
			u.drawPassage()
		case 6:
			lib.draw(u.back, -scroll, 0, stormDoorShape)
			lib.draw(u.back, scroll+stormDoor2X, 0, stormDoor2Shape)
			if scroll += doorStep; scroll > stormDoorsOpen {
				u.showPassage(8)
				scroll = 0
				stage = 7
			}
		case 7:
			u.drawPassage()
		}
		if !paletteLoaded {
			paletteLoaded = true
			u.waitRetrace()
			u.loadPalette(2)
			for i, c := range flashColours {
				normal[i] = u.getDAC(c)
			}
			flashColour = u.getDAC(flashColourIndex)
		}
		u.restoreScreen()
		if u.getKey() != 0 {
			switch stage {
			case 6:
				u.showPassage(8)
				stage++
				scroll = 0
			case 0, 4:
				stage++
				scroll = 0
			case 1:
				u.showPassage(4)
				stage++
			case 2:
				u.showPassage(5)
				stage++
			case 3:
				u.showPassage(6)
				stage++
			case 5:
				stage++
			default:
				done = true
			}
		}
		if u.escaped {
			return false
		}
		if flash != 0 {
			flash--
		}
		if frames != framesWrapAt {
			frames++
		} else {
			frames = framesWrapTo
		}
		if frames >= storyAfter && stage == 0 {
			stage = 1
		}
	}
	if u.getKey(); u.escaped {
		return false
	}
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	return true
}

// drawShapeAt draws a shape of lib on the back page.
func (u *intro) drawShapeAt(lib shapes, x, y, n int) { lib.draw(u.back, x, y, n) }

// Pictures of INTRO_3.SHP.
const (
	gateHillShape    = 0
	gateRisingShape  = 1
	gateArmShape     = 2
	gateStonesShape  = 3
	gateGroundShape  = 4
	gateSkyShape     = 5
	gateFigureShape  = 6
	gateLeapShape    = 7
	gateBeyondShape  = 26
	gateX            = 189
	gateHill         = 150
	gateHillY        = 100
	gateCutRow       = 100
	gateAreaBottom   = 101
	gateBottomStart  = 199
	gateRisen        = 84
	gateSunk         = 204
	armX             = 124
	armStart         = 101
	armRaised        = 6
	armRest          = 5
	groundX, groundY = 94, 102
	figureX          = 155
	lookUp           = 63
	figureLeapX      = -2
	figureLeapY      = 18
	leapFrames       = 59
	leapShapes       = 3
	beyondY          = -42
	gateDissolve     = 30
	gateGlowColour   = 25
)

// moongateScenes plays the moongate among the stones (INTRO_3.SHP).
func (u *intro) moongateScenes() {
	lib := u.mustShapes("INTRO_3.SHP")
	u.loadPalette(3)
	if u.getKey(); u.escaped {
		return
	}
	u.showPassage(9)
	lib.draw(u.back, 0, 0, gateHillShape)
	u.fadeIn(u.back, u.screen)
	if u.getKey(); u.escaped {
		return
	}
	u.setTimerCallback(u.cycleMoongateColours)
	for range 3 {
		lib.draw(u.back, 0, 0, gateHillShape)
		u.restoreScreen()
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}
	for {
		lib.draw(u.back, 0, 0, gateHillShape)
		u.drawPassage()
		u.restoreScreen()
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}
	u.back.bottom = gateCutRow
	fill(u.back, 0)
	u.back.bottom = screenHeight - 1
	gate := u.back // D_5BAE
	gate.bottom = gateAreaBottom
	u.showPassage(10)
	step := randomBetween(4, 10)
	for gateY, last := gateBottomStart, false; !last; {
		last = gateY == gateRisen
		lib.draw(u.back, 0, 0, gateHillShape)
		lib.draw(u.back, gateX, gateY, gateRisingShape)
		lib.draw(u.back, gateHill, gateHillY, gateStonesShape)
		u.drawPassage()
		u.restoreScreen()
		if u.getKey() != 0 {
			gateY = step + gateRisen
		}
		if u.escaped {
			return
		}
		gateY = max(gateY-step, gateRisen)
	}
	u.showPassage(11)
	u.waitKey()
	step = randomBetween(2, 8)
	for armY, last := armStart, false; !last; {
		last = armY == armRaised
		lib.draw(u.back, 0, 0, gateSkyShape)
		lib.draw(u.back, groundX, groundY, gateGroundShape)
		lib.draw(u.back, gateX, gateRisen, gateRisingShape)
		lib.draw(gate, armX, armY, gateArmShape)
		u.drawPassage()
		u.restoreScreen()
		if u.getKey() != 0 {
			armY = step + armRaised
		}
		if u.escaped {
			return
		}
		armY = max(armY-step, armRaised)
	}
	step = randomBetween(4, 10)
	for gateY, last := gateRisen, false; !last; {
		last = gateY == gateSunk
		lib.draw(u.back, 0, 0, gateSkyShape)
		lib.draw(u.back, groundX, groundY, gateGroundShape)
		lib.draw(u.back, gateX, gateY, gateRisingShape)
		lib.draw(gate, armX, armRest, gateArmShape)
		u.drawPassage()
		u.restoreScreen()
		if u.getKey() != 0 {
			gateY = gateSunk - step
		}
		gateY = min(gateY+step, gateSunk)
	}
	if u.waitKey(); u.escaped {
		return
	}
	// the view looks up
	lookFrame := func(lift int) {
		lib.draw(u.back, 0, -lift, gateSkyShape)
		lib.draw(u.back, groundX, groundY-lift, gateGroundShape)
		lib.draw(gate, armX, armRest-lift, gateArmShape)
		u.back.top = screenHeight - 1 - lift
		fill(u.back, 0)
		u.back.top = 0
		lib.draw(u.back, figureX, screenHeight-lift*2, gateFigureShape)
	}
	step = randomBetween(2, 8)
	for lift, last := 0, false; !last; {
		last = lift == lookUp
		lookFrame(lift)
		u.restoreScreen()
		if u.getKey() != 0 {
			last = true
		}
		if u.escaped {
			return
		}
		lift = min(lift+step, lookUp)
	}
	u.showPassage(12)
	for {
		lookFrame(lookUp)
		u.drawPassage()
		u.restoreScreen()
		if u.getKey() != 0 {
			break
		}
	}
	if u.escaped {
		return
	}
	fill(u.back, 0)
	step = randomBetween(2, 8)
	for lift, last := lookUp, false; !last; {
		last = lift == 0
		lookFrame(lift)
		u.restoreScreen()
		if u.getKey() != 0 {
			last = true
		}
		if u.escaped {
			return
		}
		lift = max(lift-step, 0)
	}
	standing := func(armX, armY int) {
		lib.draw(u.back, 0, 0, gateSkyShape)
		lib.draw(u.back, groundX, groundY, gateGroundShape)
		lib.draw(gate, armX, armY, gateArmShape)
	}
	untilKey := func(arm func() (int, int)) bool {
		for {
			standing(arm())
			u.drawPassage()
			u.restoreScreen()
			if u.getKey() != 0 {
				break
			}
		}
		return !u.escaped
	}
	still := func() (int, int) { return armX, armRest }
	shaking := func(rows int) func() (int, int) {
		return func() (int, int) {
			y := u.rand()&rows + armRest
			return u.rand()&1 + armX, y
		}
	}
	u.showPassage(13)
	if !untilKey(still) {
		return
	}
	u.showPassage(14)
	if !untilKey(shaking(1)) {
		return
	}
	u.showPassage(15)
	glow := u.getDAC(gateGlowColour)
	u.setDAC(gateGlowColour, rgb{})
	if !untilKey(shaking(3)) {
		return
	}
	saved := surface{right: gate.right - gate.left, bottom: gate.bottom - gate.top} // D_5BBC
	newPicture(&saved)
	step = randomBetween(1, 3)
	for leap, last := 0, false; !last; {
		last = leap == leapFrames
		standing(shaking(3)())
		lib.draw(u.back, figureLeapX, figureLeapY-leap, leap/leapShapes+gateLeapShape)
		u.restoreScreen()
		if u.getKey() != 0 {
			last = true
		}
		if u.escaped {
			return
		}
		leap = min(leap+step, leapFrames)
	}
	standing(still())
	u.copyRect(gate, saved)
	if u.getKey(); u.escaped {
		return
	}
	lib.draw(u.back, figureLeapX, beyondY, gateBeyondShape)
	u.restoreScreen()
	if u.getKey(); u.escaped {
		return
	}
	onScreen := gate
	samePicture(u.screen, &onScreen)
	u.dissolve(saved, onScreen, gateDissolve)
	if u.getKey(); u.escaped {
		return
	}
	step = randomBetween(2, 8)
	for armY, last := armRest, false; !last; {
		last = armY == armStart
		standing(armX, armY)
		u.restoreScreen()
		if u.getKey() != 0 {
			armY = armStart - step
		}
		if u.escaped {
			return
		}
		armY = min(armY+step, armStart)
	}
	u.setDAC(gateGlowColour, glow)
	fill(u.back, 0)
	for u.getKey() == 0 {
		lib.draw(u.back, 0, 0, gateHillShape)
		lib.draw(u.back, gateHill, gateHillY, gateStonesShape)
		u.restoreScreen()
	}
}
