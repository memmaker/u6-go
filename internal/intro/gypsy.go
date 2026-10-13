package intro

// The gypsy's questions: eight vials of the virtues; for each answer she pours the chosen
// virtue's vial into the balance, whose liquid shows the colours poured so far.

const (
	virtues        = 8
	vialFull       = 3
	noiseColours   = 32
	noisePattern   = 16
	noiseUnset     = 255
	maxPoured      = 16
	gypsyTop       = -20 // D_06A5, the gypsy pictures' offset
	armShapes      = 1   // D_07D7
	handShapes     = 9   // D_07D9
	vialShapes     = 66  // D_07DB
	tableRow       = 86
	steps          = 17
	stepCycle      = 16
	noiseLeft      = 147
	noiseRight     = 172
	noiseStartRow  = 127
	noiseRise      = 4
	answerA        = 'A'
	answerB        = 'B'
	firstQuestion  = 105
	finalQuestion  = 133
	introQuestion1 = 103
	introQuestion2 = 104
	codeBytes      = 2048
)

// virtueOffsetX moves the hands of the vials on the left and right of the table.
var virtueOffsetX = [virtues]int{10, 10, 10, 10, -13, -13, -13, -13}

// virtueColours are the colours of the virtues' liquids.
var virtueColours = [virtues]int{239, 14, 231, 103, 228, 5, 15, 219}

// questions[a][b] is the passage that asks to choose between virtues a and b.
var questions = [virtues][virtues]int{
	{-1, 105, 106, 107, 108, 109, 110, 111},
	{105, -1, 112, 113, 114, 115, 116, 117},
	{106, 112, -1, 118, 119, 120, 121, 122},
	{107, 113, 118, -1, 123, 124, 125, 126},
	{108, 114, 119, 123, -1, 127, 128, 129},
	{109, 115, 120, 124, 127, -1, 130, 131},
	{110, 116, 121, 125, 128, 130, -1, 132},
	{111, 117, 122, 126, 129, 131, 132, -1},
}

// answerA and answerB are the virtues of the answers of question firstQuestion+n.
var (
	virtueOfA = [...]int{0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 5, 5, 6}
	virtueOfB = [...]int{1, 2, 3, 4, 5, 6, 7, 2, 3, 4, 5, 6, 7, 3, 4, 5, 6, 7, 4, 5, 6, 7, 5, 6, 7, 6, 7, 7}
)

// What each virtue adds to strength, dexterity and intelligence.
var (
	virtueStrength     = [virtues]int{0, 0, 2, 0, 1, 1, 1, 0}
	virtueDexterity    = [virtues]int{0, 2, 0, 1, 1, 0, 1, 0}
	virtueIntelligence = [virtues]int{2, 0, 0, 1, 0, 1, 1, 0}
)

const honesty = 7

// The pictures of the parts of the pouring hand: per kind of part, per vial level (full,
// two thirds, a third) and per virtue, the shape (handPartShapes) and its offset.
var (
	handPartShapes = [4][3][virtues]int{
		{{58, 79, 100, 124, 40, 126, 32, 52}, {54, 75, 96, 118, 38, 120, 28, 48}, {54, 75, 96, 118, 38, 120, 28, 48}},
		{{60, 81, 102, 127, 20, 95, 34, 54}, {56, 77, 98, 121, 39, 123, 30, 50}, {56, 77, 98, 121, 39, 123, 30, 50}},
		{{64, 83, 104, 129, 41, 131, 36, 56}, {64, 83, 104, 129, 41, 131, 36, 56}, {64, 83, 104, 129, 41, 131, 36, 56}},
		{{0, 3, 6, 9, 22, 25, 28, 31}, {1, 4, 7, 10, 23, 26, 29, 32}, {2, 5, 8, 11, 24, 27, 30, 33}},
	}
	handPartX = [4][3][virtues]int{
		{{4, 4, 4, 4, 20, 20, 20, 20}, {4, 4, 4, 4, 19, 19, 19, 19}, {4, 4, 4, 4, 19, 19, 19, 19}},
		{{4, 4, 4, 4, 23, 23, 23, 23}, {3, 3, 3, 3, 23, 23, 23, 23}, {3, 3, 3, 3, 23, 23, 23, 23}},
		{{3, 3, 3, 3, 27, 27, 27, 27}, {3, 3, 3, 3, 27, 27, 27, 27}, {3, 3, 3, 3, 27, 27, 27, 27}},
		{{3, 3, 3, 3, 3, 3, 3, 3}, {3, 3, 3, 3, 3, 3, 3, 3}, {3, 3, 3, 3, 3, 3, 3, 3}},
	}
	handPartY = [4][3]int{{27, 19, 19}, {22, 14, 14}, {14, 14, 14}, {11, 11, 11}}
)

