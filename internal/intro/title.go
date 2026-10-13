package intro

import "github.com/memmaker/u6-go/internal/sound"

// The introduction of the main menu (INTRO.SHP): the avatar on the gargoyles' altar, the
// rescue by Dupre, Iolo and Shamino, under a twinkling starfield.

// Pictures of INTRO.SHP.
const (
	dupreShape           = 0
	ioloShape            = 1
	shaminoShape         = 2
	leftCrowdShape       = 3
	rightCrowdShape      = 4
	groundShape          = 5
	nightShape           = 6
	moongateShape        = 7
	priestShape          = 8
	avatarShape          = 9
	emptyHandShape       = 10
	daggerHandShape      = 11
	candleShape          = 12
	groundTop            = 112
	avatarX, avatarY     = 49, 68
	candleX, candleY     = 210, 132
	priestX              = 144
	handX, handY         = 222, 14
	priestRaised         = 30
	priestLow            = 141
	moongateX            = 120
	moongateUp           = 58
	moongateDown         = 134
	leftCrowdX           = -8
	leftCrowdY           = 77
	rightCrowdX          = 182
	rightCrowdY          = 79
	crowdNear            = 4
	crowdFar             = 6
	titlePassage         = 200
	moongateSinksPassage = titlePassage + 17
	tickWrap             = 1<<16 - 1 // the tick counter is 16 bits
	minLingering         = 15        // range of the random pause the crowd lingers
	maxLingering         = 40
	priestSwayStart      = 70 // the priest's sway while rising...
	priestSwayAbove      = 60 // ...jitters only above this height
	starCount            = 200
	starsShuffled        = 193
	starSwaps            = 512
	starColour           = whiteColour
	starfieldFirst       = 232
	starfieldLast        = 239
	starTwinkleMask      = 255
	starTwinkleTaps      = 184
	starTwinkleEvery     = 20
	starTwinkleStep      = 7
	starTwinkleRun       = 5
	starOffEvery         = 3
	dotColour            = 173
)

// titleState is TITLE_main's starfield, its timer callback's state.
type titleState struct {
	starX, starY, starColour [starCount]int // D_62BC, D_64BC, D_60BC
	first, run               int            // D_60AE, D_60B2: the stars lit white
	cycle                    bool           // D_40CC
	ticks                    int            // D_40CE

	// the crowds swaying (WANDER)
	left, right, leftStep, rightStep int
}

// twinkle runs on every timer tick: it cycles colours 232..239 and lights a few stars white.
// C: seg_0C17:0072 (TITLE_timer)
func (u *intro) twinkle() {
	t := &u.title
	if !t.cycle {
		top := u.getDAC(starfieldLast)
		for i := starfieldLast - 1; i >= starfieldFirst; i-- {
			u.setDAC(i+1, u.getDAC(i))
		}
		u.setDAC(starfieldFirst, top)
	}
	t.cycle = !t.cycle
	star := func(i int) *byte { return &u.screen.pixels[u.screen.at(t.starX[i], t.starY[i])] }
	if t.ticks&starOffEvery == 0 {
		for i := t.first; i <= t.first+t.run%starTwinkleRun; i++ {
			if *star(i) == starColour {
				*star(i) = byte(t.starColour[i])
			}
		}
	}
	t.ticks = (t.ticks + 1) & tickWrap
	if t.ticks%starTwinkleEvery == 0 {
		t.first = (t.first + starTwinkleStep) % starsShuffled
		odd := t.run&1 != 0
		t.run >>= 1
		if odd {
			t.run ^= starTwinkleTaps
		}
		for i := t.first; i <= t.first+t.run%starTwinkleRun; i++ {
			// a signed char against an int, as the original compared
			if int(int8(*star(i))) == t.starColour[i] {
				*star(i) = starColour
			}
		}
	}
}

// wander sways the two crowds by up to limit pixels.
func (u *intro) wander(limit int) {
	t := &u.title
	if u.rand()%3 != 0 {
		t.left += t.leftStep
	}
	if abs(t.left) == limit {
		t.leftStep = -t.leftStep
		t.left += t.leftStep
	}
	if u.rand()%3 != 0 {
		t.right += t.rightStep
	}
	if abs(t.right) == limit {
		t.rightStep = -t.rightStep
		t.right += t.rightStep
	}
}

// drawCrowds draws the gargoyles, shifted left and right, y lower.
// C: seg_0C17:0000 (TITLE_drawFigure)
func (u *intro) drawCrowds(lib shapes, left, right, y int) {
	lib.draw(u.back, left+leftCrowdX, y+leftCrowdY, leftCrowdShape)
	lib.draw(u.back, right+rightCrowdX, y+rightCrowdY, rightCrowdShape)
}

