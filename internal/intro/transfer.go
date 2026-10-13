package intro

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"

	"github.com/memmaker/u6-go/internal/sound"
)

// The transfer of an Ultima IV or V character: its saved game is read, the character
// converted, compared with the Ultima VI avatar it becomes, and saved like a created one.
//
// The original asked for the drive of the Ultima V Britannia or Ultima IV Player disk and
// looked for PARTY.SAV (Ultima IV) or SAVED.GAM (Ultima V) there and in its \ULTIMA
// directory. Here the saves are looked for in the game's directory, its ULTIMA directory, and
// the directories of the other goplay games next to it (u4, u5).

// The games a character can come from.
const (
	noGame  = 0
	ultima4 = 4
	ultima5 = 5
)

// The saved games of Ultima IV and V and where their first character is.
const (
	ultima4Save        = "PARTY.SAV"
	ultima5Save        = "SAVED.GAM"
	ultima4CharacterAt = 8
	ultima5CharacterAt = 2
	transferSubdir     = "ULTIMA"
)

// An Ultima IV character record (offsets from its start).
const (
	u4HitPoints    = 0
	u4MaxHitPoints = 2
	u4Experience   = 4
	u4Strength     = 6
	u4Dexterity    = 8
	u4Intelligence = 10
	u4Magic        = 12
	u4Name         = 20
	u4Sex          = 36
	u4Class        = 37
	u4Virtues      = 320 + 6 // the party's eight virtues; all 0 for an Avatar
	u4VirtueCount  = 8
	u4MaxAttribute = 99
	u4Classes      = 8
	hitPointsPerLv = 100
)

// An Ultima V character record.
const (
	u5Name         = 0
	u5Sex          = 9
	u5Strength     = 12
	u5Dexterity    = 13
	u5Intelligence = 14
	u5Magic        = 15
	u5HitPoints    = 16
	u5MaxHitPoints = 18
	u5Experience   = 20
	u5Level        = 22
	u5MaxAttribute = 30
	u5RecordSize   = 23
)

const (
	maleSex           = 11 // the sex byte of a male character; anything else is female
	sourceNameBytes   = 8
	maxSourceValue    = 9999
	avatarClass       = 8
	experienceDivisor = 10
	experiencePerLv   = 1000
	hitPointsPerLevel = 30
)

var classNames = [...]string{"Mage", "Bard", "Fighter", "Druid", "Tinker", "Paladin", "Ranger", "Shepherd", "Avatar"}

// importedCharacter is the character as the old game had it.
// C: D_5944..D_597C
type importedCharacter struct {
	name, sex, class                  string
	strength, dexterity, intelligence int
	magic, experience, level          int
	hitPoints, maxHitPoints           int
}

func word(b []byte, at int) int { return int(int16(binary.LittleEndian.Uint16(b[at:]))) }

func sexName(b byte) string {
	if b == maleSex {
		return "Male"
	}
	return "Female"
}

// loadUltima4 reads the first character of PARTY.SAV; false when it is not a valid one.
// C: seg_0263:0000 (U4_load)
func loadUltima4(save []byte) (importedCharacter, bool) {
	if len(save) < ultima4CharacterAt+u4Virtues+2*u4VirtueCount {
		return importedCharacter{}, false
	}
	r := save[ultima4CharacterAt:]
	class := int(r[u4Class])
	avatar := true
	for v := range u4VirtueCount {
		avatar = avatar && word(r, u4Virtues+2*v) == 0
	}
	if avatar {
		class = avatarClass
	}
	maxHP := int(binary.LittleEndian.Uint16(r[u4MaxHitPoints:]))
	c := importedCharacter{
		name: cString(r[u4Name : u4Name+sourceNameBytes]), sex: sexName(r[u4Sex]),
		strength: word(r, u4Strength), dexterity: word(r, u4Dexterity), intelligence: word(r, u4Intelligence),
		experience: word(r, u4Experience), level: maxHP / hitPointsPerLv,
		hitPoints: word(r, u4HitPoints), maxHitPoints: maxHP, magic: word(r, u4Magic),
	}
	if class <= avatarClass {
		c.class = classNames[class]
	}
	invalid := c.strength > u4MaxAttribute || c.dexterity > u4MaxAttribute || c.intelligence > u4MaxAttribute ||
		c.hitPoints > maxSourceValue || c.maxHitPoints > maxSourceValue || c.experience > maxSourceValue ||
		class >= u4Classes && !avatar
	return c, !invalid && c.name != "" // fix: the original transferred invalid and empty characters
}

