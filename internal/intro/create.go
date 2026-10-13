package intro

import "github.com/memmaker/u6-go/internal/sound"

// The character creation: name, sex and portrait on the vellum, the walk through the woods,
// and the gypsy's questions that set the avatar's attributes.

// character is the avatar being created.
type character struct {
	name                              [16]byte // D_5A30
	sex                               [8]byte  // D_5A46: "Male" or "Female"
	strength, dexterity, intelligence int      // D_5A40, D_5A42, D_5A44
	karma                             int      // D_0692, kept from one creation to the next
	portraitChoice                    int      // D_5A2E
	portrait                          int      // D_5A6A
	experience, hitPoints, level      int      // D_5A60, D_5A64, D_5A62
}

const (
	initialKarma     = 'K' // 75
	karmaPerHonesty  = 3
	startAttribute   = 15
	startExperience  = 370
	startHitPoints   = 90
	startLevel       = 3
	nameLength       = 13
	portraitsPerSex  = 6
	malePortraits    = 6
	vellumColour     = 49
	vellumTextColour = 61
)

// The vellum's layout during the creation.
const (
	creationLeft, creationRight = 28, 292
	creationTop, creationBottom = 87, 189
	nameMargin                  = 16
	nameInputX                  = 53
	sexInputX                   = 157
)

// toggleSex turns "M..." into "Female" and "F..." into "Male".
// C: seg_0263:055F (toggle_sex)
func (u *intro) toggleSex() {
	switch u.char.sex[0] {
	case 'M':
		strcpy(u.char.sex[:], "Female")
	case 'F':
		strcpy(u.char.sex[:], "Male")
	}
}

// askNameSex asks for the name and sex on the vellum; false at Escape.
// C: seg_0406:012A (AskNameSex)
func (u *intro) askNameSex(fadedIn *bool) bool {
	box := u.vellumBack
	box.left, box.top, box.right, box.bottom = creationLeft, creationTop, creationRight, creationBottom
	fill(box, vellumColour)
	box.right -= nameMargin
	box.left += nameMargin
	box.top += nameMargin
	box.bottom -= nameMargin
	win := textWindow{surf: &box, fg: vellumTextColour, bg: vellumColour, justify: justifyCenter}
	u.win = &win
	u.clearScreen()
	u.putsJustified("By what name shalt thou be called?\n\n")
	if !*fadedIn {
		u.fadeIn(u.vellumBack, u.vellum)
	} else {
		u.copyRect(u.vellumBack, u.vellum)
	}
	*fadedIn = true
	samePicture(u.screen, &box)
	row := win.y
	for ok := false; !ok; {
		win.x, win.y = box.left, row
		u.clearDown()
		win.x = nameInputX
		if !u.inputString(u.char.name[:], nameLength) {
			return false
		}
		win.x = box.left
		u.clearToEOL()
		win.justify = justifyCenter
		u.putsJustified(cString(u.char.name[:]) + "\n\n")
		u.putsJustified("And art thou Male, or Female?\n\n")
		for {
			win.x = sexInputX
			u.clearToEOL()
			ok = u.inputString(u.char.sex[:], 1)
			u.char.sex[0] = byte(toUpper(int(u.char.sex[0])))
			if u.char.sex[0] == 'M' || u.char.sex[0] == 'F' || !ok {
				break
			}
		}
	}
	u.toggleSex()
	u.toggleSex()
	return true
}

// The portrait page's buttons (VELLUM1.SHP) and keys.
const (
	sexButtonShape      = 3
	portraitButtonShape = 18
	continueButtonShape = 16
	abortButtonShape    = 4
	buttonRight         = 151
	portraitLeft        = 195
	portraitTop         = 118
	portraitRight       = 258
	portraitBottom      = 173
	nameBoxLeft         = 30
	nameBoxRight        = 291
	nameBoxTop          = 87
	nameBoxBottom       = 99
	nameRowOffset       = 2
)

var (
	buttonLefts   = [...]int{94, 62, 86, 80, 80}
	buttonTops    = [...]int{112, 129, 146, 163, 163}
	buttonBottoms = [...]int{127, 144, 161, 178, 178}
	buttonShapes  = [...]int{sexButtonShape, portraitButtonShape, continueButtonShape, abortButtonShape}
)