// drawDot draws the blood drop on the avatar.
// C: seg_0C17:03EF (TITLE_drawDot)
func (u *intro) drawDot() {
	const x, y = 88, 85
	for _, p := range [...][2]int{{0, 0}, {-1, 1}, {0, 1}, {1, 1}, {-1, 2}, {0, 2}, {1, 2}, {-1, 3}, {0, 3}} {
		putPixel(u.back, x+p[0], y+p[1], dotColour)
	}
}

// titleWait waits n short delays or a key.
// C: seg_0C17:03CA (TITLE_wait)
func (u *intro) titleWait(n int) {
	for ; n != 0; n-- {
		u.busyDelay(1)
		if u.getKey() != 0 {
			break
		}
	}
}

// waitTitleKey flushes the keyboard and waits for a key; false at Escape.
func (u *intro) waitTitleKey() bool {
	u.flushKeys()
	u.waitKey()
	return !u.escaped
}

// playTitle plays the introduction.
// C: seg_0C17:0508 (TITLE_main)
func (u *intro) playTitle() {
	u.escaped = false
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	u.calibrateTimer()
	lingering := randomBetween(minLingering, maxLingering)
	lib := u.mustShapes("INTRO.SHP")
	t := &u.title
	if raw, err := u.readLZW("STARPOS.DAT"); err == nil && len(raw) >= starCount*6 {
		for i := range starCount {
			word := func(table int) int { at := table*starCount*2 + i*2; return int(raw[at]) | int(raw[at+1])<<8 }
			t.starX[i], t.starY[i], t.starColour[i] = word(0), word(1), word(2)
		}
	}
	u.loadPalette(7)
	u.savePalette(&u.scenePalette)
	u.playMusic(sound.SongIntro)
	lib.draw(u.back, 0, 0, nightShape)
	lib.draw(u.back, 0, groundTop, groundShape)
	u.fadeIn(u.back, u.screen)
	for range starSwaps {
		a := u.rand() % starsShuffled
		b := u.rand() % starsShuffled
		t.starX[a], t.starX[b] = t.starX[b], t.starX[a]
		t.starY[a], t.starY[b] = t.starY[b], t.starY[a]
		t.starColour[a], t.starColour[b] = t.starColour[b], t.starColour[a]
	}
	t.run = u.rand()%starTwinkleMask + 1
	t.first = 0
	for i := t.first; i <= t.first+t.run%starTwinkleRun; i++ {
		putPixel(u.screen, t.starX[i], t.starY[i], starColour)
	}
	u.installTimer()
	u.setTimerCallback(u.twinkle)
	u.titleScenes(lib, lingering)
	u.removeTimer()
	u.setTimerCallback(nil)
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	u.setPalette(&u.menuPalette)
	u.menuDrawn, u.escaped = false, false
	u.playMusic(sound.SongStop)
}

