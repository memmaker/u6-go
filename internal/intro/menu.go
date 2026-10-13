package intro

import "github.com/memmaker/u6-go/internal/sound"

// The main menu and the input helpers of the character creation.

// The main menu's layout.
const (
	titleX              = 19
	titleBottom         = 63
	subtitleLeft        = 59
	subtitleRight       = 264
	subtitleTop         = 47
	subtitleBottom      = 75
	subtitleDissolve    = 10
	menuLeft            = 49
	menuRight           = 299
	menuTop             = 83
	menuBottom          = 196
	menuItemLeft        = 58
	menuItemRight       = 262
	titleShape          = 0
	subtitleShape       = 1
	menuShape           = 0
	firstMusicKeySong   = sound.SongUltima
	lastMusicKeySong    = sound.SongEnd
	musicKeyTenthSong   = sound.SongBrit
	musicKeyEleventh    = '-'
	musicKeyTwelfth     = '='
	scanUp, scanLeft    = 72, 75
	scanRight, scanDown = 77, 80
)

// menuItemColours are the colours of the five menu items' letters, which the menu sets to
// show the highlighted one.
var menuItemColours = [menuItems]int{14, 33, 34, 35, 36}

// menuItemTops and menuItemBottoms are the items' rows.
var (
	menuItemTops    = [menuItems]int{87, 108, 128, 149, 169}
	menuItemBottoms = [menuItems]int{107, 127, 148, 168, 193}
)

// menuKeys are the items' keys.
var menuKeys = [menuItems]int{'I', 'C', 'T', 'A', 'J'}

// mainMenu shows the main menu (drawing it when it is not on the screen) and returns the
// choice: menuIntroduction..menuJourneyOnward, or menuQuit for ctrl-Q.
// C: seg_00E1:0595 (main_menu)
func (u *intro) mainMenu() int {
	menu := u.mustShapes("MAINMENU.SHP")
	titles := u.mustShapes("TITLES.SHP")
	if u.curMusic != sound.SongUltima {
		u.playMusic(sound.SongUltima)
	}
	if !u.menuDrawn {
		u.setPalette(&u.menuPalette)
		front, back := u.screen, u.back
		back.left, back.top, back.right, back.bottom = 0, 0, screenWidth-1, titleBottom
		front.left, front.top, front.right, front.bottom = 0, 0, screenWidth-1, titleBottom
		titles.draw(back, titleX, 0, titleShape)
		u.fadeIn(back, front)

		front, back = u.screen, u.back
		back.left, back.top, back.right, back.bottom = subtitleLeft, subtitleTop, subtitleRight, subtitleBottom
		front.left, front.top, front.right, front.bottom = subtitleLeft, subtitleTop, subtitleRight, subtitleBottom
		titles.draw(back, back.left, back.top, subtitleShape)
		u.dissolve(back, front, subtitleDissolve)

		front, back = u.screen, u.back
		back.left, back.top, back.right, back.bottom = menuLeft, menuTop, menuRight, menuBottom
		front.left, front.top, front.right, front.bottom = menuLeft, menuTop, menuRight, menuBottom
		menu.draw(back, back.left, back.top, menuShape)
		u.fadeIn(back, front)
		for i, c := range menuItemColours {
			u.menuColours[i] = u.getDAC(c)
		}
		u.menuDrawn = true
	}
	u.escaped = false
	u.mouseShow()
	var items [menuItems]surface
	for i := range items {
		items[i] = surface{left: menuItemLeft, right: menuItemRight, top: menuItemTops[i], bottom: menuItemBottoms[i]}
	}
	highlight := 0
	pointed := false
	chosen := false
	lastX, lastY := u.mouseX+1, u.mouseY
	highlightItem := func() {
		for i, c := range menuItemColours {
			u.setDAC(c, u.menuColours[abs(highlight-i)])
		}
	}
	for !chosen {
		u.mousePoll()
		u.waitRetrace()
		highlightItem()
		if u.mouseX != lastX || u.mouseY != lastY {
			pointed = false
			lastX, lastY = u.mouseX, u.mouseY
			for i, r := range items {
				if u.mouseInside(r) {
					highlight = i
					pointed = true
				}
			}
		}
		if u.kbhit() {
			c := toUpper(u.getch())
			switch {
			case c == charCtrlQ:
				highlight = menuQuit
				chosen = true
			case c == charEnter:
				chosen = true
			case c >= '1' && c <= '9':
				u.playMusic(c - '1' + firstMusicKeySong)
				if c == '2' && u.curMusic == songBootupLoaded {
					u.startLoadedSong()
					u.curMusic = sound.SongBootup
				}
			case c == '0':
				u.playMusic(musicKeyTenthSong)
			case c == musicKeyEleventh:
				u.playMusic(lastMusicKeySong - 1)
			case c == musicKeyTwelfth:
				u.playMusic(lastMusicKeySong)
			case c == 0:
				switch u.getch() {
				case scanUp, scanLeft:
					if highlight != 0 {
						highlight--
					}
				case scanRight, scanDown:
					if highlight != menuItems-1 {
						highlight++
					}
				}
			default:
				for i, k := range menuKeys {
					if c == k {
						highlight = i
						chosen = true
						highlightItem()
					}
				}
			}
		}
		if u.mouseButton() && pointed {
			chosen = true
		}
	}
	u.mouseHide()
	if highlight != menuAcknowledgments {
		u.playMusic(sound.SongStop)
	}
	return highlight
}