// portraitKeys are the keys of the buttons: Sex, Portrait, Continue, Abort, and Escape.
const portraitKeys = "SPCA" + string(rune(charEscape))

// choosePortrait shows the name and the portraits; true when the player continues, false
// to enter the name again.
// C: seg_0263:0987 (choose_portrait)
func (u *intro) choosePortrait(vellum, montage shapes) bool {
	page := u.vellumBack
	page.left, page.top, page.right, page.bottom = creationLeft, creationTop, creationRight, creationBottom
	pageOnScreen := page
	samePicture(u.screen, &pageOnScreen)
	portrait := page
	portrait.left, portrait.top, portrait.right, portrait.bottom = portraitLeft, portraitTop, portraitRight, portraitBottom
	nameBox := page
	nameBox.left, nameBox.right, nameBox.top, nameBox.bottom = nameBoxLeft, nameBoxRight, nameBoxTop, nameBoxBottom
	rule := nameBox
	rule.top, rule.bottom = nameBoxBottom-1, nameBoxBottom-1
	buttons := make([]surface, len(portraitKeys))
	for i := range buttons {
		buttons[i] = page
		buttons[i].left, buttons[i].right = buttonLefts[i], buttonRight
		buttons[i].top, buttons[i].bottom = buttonTops[i], buttonBottoms[i]
	}
	fill(page, vellumColour)
	fill(rule, vellumTextColour)
	win := textWindow{surf: &nameBox, x: nameBox.left, y: nameBox.top + nameRowOffset,
		fg: vellumTextColour, bg: vellumColour, justify: justifyCenter}
	u.win = &win
	u.putsJustified(cString(u.char.name[:]))
	for i, shape := range buttonShapes {
		vellum.draw(page, buttons[i].left, buttons[i].top, shape)
	}
	for {
		u.mouseHide()
		fill(portrait, vellumColour)
		c := &u.char
		c.portrait = malePortraits + c.portraitChoice
		if c.sex[0] == 'F' {
			c.portrait = c.portraitChoice
		}
		montage.draw(portrait, portrait.left, portrait.top, c.portrait)
		u.copyRect(page, pageOnScreen)
		u.mouseShow()
		switch portraitKeys[u.menuSelect(buttons, portraitKeys)] {
		case 'S':
			u.toggleSex()
		case 'P':
			c.portraitChoice = (c.portraitChoice + 1) % portraitsPerSex
		case 'C':
			return true
		default:
			return false
		}
	}
}

// Pictures of WOODS.SHP and GYPSY.SHP.
const (
	woodsShape       = 0
	gypsyWagonShape  = 1
	woodsScroll      = 191
	gypsyTableShape  = 0
	gypsyFaceShape   = 198
	gypsyPortalShape = 199
	creationPassage  = 100
	gameInProgress   = 998
	portalColour     = 117
	gypsyMillis      = 2000
	finalNoiseLeft   = 116
	finalNoiseRight  = 203
	finalNoiseTop    = 35
	finalNoiseBottom = 127
)