// pour is the pouring animation: per row a value per virtue (see the pour* names).
var pour = [22][virtues]int{
	{2, 5, 8, 11, 24, 27, 30, 33},
	{93, 93, 93, 93, 172, 172, 172, 172},
	{41, 62, 83, 104, 200, 221, 242, 263},
	{29, 50, 71, 92, 202, 223, 244, 264},
	{107, 107, 107, 107, 170, 170, 170, 170},
	{107, 107, 107, 107, 170, 170, 170, 170},
	{109, 109, 109, 109, 168, 168, 168, 168},
	{112, 112, 112, 112, 165, 165, 165, 165},
	{40, 40, 40, 40, 40, 40, 40, 40},
	{40, 40, 40, 40, 40, 40, 40, 40},
	{41, 41, 41, 41, 41, 41, 41, 41},
	{33, 33, 33, 33, 33, 33, 33, 33},
	{11, 11, 11, 11, 14, 14, 14, 14},
	{12, 12, 12, 12, 13, 13, 13, 13},
	{9, 9, 9, 9, 17, 17, 17, 17},
	{9, 9, 9, 9, 17, 17, 17, 17},
	{10, 10, 10, 10, 18, 18, 18, 18},
	{16, 16, 16, 16, 46, 46, 46, 46},
	{15, 15, 15, 15, 37, 37, 37, 37},
	{0, 0, 0, 0, 5, 5, 5, 5},
	{1, 1, 1, 1, 4, 4, 4, 4},
	{2, 2, 2, 2, 3, 3, 3, 3},
}

// Rows of pour.
const (
	pourLevelShape = 0  // the vial at its level
	pourArmX       = 1  // the arm's x
	pourVialX      = 2  // the vial's place on the table
	pourLiftedX    = 3  // the vial lifted
	pourTiltX      = 4  // 4..7: the vial tilted further and further
	pourTiltY      = 8  // 8..11
	pourHandShape  = 12 // 12..18: the hand with the vial at each stage
	pourArmShape   = 19 // 19..21: the arm reaching further
)

// The vial poured, at the levels left after pouring (3 - level), and the balance.
var (
	pouredShape = [3][virtues]int{{43, 67, 86, 107, 36, 116, 25, 44}, {44, 68, 87, 108, 35, 115, 24, 43}, {45, 69, 88, 109, 34, 114, 23, 42}}
	pouredX     = [3][virtues]int{{146, 146, 146, 146, 169, 169, 169, 169}, {145, 145, 145, 145, 169, 169, 169, 169}, {148, 148, 148, 148, 170, 170, 170, 170}}
	pouredY     = [3]int{50, 55, 64}
)

const (
	balanceX, balanceY = 142, 76
	balanceShape       = 7
	vialLevelX         = 3
	vialLevelY         = 97
	armReachY          = 41
	liftX, liftY       = -6, -45
	liftY1, liftY2     = 37, 41
	firstRightVirtue   = 6
	choiceShapeA       = 6
	choiceShapeB       = 7
	choiceAX, choiceBX = 271, 288
	choiceY            = 41
)