// titleScenes is TITLE_main up to its end label; it returns at Escape.
func (u *intro) titleScenes(lib shapes, lingering int) {
	t := &u.title
	draw := func(x, y, n int) { lib.draw(u.back, x, y, n) }
	u.showPassage(titlePassage)
	u.titleWait(10)
	u.drawPassage()
	u.copyRect(u.back, u.screen)
	if !u.waitTitleKey() {
		return
	}
	u.showPassage(titlePassage + 1)
	u.drawPassage()
	u.copyRect(u.back, u.screen)
	if !u.waitTitleKey() {
		return
	}

	// the crowds gather
	crowds := []int{-84, 109, 320, 200}
	crowdsGathered := []int{leftCrowdX, leftCrowdY, rightCrowdX, rightCrowdY}
	if !u.slide(randomBetween(1, 4), crowds, crowdsGathered, func() {
		draw(0, 0, nightShape)
		draw(crowds[0], crowds[1], leftCrowdShape)
		draw(crowds[2], crowds[3], rightCrowdShape)
		draw(0, groundTop, groundShape)
	}) {
		return
	}
	u.showPassage(titlePassage + 2)
	u.drawPassage()
	u.restoreScreen()
	draw(0, 0, nightShape)
	u.drawCrowds(lib, 0, 0, 0)
	draw(0, groundTop, groundShape)
	u.showPassage(titlePassage + 3)
	u.drawPassage()
	if !u.waitTitleKey() {
		return
	}
	u.restoreScreen()
	draw(0, 0, nightShape)
	u.drawCrowds(lib, 0, 0, 0)
	draw(avatarX, avatarY, avatarShape)
	draw(0, groundTop, groundShape)
	draw(candleX, candleY, candleShape)
	u.showPassage(titlePassage + 4)
	u.drawPassage()
	if !u.waitTitleKey() {
		return
	}
	u.restoreScreen()
	if !u.waitTitleKey() {
		return
	}

	// the priest rises with the dagger
	u.showPassage(titlePassage + 5)
	step := randomBetween(2, 5)
	u.flushKeys()
	sway := priestSwayStart
	for priestY, last := priestLow, false; !last; {
		last = priestY == priestRaised
		x := u.rand()%8 + sway - 3
		if priestY > priestSwayAbove || sway == 0 {
			x = sway
		}
		draw(0, 0, nightShape)
		u.drawCrowds(lib, 0, 0, crowdNear)
		draw(x+priestX, priestY, priestShape)
		draw(x+handX, priestY+handY, daggerHandShape)
		draw(avatarX, avatarY, avatarShape)
		draw(0, groundTop, groundShape)
		draw(candleX, candleY, candleShape)
		u.restoreScreen()
		if sway != 0 {
			sway = max(sway-step, 0)
		}
		if u.getKey() != 0 {
			priestY = step + priestRaised
		}
		if u.escaped {
			return
		}
		priestY = max(priestY-step, priestRaised)
	}
	t.left, t.right, t.leftStep, t.rightStep = 0, 0, 1, 1
	ritual := func(hand int, moongate int, dot bool) {
		draw(0, 0, nightShape)
		if moongate >= 0 {
			draw(moongateX, moongate, moongateShape)
		}
		u.drawCrowds(lib, t.left, t.right, crowdNear)
		draw(priestX, priestRaised, priestShape)
		draw(handX, priestRaised+handY, hand)
		draw(avatarX, avatarY, avatarShape)
		draw(0, groundTop, groundShape)
		draw(candleX, candleY, candleShape)
		if dot {
			u.drawDot()
		}
	}
	untilKey := func(frame func()) bool {
		u.flushKeys()
		for {
			frame()
			u.drawPassage()
			u.restoreScreen()
			u.wander(3)
			if u.getKey() != 0 {
				break
			}
		}
		return !u.escaped
	}
	for _, p := range []int{5, 6, 7} {
		if p != 5 {
			u.showPassage(titlePassage + p)
		}
		if !untilKey(func() { ritual(daggerHandShape, -1, false) }) {
			return
		}
	}

	// the moongate rises
	u.showPassage(titlePassage + 8)
	step = randomBetween(2, 5)
	u.flushKeys()
	for moongateY, last := moongateDown, false; !last; {
		last = moongateY == moongateUp
		ritual(daggerHandShape, moongateY, true)
		u.drawPassage()
		u.restoreScreen()
		u.wander(3)
		if u.getKey() != 0 {
			moongateY = step + moongateUp
		}
		if u.escaped {
			return
		}
		moongateY = max(moongateY-step, moongateUp)
	}
	if u.waitKey(); u.escaped {
		return
	}
	ritual(daggerHandShape, moongateUp, true)
	u.restoreScreen()
	fill(u.back, 0)
	u.dissolve(u.back, u.screen, 0)

	// the companions
	for i, p := range []int{9, 10, 11, 12} {
		if i > 0 {
			u.fadeOut(u.screen, 0)
			fill(u.screen, 0)
			u.setPalette(&u.scenePalette)
		}
		u.showPassage(titlePassage + p)
		u.drawPassage()
		u.fadeIn(u.back, u.screen)
		if !u.waitTitleKey() {
			return
		}
	}
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	u.setPalette(&u.scenePalette)
	saved := *t
	t.left, t.right = 0, 0
	ritual(daggerHandShape, moongateUp, true)
	t.left, t.right = saved.left, saved.right
	u.dissolve(u.back, u.screen, 0)
	u.flushKeys()
	for range lingering / 2 {
		ritual(daggerHandShape, moongateUp, true)
		u.restoreScreen()
		u.wander(3)
		if u.getKey() != 0 {
			break
		}
		if u.escaped {
			return
		}
	}
	// the dagger is gone
	t.left, t.right = 0, 0
	ritual(emptyHandShape, moongateUp, true)
	t.left, t.right = saved.left, saved.right
	for range lingering/2 + 1 {
		u.restoreScreen()
	}
	u.showPassage(titlePassage + 13)
	if !untilKey(func() { ritual(emptyHandShape, moongateUp, false) }) {
		return
	}

	// the priest falls
	step = randomBetween(4, 10)
	u.flushKeys()
	sway = 0
	crowdY := func() int { return crowdFar - u.rand()%4 }
	for priestY, last := priestRaised, false; !last; {
		last = priestY == priestLow
		sway += u.rand()%9 - 3
		draw(0, 0, nightShape)
		draw(moongateX, moongateUp, moongateShape)
		u.drawCrowds(lib, t.left, t.right, crowdY())
		draw(sway+priestX, priestY, priestShape)
		draw(sway+handX, priestY+handY, emptyHandShape)
		draw(avatarX, avatarY, avatarShape)
		draw(0, groundTop, groundShape)
		draw(candleX, candleY, candleShape)
		u.restoreScreen()
		u.wander(4)
		if u.getKey() != 0 {
			priestY = priestLow - step
		}
		if u.escaped {
			return
		}
		priestY = min(priestY+step, priestLow)
	}

	// the companions come through the moongate
	companions := []int{-32, 122, 68, 122, 168, 202}
	arrived := []int{99, 40, 183, 53, 240, 62}
	companionShapes := [...]int{dupreShape, shaminoShape, ioloShape, avatarShape}
	drawCompanions := func(pos []int) {
		for i := 0; i < len(pos); i += 2 {
			draw(pos[i], pos[i+1], companionShapes[i/2])
		}
	}
	gate := func() {
		draw(0, 0, nightShape)
		draw(moongateX, moongateUp, moongateShape)
		u.drawCrowds(lib, t.left, t.right, crowdY())
	}
	step = randomBetween(3, 9)
	if !u.slide(step, companions, arrived, func() {
		gate()
		drawCompanions(companions)
		draw(avatarX, avatarY, avatarShape)
		draw(0, groundTop, groundShape)
		draw(candleX, candleY, candleShape)
	}) {
		return
	}
	untilKeyWander := func(frame func()) bool {
		u.flushKeys()
		for {
			frame()
			u.drawPassage()
			u.restoreScreen()
			u.wander(4)
			if u.getKey() != 0 {
				break
			}
		}
		return !u.escaped
	}
	u.showPassage(titlePassage + 14)
	if !untilKeyWander(func() {
		gate()
		drawCompanions(arrived)
		draw(avatarX, avatarY, avatarShape)
		draw(0, groundTop, groundShape)
		draw(candleX, candleY, candleShape)
	}) {
		return
	}

	// they carry the avatar to the moongate
	carried := []int{99, 40, 183, 53, 240, 62, avatarX, avatarY}
	toGate := []int{18, 142, 18, 142, 118, 122, 49, 138}
	if !u.slide(step, carried, toGate, func() {
		gate()
		drawCompanions(carried)
		draw(0, groundTop, groundShape)
	}) {
		return
	}
	for _, p := range []int{15, 16} {
		u.showPassage(titlePassage + p)
		if !untilKeyWander(func() {
			gate()
			draw(0, groundTop, groundShape)
		}) {
			return
		}
	}
	u.showPassage(moongateSinksPassage)
	step = randomBetween(1, 3)
	u.flushKeys()
	for moongateY, last := moongateUp, false; !last; {
		last = moongateY == moongateDown
		draw(0, 0, nightShape)
		draw(moongateX, moongateY, moongateShape)
		u.drawCrowds(lib, t.left, t.right, crowdY())
		draw(0, groundTop, groundShape)
		u.drawPassage()
		u.restoreScreen()
		u.wander(4)
		if u.getKey() != 0 {
			moongateY = moongateDown - step
		}
		if u.escaped {
			return
		}
		moongateY = min(moongateY+step, moongateDown)
	}
	stones := func() {
		draw(0, 0, nightShape)
		u.drawCrowds(lib, t.left, t.right, crowdY())
		draw(0, groundTop, groundShape)
	}
	for _, p := range []int{18, 19} {
		u.showPassage(titlePassage + p)
		if !untilKeyWander(stones) {
			return
		}
	}
	u.flushKeys()
	for range lingering / 2 {
		stones()
		u.restoreScreen()
		u.wander(4)
		if u.getKey() != 0 || u.escaped {
			break
		}
	}
}

// slide moves the positions pos towards to a few steps a frame, drawing each frame with
// frame; a key finishes the move. False at Escape.
func (u *intro) slide(steps int, pos, to []int, frame func()) bool {
	done := false
	u.flushKeys()
	for !done {
		if u.getKey() != 0 {
			for u.fade.step(pos, to) {
			}
			done = true
		}
		if u.escaped {
			return false
		}
		for i := 0; i < steps && !done; i++ {
			done = !u.fade.step(pos, to)
		}
		frame()
		u.restoreScreen()
		if len(pos) > 4 {
			u.wander(4)
		}
	}
	return true
}