// loadUltima5 reads the first character of SAVED.GAM.
// C: seg_0263:013D (U5_load)
func loadUltima5(save []byte) (importedCharacter, bool) {
	if len(save) < ultima5CharacterAt+u5RecordSize {
		return importedCharacter{}, false
	}
	r := save[ultima5CharacterAt:]
	c := importedCharacter{
		name: cString(r[u5Name : u5Name+sourceNameBytes]), sex: sexName(r[u5Sex]), class: classNames[avatarClass],
		strength: int(r[u5Strength]), dexterity: int(r[u5Dexterity]), intelligence: int(r[u5Intelligence]),
		magic: int(r[u5Magic]), hitPoints: word(r, u5HitPoints), maxHitPoints: word(r, u5MaxHitPoints),
		experience: word(r, u5Experience), level: int(r[u5Level]),
	}
	invalid := c.strength > u5MaxAttribute || c.dexterity > u5MaxAttribute || c.intelligence > u5MaxAttribute ||
		c.hitPoints > maxSourceValue || c.maxHitPoints > maxSourceValue || c.experience > maxSourceValue
	return c, !invalid && c.name != ""
}

// convertAttribute scales an Ultima IV attribute (up to 99) to Ultima VI's.
// C: seg_0263:01FD (convert_stat)
func convertAttribute(v int) int {
	switch {
	case v < 10:
		return v
	case v >= 30:
		return (v-30)/4 + 20
	}
	return (v-9)/2 + 10
}

// convert makes the avatar of the imported character: Ultima IV attributes are scaled,
// Ultima V's kept; the level grows with each doubling of a thousand experience points.
// C: seg_0263:0233 (U4_convert), seg_0263:02D4 (U5_convert)
func (u *intro) convert(from importedCharacter, game int) {
	c := &u.char
	strcpy(c.name[:], from.name)
	c.strength, c.dexterity, c.intelligence = from.strength, from.dexterity, from.intelligence
	if game == ultima4 {
		c.strength = convertAttribute(from.strength)
		c.dexterity = convertAttribute(from.dexterity)
		c.intelligence = convertAttribute(from.intelligence)
	}
	strcpy(c.sex[:], from.sex)
	c.experience = from.experience / experienceDivisor
	c.level = 1
	for k := from.experience / experiencePerLv; k > 0; k >>= 1 {
		c.level++
	}
	c.hitPoints = c.level * hitPointsPerLevel
}

// transferDirs are the directories searched for an old saved game.
func (u *intro) transferDirs() []string {
	return []string{
		u.saveDir, filepath.Join(u.saveDir, transferSubdir),
		filepath.Join(u.saveDir, "..", "u4"), filepath.Join(u.saveDir, "..", "u5"),
	}
}

// findSave looks for an Ultima IV, then an Ultima V character, as the original did on the
// chosen drive.
func (u *intro) findSave() (importedCharacter, int) {
	for _, game := range []int{ultima4, ultima5} {
		name, load := ultima4Save, loadUltima4
		if game == ultima5 {
			name, load = ultima5Save, loadUltima5
		}
		for _, dir := range u.transferDirs() {
			b, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if c, ok := load(b); ok {
				return c, game
			}
		}
	}
	return importedCharacter{}, noGame
}

// The transfer pages' layout on the vellum.
const (
	transferLeft, transferRight   = 28, 292
	transferTop, transferBottom   = 87, 189
	headerLeft, headerRight       = 30, 291
	headerTop, headerBottom       = 87, 99
	statsTop, statsBottom         = 99, 174
	labelsRightMargin             = 214
	sourceColumn                  = 53
	targetColumn                  = 134
	columnGap                     = 81
	transferButtonTop             = 174
	transferButtonBottom          = 189
	findHeaderTop, findBodyTop    = 89, 100
	findTextTop                   = findHeaderTop + 32
	sourceTextColour              = 2
	targetTextColour              = 9
	nameButtonShape               = 2
	transferButtonShape           = 1
	textRowOffset                 = 2
	findContinueLeft              = 31
	findButtonsWidth              = 65
	findAbortLeft, findAbortRight = 221, 292
)