// menuSelect waits for a key of keys or a click into the matching rectangle of rects and
// returns its index.
// C: seg_00E1:041B (menu_select)
func (u *intro) menuSelect(rects []surface, keys string) int {
	held := u.mouseButton()
	for {
		if !u.mouseButton() {
			held = false
		}
		if u.kbhit() {
			c := toUpper(u.getch())
			for i := range len(keys) {
				if int(keys[i]) == c {
					return i
				}
			}
		}
		u.mousePoll()
		pointed := -1
		for i := range len(keys) {
			if u.mouseInside(rects[i]) {
				pointed = i
				break
			}
		}
		if u.mouseButton() && !held && pointed != -1 {
			return pointed
		}
	}
}

// cursorBlinkFrames is how many retraces the text cursor stays on or off.
const cursorBlinkFrames = 20

// getKeyBlinking waits for a key with a blinking cursor at the text cursor: Enter, Backspace,
// Escape, space, a letter, digit or punctuation; the left arrow is Backspace.
// C: seg_00E1:016C (get_key)
func (u *intro) getKeyBlinking() int {
	x, y := u.win.x, u.win.y
	fg := u.win.fg
	frames := 0
	c := 0
	for done := false; !done; {
		u.waitRetrace()
		u.mousePoll()
		if frames++; frames == cursorBlinkFrames {
			frames = 0
			u.win.x, u.win.y = x, y
			u.textChar('_')
			if u.win.fg == fg {
				u.win.fg = u.win.bg
			} else {
				u.win.fg = fg
			}
		}
		if u.kbhit() {
			switch c = u.getch(); c {
			case charEnter, charBackspace, charEscape, ' ':
				done = true
			case 0:
				switch c = u.getch(); c {
				case scanLeft:
					c, done = charBackspace, true
				case scanRight:
					c, done = charNAK, true
				default:
					c = 0
				}
			}
			if isAlnum(c) || isPunct(c) {
				done = true
			}
		}
	}
	u.win.x, u.win.y = x, y
	u.win.fg = u.win.bg
	u.textChar('_')
	u.win.fg = fg
	u.win.x, u.win.y = x, y
	return c
}

// inputString reads a line of at most max characters into the C string buf; false when the
// player pressed Escape or Enter on an empty line. With max 1 the character is not echoed
// and needs no Enter.
// C: seg_00E1:028A (input_string)
func (u *intro) inputString(buf []byte, max int) bool {
	typed := make([]byte, max+1)
	xs, ys := make([]int, max+1), make([]int, max+1)
	n := 0
	for {
		var c int
		if max == 1 && n != 0 {
			c = charEnter
		} else {
			c = u.getKeyBlinking()
		}
		if c == charEscape || c == charEnter && n == 0 {
			return false
		}
		if c == charBackspace && n != 0 {
			n--
			u.win.x, u.win.y = xs[n], ys[n]
			fg := u.win.fg
			u.win.fg = u.win.bg
			u.textChar(typed[n])
			u.win.fg = fg
			u.win.x, u.win.y = xs[n], ys[n]
		}
		if c == charEnter {
			typed[n] = 0
			strcpy(buf, cString(typed))
			return true
		}
		if (isAlnum(c) || c == ' ' || c == '-' || c == '\'' || c == '.' || c == ',') && n < max {
			typed[n] = byte(c)
			xs[n], ys[n] = u.win.x, u.win.y
			if max != 1 {
				u.textChar(byte(c))
			}
			n++
		}
	}
}