// gypsyState is the gypsy's table.
type gypsyState struct {
	shapes  shapes             // D_5ADC, GYPSY.SHP
	levels  [virtues]int       // D_5B18: thirds left in each vial
	order   [virtues]int       // D_5B2A
	colours [noiseColours]byte // D_5B7A: the balance's liquid, kept from one creation to the next
	poured  [maxPoured]int     // D_5B3A
	count   int                // D_5B28
	pours   int                // D_5B16
	noise   surface            // D_5AE6
	pouring bool               // D_0693
}

// addColour pours a colour into the balance: its liquid rises and shows the colours.
// C: seg_0406:033B (AddColor)
func (u *intro) addColour(c int) {
	g := &u.gypsy
	g.noise.top -= noiseRise
	g.poured[g.count] = c
	g.count++
	for k := 1; k <= g.count; k++ {
		for i := 0; noisePattern/g.count > i; i++ {
			g.colours[i*k] = byte(g.poured[k-1])
		}
	}
}

// noiseFill fills the balance's liquid with the colours in a random pattern. The original
// took the pattern from bytes of its own code; the port takes them from a pseudo random
// stream starting at the same random offset.
// C: seg_0406:038C (NoiseFill)
func (u *intro) noiseFill() {
	g := &u.gypsy
	r := borlandRand{seed: uint32(u.rand() % codeBytes)}
	for y := g.noise.top; y <= g.noise.bottom; y++ {
		for x := g.noise.left; x <= g.noise.right; x++ {
			if g.noise.inPicture(x, y) {
				g.noise.pixels[g.noise.at(x, y)] = g.colours[r.next()%noisePattern]
			}
		}
	}
}

// drawHandPart draws the part of the hand that holds vial virtue.
// C: seg_0406:05D3 (DrawPart)
func (u *intro) drawHandPart(virtue, part, x, y int) {
	g := &u.gypsy
	kind := 0
	switch part {
	case 15, 37:
		return
	case 9, 17:
		kind = 0
	case 10, 18:
		kind = 1
	case 16, 46:
		kind = 2
	case 11, 12, 13, 14:
		kind = 3
	}
	level := g.levels[virtue] - 1
	base := vialShapes
	if virtue > 5 && kind != 3 {
		base = handShapes
	}
	if level > -1 && level < 3 {
		g.shapes.draw(u.back, handPartX[kind][level][virtue]+x+virtueOffsetX[virtue],
			handPartY[kind][level]+y+gypsyTop, handPartShapes[kind][level][virtue]+base)
	}
}