var (
	transferButtonLefts  = [...]int{28, 95, 154, 221, 221}
	transferButtonRights = [...]int{93, 152, 219, 292, 292}
	transferButtonShapes = [...]int{nameButtonShape, sexButtonShape, continueButtonShape, abortButtonShape}
)

// transferKeys are the comparison page's buttons: Name, Sex, Continue, Abort and Escape.
const transferKeys = "NSCA" + string(rune(charEscape))

// findKeys are the search page's buttons: Continue, Abort and Escape.
const findKeys = "CA" + string(rune(charEscape))

// vellumPage is the vellum's page on the back picture, and the same on the screen.
func (u *intro) vellumPage() (back, front surface) {
	back = u.back
	back.left, back.right, back.top, back.bottom = transferLeft, transferRight, transferTop, transferBottom
	front = back
	samePicture(u.screen, &front)
	return back, front
}

func (u *intro) pageRect(left, right, top, bottom int) surface {
	r := u.back
	r.left, r.right, r.top, r.bottom = left, right, top, bottom
	return r
}

// findCharacter shows what was found of an old game; the game (ultima4, ultima5) when the
// player continues, noGame to give up.
// C: seg_0263:0E40 (find_character)
func (u *intro) findCharacter(vellum shapes, fadedIn *bool) (importedCharacter, int) {
	page, onScreen := u.vellumPage()
	fill(page, vellumColour)
	header := u.pageRect(headerLeft, transferRight, findHeaderTop, findBodyTop)
	body := u.pageRect(headerLeft, transferRight, findBodyTop, transferBottom)
	win := textWindow{surf: &header, x: header.left, y: header.top, fg: vellumTextColour, bg: vellumColour, justify: justifyCenter}
	u.win = &win
	u.putsJustified("TRANSFER A CHARACTER")
	fill(u.pageRect(headerLeft, transferRight-1, findBodyTop-textRowOffset, findBodyTop-textRowOffset), vellumTextColour)
	from, game := u.findSave()
	win.surf, win.x, win.y = &body, body.left, findTextTop
	keys := findKeys
	buttons := []surface{
		u.pageRect(findContinueLeft, findContinueLeft+findButtonsWidth, transferButtonTop, transferButtonBottom),
		u.pageRect(findAbortLeft, findAbortRight, transferButtonTop, transferButtonBottom),
	}
	if game == noGame {
		u.putsJustified("No character found!  To transfer a character, put the PARTY.SAV of " +
			"Ultima IV or the SAVED.GAM of Ultima V into the game's folder.")
		keys = findKeys[1:]
		buttons = buttons[1:]
	} else {
		roman := map[int]string{ultima4: "IV", ultima5: "V"}[game]
		u.putsJustified(from.name + ", the " + from.class + " of Ultima " + roman + ", was found.")
		vellum.draw(page, buttons[0].left, buttons[0].top, continueButtonShape)
	}
	vellum.draw(page, findAbortLeft, transferButtonTop, abortButtonShape)
	buttons = append(buttons, buttons[len(buttons)-1])
	if !*fadedIn {
		u.fadeIn(u.vellumBack, u.vellum)
		*fadedIn = true
	} else {
		u.copyRect(page, onScreen)
	}
	u.mouseShow()
	choice := keys[u.menuSelect(buttons, keys)]
	u.mouseHide()
	if choice != 'C' {
		return importedCharacter{}, noGame
	}
	return from, game
}