// createCharacter runs the character creation; true when the avatar was saved and the
// game can start.
// C: seg_0406:1269 (CharCreation)
func (u *intro) createCharacter() (bool, error) {
	saveWin := u.win
	defer func() { u.win = saveWin }()
	u.vellumBack = u.vellum
	samePicture(u.back, &u.vellumBack)
	fill(u.vellumBack, 0)
	saved := surface{right: u.vellum.right - u.vellum.left, bottom: u.vellum.bottom - u.vellum.top}
	newPicture(&saved)
	vellum := u.mustShapes("VELLUM1.SHP")
	montage := u.mustShapes("MONTAGE.SHP")
	u.copyRect(u.vellum, saved)
	u.fadeOut(u.vellum, 0)
	fill(u.vellum, 0)
	u.setPalette(&u.menuPalette)
	vellum.draw(u.vellumBack, u.vellumBack.left, u.vellumBack.top, vellumShape)
	u.char.portraitChoice = 0
	fadedIn := false
	u.mouseShow()
	u.playMusic(sound.SongCreate)
	u.menuDrawn = false
	for {
		u.mouseHide()
		if !u.askNameSex(&fadedIn) {
			u.menuDrawn = true
			break
		}
		u.mouseShow()
		if u.choosePortrait(vellum, montage) {
			break
		}
	}
	if u.menuDrawn {
		u.fadeOut(u.vellum, 0)
		fill(u.vellum, 0)
		u.setPalette(&u.menuPalette)
		u.fadeIn(saved, u.vellum)
		u.playMusic(sound.SongStop)
		return false, nil
	}
	u.mouseHide()
	u.calibrateTimer()

	// the woods
	woods := u.mustShapes("WOODS.SHP")
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	u.loadPalette(4)
	woods.draw(u.back, 0, 0, woodsShape)
	u.fadeIn(u.back, u.screen)
	u.showPassage(creationPassage)
	for {
		woods.draw(u.back, 0, 0, woodsShape)
		u.drawPassage()
		u.copyRect(u.back, u.screen)
		if u.pollInput() != 0 {
			break
		}
	}
	step := randomBetween(2, 10)
	for si, last := 0, false; !last; {
		last = si == woodsScroll
		woods.draw(u.back, u.back.left-si, 0, woodsShape)
		woods.draw(u.back, u.back.left+screenWidth-si, 0, gypsyWagonShape)
		u.waitRetrace()
		u.copyRect(u.back, u.screen)
		if u.pollInput() != 0 {
			si = woodsScroll - step
		}
		si = min(si+step, woodsScroll)
	}
	wagon := func() {
		woods.draw(u.back, u.back.left-woodsScroll, 0, woodsShape)
		woods.draw(u.back, u.back.left+screenWidth-woodsScroll, 0, gypsyWagonShape)
	}
	for _, p := range []int{1, 2} {
		u.showPassage(creationPassage + p)
		for {
			wagon()
			u.drawPassage()
			u.copyRect(u.back, u.screen)
			if u.pollInput() != 0 {
				break
			}
		}
	}
	wagon()
	u.copyRect(u.back, u.screen)
	u.fadeOut(u.screen, 0)
	u.menuDrawn = false
	fill(u.screen, 0)
	fill(u.back, 0)

	// the gypsy
	u.loadPalette(5)
	u.gypsy.shapes = u.mustShapes("GYPSY.SHP")
	u.gypsyQuestions()

	// the portal
	g := &u.gypsy
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	u.loadPalette(6)
	g.noise = u.back
	g.noise.left, g.noise.right, g.noise.top, g.noise.bottom = finalNoiseLeft, finalNoiseRight, finalNoiseTop, finalNoiseBottom
	face := func() {
		u.noiseFill()
		g.shapes.draw(u.back, 0, 0, gypsyFaceShape)
	}
	face()
	u.showPassage(creationPassage + 34)
	u.fadeIn(u.back, u.screen)
	for {
		face()
		u.drawPassage()
		u.copyRect(u.back, u.screen)
		if u.pollInput() != 0 {
			break
		}
	}
	face()
	u.copyRect(u.back, u.screen)
	u.fadeOut(u.screen, portalColour)
	fill(u.screen, portalColour)
	u.loadPalette(6)
	g.shapes.draw(u.back, 0, 0, gypsyPortalShape)
	u.showPassage(creationPassage + 35)
	u.fadeIn(u.back, u.screen)
	u.delayMilliseconds(gypsyMillis)
	u.drawPassage()
	u.copyRect(u.back, u.screen)
	for _, p := range []int{36, 37} {
		u.showPassage(creationPassage + p)
		u.waitInput()
		g.shapes.draw(u.back, 0, 0, gypsyPortalShape)
		u.drawPassage()
		u.copyRect(u.back, u.screen)
	}
	u.waitInput()
	g.shapes.draw(u.back, 0, 0, gypsyPortalShape)
	u.copyRect(u.back, u.screen)
	u.fadeOut(u.screen, 0)
	fill(u.screen, 0)
	fill(u.back, 0)
	u.setPalette(&u.menuPalette)
	c := &u.char
	c.experience, c.hitPoints, c.level = startExperience, startHitPoints, startLevel
	inProgress, err := u.saveObjlist()
	if err != nil {
		return false, err
	}
	if inProgress {
		u.showPassage(gameInProgress)
		u.drawPassage()
		u.copyRect(u.back, u.screen)
		u.waitInput()
		fill(u.back, 0)
		u.copyRect(u.back, u.screen)
	}
	u.playMusic(sound.SongStop)
	return !inProgress, nil
}