// drawStep draws step n of pouring vial virtue (redraw: a repeated frame of the step);
// false when the step shows nothing new and is skipped.
// C: seg_0406:06F8 (DrawStep)
func (u *intro) drawStep(virtue, n int, redraw bool) bool {
	g := &u.gypsy
	skip := false
	g.pouring = false
	drawArm := func(row int) {
		g.shapes.draw(u.back, pour[pourArmX][virtue], gypsyTop+armReachY, pour[row][virtue]+armShapes)
	}
	drawLift := func(row int) {
		g.shapes.draw(u.back, virtueOffsetX[virtue]+pour[pourVialX][virtue]+liftX, tableRow+gypsyTop+liftY, pour[row][virtue]+armShapes)
	}
	var x, y, part int
	n %= stepCycle
	switch n {
	case 0:
		x, y, part = pour[pourVialX][virtue], tableRow, pour[pourHandShape][virtue]
	case 1, 15:
		drawLift(pourArmShape)
		x, y, part = pour[pourVialX][virtue], tableRow, pour[pourHandShape+1][virtue]
	case 2, 14:
		drawLift(pourArmShape + 1)
		x, y, part = pour[pourVialX][virtue], tableRow-liftY1, pour[pourHandShape+1][virtue]
	case 3, 13:
		drawLift(pourArmShape + 1)
		x, y, part = pour[pourLiftedX][virtue], tableRow-liftY2, pour[pourHandShape+2][virtue]
	case 4, 12:
		drawArm(pourArmShape + 2)
		x, y, part = pour[pourTiltX][virtue], pour[pourTiltY][virtue], pour[pourHandShape+3][virtue]
	case 5, 11:
		drawArm(pourArmShape + 2)
		x, y, part = pour[pourTiltX+1][virtue], pour[pourTiltY+1][virtue], pour[pourHandShape+4][virtue]
		if n == 11 && !redraw {
			g.levels[virtue]--
		}
		if g.levels[virtue] == 2 && n == 11 {
			skip = true
		}
		if g.levels[virtue] == 3 {
			g.pouring = true
		}
	case 6, 10:
		drawArm(pourArmShape + 2)
		x, y, part = pour[pourTiltX+2][virtue], pour[pourTiltY+2][virtue], pour[pourHandShape+5][virtue]
		if (g.levels[virtue] == 1 || g.levels[virtue] == 2) && n == 10 {
			skip = true
		}
		if g.levels[virtue] == 2 {
			g.pouring = true
		}
		if g.levels[virtue] == 3 {
			skip = true
		}
	case 7, 9:
		drawArm(pourArmShape + 2)
		x, y, part = pour[pourTiltX+3][virtue], pour[pourTiltY+3][virtue], pour[pourHandShape+6][virtue]
		if g.levels[virtue] == 1 && n == 9 || g.levels[virtue] == 2 || g.levels[virtue] == 3 {
			skip = true
		}
	case 8:
		drawArm(pourArmShape + 2)
		x, y, part = pour[pourTiltX+3][virtue], pour[pourTiltY+3][virtue], pour[pourHandShape+6][virtue]
		if g.levels[virtue] == 1 {
			g.pouring = true
		}
		if g.levels[virtue] == 2 || g.levels[virtue] == 3 {
			skip = true
		}
	}
	if n > 3 && n < 9 || n > 8 && n < 13 {
		x -= virtueOffsetX[virtue]
	}
	if skip {
		return false
	}
	u.drawHandPart(virtue, part, x, y)
	g.shapes.draw(u.back, virtueOffsetX[virtue]+x, gypsyTop+y, handShapes+part)
	for v := range virtues {
		if v == virtue {
			continue
		}
		if g.levels[v] > 0 {
			g.shapes.draw(u.back, virtueOffsetX[v]+pour[pourVialX][v]+vialLevelX, gypsyTop+vialLevelY,
				pour[pourLevelShape][v]+vialShapes+g.levels[v]-vialFull)
		}
		g.shapes.draw(u.back, virtueOffsetX[v]+pour[pourVialX][v], tableRow+gypsyTop, pour[pourHandShape][v]+handShapes)
	}
	if n < 9 && g.pouring && !redraw {
		u.addColour(virtueColours[virtue])
		g.pours++
	}
	if g.pours != 0 {
		u.noiseFill()
	}
	g.shapes.draw(u.back, balanceX, gypsyTop+balanceY, handShapes+balanceShape)
	if g.pouring {
		left := vialFull - g.levels[virtue]
		base := vialShapes
		if virtue > 5 {
			base = handShapes
		}
		g.shapes.draw(u.back, pouredX[left][virtue], pouredY[left]+gypsyTop, pouredShape[left][virtue]+base)
	}
	return true
}

// shuffle mixes the first n entries of a.
// C: seg_0406:0D73 (Shuffle)
func (u *intro) shuffle(a []int, n int) {
	for k := u.rand()%(n*n) + n; k != 0; k-- {
		i := u.rand() % n
		j := u.rand() % n
		a[i], a[j] = a[j], a[i]
	}
}

// The answer buttons A and B.
var (
	answerAButton = surface{left: 278, right: 293, top: 174, bottom: 188}
	answerBButton = surface{left: 295, right: 310, top: 174, bottom: 188}
)