// showStats prints the old and the new character's values side by side.
// C: seg_0263:0360 (show_stats)
func (u *intro) showStats(from importedCharacter, vellum shapes) {
	stats := u.pageRect(headerLeft+sourceColumn, headerRight, statsTop, statsBottom)
	fill(stats, vellumColour)
	win := textWindow{surf: &stats, x: stats.left, y: stats.top + textRowOffset, fg: sourceTextColour, bg: vellumColour, justify: justifyLeft}
	u.win = &win
	n := strconv.Itoa
	u.putsJustified(from.name + "\n" + from.sex + "\n" + from.class + "\n" + n(from.strength) + "\n" + n(from.dexterity) + "\n" +
		n(from.intelligence) + "\n" + n(from.magic) + "\n" + n(from.experience) + "\n" + n(from.level))
	stats.left += columnGap
	c := &u.char
	win.x, win.y, win.fg = stats.left, stats.top+textRowOffset, targetTextColour
	u.putsJustified(cString(c.name[:]) + "\n" + cString(c.sex[:]) + "\n" + classNames[avatarClass] + "\n" + n(c.strength) + "\n" +
		n(c.dexterity) + "\n" + n(c.intelligence) + "\n" + n(c.intelligence) + "\n" + n(c.experience) + "\n" + n(c.level))
	for i, shape := range transferButtonShapes {
		vellum.draw(u.back, transferButtonLefts[i], transferButtonTop, shape)
	}
}

// showTransfer shows the comparison and lets the player rename the avatar or change its
// sex; true to continue, false to give up this character.
// C: seg_0263:0592 (show_transfer)
func (u *intro) showTransfer(from importedCharacter, game int, vellum shapes) bool {
	page, onScreen := u.vellumPage()
	fill(page, vellumColour)
	fill(u.pageRect(headerLeft, headerRight, statsTop-1, statsTop-1), vellumTextColour)
	header := u.pageRect(headerLeft, headerRight, headerTop, headerBottom)
	win := textWindow{surf: &header, x: header.left + sourceColumn, y: header.top + textRowOffset, fg: vellumTextColour, bg: vellumColour}
	u.win = &win
	roman := map[int]string{ultima4: "IV", ultima5: "V"}[game]
	u.textPrint("Ultima " + roman)
	win.x = header.left + targetColumn
	u.textPrint("Ultima VI")
	labels := u.pageRect(headerLeft, headerRight-labelsRightMargin, statsTop, statsBottom)
	win = textWindow{surf: &labels, x: labels.left, y: labels.top + textRowOffset, fg: vellumTextColour, bg: vellumColour, justify: justifyRight}
	u.putsJustified("Name\nSex\nClass\nStr\nDex\nInt\nMagic\nExp\nLevel")
	u.showStats(from, vellum)
	u.copyRect(page, onScreen)
	buttons := make([]surface, len(transferKeys))
	for i := range buttons {
		buttons[i] = u.pageRect(transferButtonLefts[i], transferButtonRights[i], transferButtonTop, transferButtonBottom)
	}
	u.mouseShow()
	for {
		switch transferKeys[u.menuSelect(buttons, transferKeys)] {
		case 'C':
			return true
		case 'N':
			stats := u.pageRect(headerLeft, headerRight, statsTop, statsBottom)
			stats.pixels, stats.width = u.screen.pixels, u.screen.width
			win := textWindow{surf: &stats, x: stats.left + targetColumn, y: stats.top + textRowOffset, bg: vellumColour}
			u.win = &win
			u.mouseHide()
			u.clearToEOL()
			u.inputString(u.char.name[:], nameLength)
		case 'S':
			u.mouseHide()
			u.toggleSex()
		default:
			return false
		}
		u.showStats(from, vellum)
		u.copyRect(page, onScreen)
		u.mouseShow()
	}
}

// transferCharacter runs the transfer; true when the avatar was saved and the game can
// start.
// C: seg_0263:13D6 (transfer_character)
func (u *intro) transferCharacter() (bool, error) {
	saveWin := u.win
	defer func() { u.win = saveWin }()
	u.playMusic(sound.SongCreate)
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
	fadedIn, done := false, false
	for !done {
		from, game := u.findCharacter(vellum, &fadedIn)
		if game == noGame {
			break
		}
		u.convert(from, game)
		for u.showTransfer(from, game, vellum) {
			if u.choosePortrait(vellum, montage, true) {
				done = true
				break
			}
		}
		u.mouseHide()
	}
	u.mouseHide()
	if done {
		if err := u.saveObjlist(); err != nil {
			return false, err
		}
		u.menuDrawn = false
	}
	fill(u.vellum, 0)
	u.setPalette(&u.menuPalette)
	if u.menuDrawn {
		u.fadeIn(saved, u.vellum)
	}
	u.playMusic(sound.SongStop)
	return done, nil
}