// waitChoice shows the gypsy's table until a key or a click: the key, 'A' or 'B' for a
// click on a button, 1 for a click elsewhere.
// C: seg_0406:0DDE (WaitChoice)
func (u *intro) waitChoice() int {
	g := &u.gypsy
	for {
		g.shapes.draw(u.back, 0, gypsyTop, gypsyTableShape)
		u.mousePoll()
		u.drawStep(0, 0, true)
		u.mousePoll()
		u.drawPassage()
		u.mousePoll()
		u.copyRect(u.back, u.screen)
		u.mousePoll()
		if u.mouseButton() {
			u.waitButtonsUp()
			switch {
			case u.mouseInside(answerAButton):
				return answerA
			case u.mouseInside(answerBButton):
				return answerB
			}
			return 1
		}
		if u.kbhit() {
			return toUpper(u.getch())
		}
	}
}

// chooseAB asks question q and returns the virtue of the answer.
// C: seg_0406:0F99 (ChooseAB)
func (u *intro) chooseAB(q int) int {
	g := &u.gypsy
	g.shapes.draw(u.passageText, choiceAX, choiceY, armShapes+choiceShapeA)
	g.shapes.draw(u.passageText, choiceBX, choiceY, armShapes+choiceShapeB)
	for {
		switch u.waitChoice() {
		case answerA:
			return virtueOfA[q-firstQuestion]
		case answerB:
			return virtueOfB[q-firstQuestion]
		}
	}
}

// animate adds the virtue's attributes and pours its vial.
// C: seg_0406:1040 (Animate)
func (u *intro) animate(virtue int) {
	g := &u.gypsy
	c := &u.char
	c.strength += virtueStrength[virtue]
	c.dexterity += virtueDexterity[virtue]
	c.intelligence += virtueIntelligence[virtue]
	if virtue == honesty {
		c.karma += karmaPerHonesty
	}
	repeats := randomBetween(3, 0)
	table := func() { g.shapes.draw(u.back, 0, gypsyTop, gypsyTableShape) }
	u.mousePoll()
	table()
	u.mousePoll()
	for n := range steps {
		if !u.drawStep(virtue, n, false) {
			continue
		}
		u.mousePoll()
		u.drawPassage()
		u.mousePoll()
		u.copyRect(u.back, u.screen)
		u.mousePoll()
		for range repeats {
			table()
			u.mousePoll()
			u.drawStep(virtue, n, true)
			u.mousePoll()
			u.drawPassage()
			u.mousePoll()
			u.copyRect(u.back, u.screen)
			u.mousePoll()
		}
		table()
		u.mousePoll()
	}
}

// gypsyQuestions sets up the table and asks the seven questions.
// (part of CharCreation)
func (u *intro) gypsyQuestions() {
	g := &u.gypsy
	c := &u.char
	c.strength, c.dexterity, c.intelligence = startAttribute, startAttribute, startAttribute
	for i := range virtues {
		g.order[i] = i
		g.levels[i] = vialFull
		g.colours[i] = noiseUnset
	}
	g.pours, g.count = 0, 0
	g.noise = u.back
	g.noise.left, g.noise.right = noiseLeft, noiseRight
	g.noise.top, g.noise.bottom = gypsyTop+noiseStartRow, gypsyTop+noiseStartRow
	g.shapes.draw(u.back, 0, gypsyTop, gypsyTableShape)
	u.drawStep(0, 0, true)
	u.fadeIn(u.back, u.screen)
	u.showPassage(introQuestion1)
	u.waitChoice()
	u.showPassage(introQuestion2)
	u.waitChoice()
	u.mouseShow()
	ask := func(a, b int) int {
		q := questions[a][b]
		u.showPassage(q)
		v := u.chooseAB(q)
		u.animate(v)
		return v
	}
	u.shuffle(g.order[:], virtues)
	for i := range 4 {
		g.order[i] = ask(g.order[i*2], g.order[i*2+1])
	}
	u.shuffle(g.order[:], 4)
	for i := range 2 {
		g.order[i] = ask(g.order[i*2], g.order[i*2+1])
	}
	ask(g.order[0], g.order[1])
	u.mouseHide()
	u.showPassage(finalQuestion)
	u.waitChoice()
}
