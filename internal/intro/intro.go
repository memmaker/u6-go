// Package intro is U.EXE: the title sequence, the introduction story, the main menu, the
// credits and the character creation at the gypsy's wagon, which writes the new avatar into
// the save game GAME.EXE starts from.
//
// Only the VGA (MCGA) path of the original is ported; see README.md for the deviations.
package intro

import (
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/memmaker/u6-go/internal/data"
	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
	"github.com/memmaker/u6-go/internal/vga"
)

// Result is what the player chose.
type Result int

const (
	// Quit: the player left with ctrl-Q.
	Quit Result = iota
	// StartGame: run GAME.EXE (after a character was created, or "Journey Onward").
	StartGame
)

// intro is U.EXE's state: its globals.
type intro struct {
	fsys    fs.FS
	saveDir string

	screen surface // D_5710, the VGA memory
	back   surface // D_5732, the page pictures are drawn on before they are shown
	dac    [colours * rgbBytes]byte

	menuPalette  [colours]rgb // D_4A40, the palette of the main menu (PALETTES.INT 0)
	scenePalette [colours]rgb // D_505F, the palette of the current scene
	menuColours  [menuItems]rgb

	font       *font
	mainWin    textWindow // D_566D, the window the program prints in
	win        *textWindow
	screenWin  surface // D_5722, the screen as the main window's surface
	vellum     surface // D_565F, the parchment's text area on the screen
	vellumBack surface // D_5702, the same area on the back page
	pointer    shapes  // INTRO.PTR
	blocks     shapes  // BLOCKS.SHP, the passages' backgrounds

	// passages
	passageText                                surface // D_6072, the passage's text is printed here
	passageDest                                surface // D_6086, where it goes on the back page
	passageBlock, passageBlockX, passageBlockY int

	// input
	pendingScan           int // the scan code getch returns after a 0 for an extended key
	mouseX, mouseY        int
	mouseLeft, mouseRight bool
	mouseShown            bool
	mousePress            bool // MousePress

	// pacing
	synced      bool
	presentTick uint64
	restoreTick uint64 // host tick of the last restoreScreen
	fade        fader

	timerInstalled bool
	timerCallback  func()

	random   borlandRand
	curMusic int // CurMusic: the song playing, -2 BOOTUP.M loaded but not started

	escaped   bool // D_58F0: Escape pressed during a sequence
	menuDrawn bool // D_575A: the main menu is on the screen
	introSeen bool // IntroSeen

	boot  bootState
	title titleState
	char  character
	gypsy gypsyState
}

// Run plays U.EXE from the title to the main menu's choice. fsys holds the game's files;
// saveDir is the game's writable directory: a new character is saved in its SAVEGAME
// directory as U.EXE saved it (see README.md).
func Run(fsys fs.FS, saveDir string) (Result, error) {
	u := &intro{fsys: fsys, saveDir: saveDir, curMusic: 0}
	if err := u.load(); err != nil {
		return Quit, err
	}
	sound.Start(fsys)
	host.ShowFrame = u.frame
	defer func() { u.removeTimer() }()
	return u.main()
}

// load reads the files U.EXE keeps loaded.
func (u *intro) load() error {
	u.screen = surface{pixels: make([]byte, screenWidth*screenHeight), width: screenWidth,
		right: screenWidth - 1, bottom: screenHeight - 1}
	u.back = u.screen
	newPicture(&u.back)
	raw, err := u.readLZW("U6.SET")
	if err != nil {
		return err
	}
	u.font = &font{raw: raw}
	if u.pointer, err = u.loadShapes("INTRO.PTR"); err != nil {
		return err
	}
	if u.blocks, err = u.loadShapes("BLOCKS.SHP"); err != nil {
		return err
	}
	u.random.seed = uint32(time.Now().Unix())
	u.char.karma = initialKarma
	u.boot.init()
	return nil
}

func (u *intro) readFile(name string) ([]byte, error) {
	b, err := fs.ReadFile(u.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		b, err = fs.ReadFile(u.fsys, strings.ToLower(name))
	}
	return b, err
}

// readLZW reads an LZW packed file.
// C: lzw_load
func (u *intro) readLZW(name string) ([]byte, error) {
	packed, err := u.readFile(name)
	if err != nil {
		return nil, err
	}
	return data.Decompress(packed)
}

// loadShapes reads a shape library (an empty one when the file is missing, as the original
// carried on with a NULL pointer).
func (u *intro) loadShapes(name string) (shapes, error) {
	lib, err := vga.LoadShapes(u.fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		lib, err = vga.LoadShapes(u.fsys, strings.ToLower(name))
	}
	return lib, err
}

// mustShapes loads a shape library, empty on error.
func (u *intro) mustShapes(name string) shapes {
	lib, _ := u.loadShapes(name)
	return lib
}

// frame is what the host shows: the screen, the DAC and the pointer.
func (u *intro) frame() *host.Frame {
	f := &host.Frame{Pixels: u.screen.pixels, Palette: &u.dac}
	if u.mouseShown && len(u.pointer) > 0 {
		shape := u.pointer[0]
		box := surface{pixels: make([]byte, shape.Width()*shape.Height()), width: shape.Width(),
			right: shape.Width() - 1, bottom: shape.Height() - 1}
		u.pointer.draw(box, shape.Left, shape.Up, 0)
		f.Pointer, f.PointerWidth, f.PointerHeight = box.pixels, shape.Width(), shape.Height()
		f.PointerHotX, f.PointerHotY = shape.Left, shape.Up
		f.PointerX, f.PointerY = u.mouseX, u.mouseY
		f.PointerShown = true
	}
	return f
}

// Main menu choices.
const (
	menuIntroduction = iota
	menuCreate
	menuTransfer
	menuAcknowledgments
	menuJourneyOnward
	menuQuit
	menuItems = menuQuit
)

const (
	mainWindowLeft   = 18
	mainWindowRight  = 300
	mainWindowTop    = 80
	mainWindowBottom = 197
	whiteColour      = 15
)

// main is U.EXE's main after the hardware set up.
// C: seg_00E1:0F23 (main)
func (u *intro) main() (Result, error) {
	u.screenWin = u.screen
	u.vellum = u.screen
	u.vellum.left, u.vellum.right = mainWindowLeft, mainWindowRight
	u.vellum.top, u.vellum.bottom = mainWindowTop, mainWindowBottom
	u.mainWin = textWindow{surf: &u.screenWin, fg: whiteColour, justify: justifyFull}
	u.win = &u.mainWin
	u.passageText = surface{right: screenWidth - 1, bottom: passageTextRows - 1}
	newPicture(&u.passageText)
	u.mouseHide()
	u.loadPalette(0)
	u.savePalette(&u.menuPalette)
	u.playIntro()
	for {
		switch u.mainMenu() {
		case menuIntroduction:
			u.playTitle()
			u.introSeen = true
		case menuCreate:
			if created, err := u.createCharacter(); err != nil {
				return Quit, err
			} else if created {
				if !u.introSeen {
					u.playTitle()
				}
				return u.runGame(), nil
			}
		case menuTransfer:
			if transferred, err := u.transferCharacter(); err != nil {
				return Quit, err
			} else if transferred {
				if !u.introSeen {
					u.playTitle()
				}
				return u.runGame(), nil
			}
		case menuAcknowledgments:
			u.showCredits()
		case menuJourneyOnward:
			return u.runGame(), nil
		default:
			u.playMusic(sound.SongStop)
			return Quit, nil
		}
	}
}

// passageTextRows is the height of the picture the passages are printed in (the largest
// SURF_alloc of D_6072).
const passageTextRows = 56

// runGame leaves for GAME.EXE.
// C: seg_0676:0699 (RunGame)
func (u *intro) runGame() Result {
	u.fadeOut(u.screen, 0)
	return StartGame
}

// Songs of play_music.
const (
	songBootupLoaded = -2
)

// playMusic starts song n (sound.Songs); BOOTUP.M is only loaded, startLoadedSong starts it.
// C: seg_00E1:178A (play_music)
func (u *intro) playMusic(n int) {
	if n == u.curMusic {
		return
	}
	if u.curMusic != 0 {
		_ = sound.Play(sound.SongStop)
		u.curMusic = 0
	}
	if n != 0 {
		if n != sound.SongBootup {
			_ = sound.Play(n)
		} else {
			n = songBootupLoaded
		}
		u.curMusic = n
	}
}

// startLoadedSong starts the song play_music loaded (the driver's entry 6).
func (u *intro) startLoadedSong() { _ = sound.Play(sound.SongBootup) }

// Keyboard.

const (
	charCtrlQ     = 17
	charEscape    = 27
	charEnter     = 13
	charBackspace = 8
	charNAK       = 21 // what get_key makes of the right arrow
)

// keyWaiting reports a key in the BIOS buffer without waiting.
func (u *intro) keyWaiting() bool { return u.pendingScan != 0 || len(host.KeyQueue) > 0 }

// kbhit reports a key in the buffer; a loop polling it runs once per host tick.
// C: kbhit
func (u *intro) kbhit() bool {
	u.idle()
	return u.keyWaiting()
}

// getch reads a key: its character, or 0 and then its scan code for keys without one.
// C: getch
func (u *intro) getch() int {
	for !u.keyWaiting() {
		u.yield()
	}
	if u.pendingScan != 0 {
		scan := u.pendingScan
		u.pendingScan = 0
		return scan
	}
	k := host.KeyQueue[0]
	host.KeyQueue = host.KeyQueue[1:]
	if k.Char == 0 {
		u.pendingScan = int(k.Scan)
	}
	return int(k.Char)
}

// flushKeys empties the keyboard buffer.
func (u *intro) flushKeys() {
	for u.keyWaiting() {
		u.getch()
	}
}

func isDigit(c int) bool { return c >= '0' && c <= '9' }
func isAlpha(c int) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isAlnum(c int) bool { return isDigit(c) || isAlpha(c) }
func isPunct(c int) bool { return c > ' ' && c < asciiDelete && !isAlnum(c) }

const asciiDelete = 127

func toUpper(c int) int {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// Mouse.

// mousePoll reads the mouse; true when it moved or a button changed.
// C: seg_132C:0008 (mouse_poll)
func (u *intro) mousePoll() bool {
	x, y, left, right := host.MouseX, host.MouseY, host.MouseLeft, host.MouseRight
	changed := x != u.mouseX || y != u.mouseY || left != u.mouseLeft || right != u.mouseRight
	u.mouseX, u.mouseY, u.mouseLeft, u.mouseRight = x, y, left, right
	return changed
}

func (u *intro) mouseButton() bool { return u.mouseLeft || u.mouseRight }

// waitButtonsUp waits until the mouse buttons are released.
func (u *intro) waitButtonsUp() {
	for u.mouseButton() {
		u.yield()
		u.mousePoll()
	}
}

// mouseShow and mouseHide show and hide the pointer (the host draws it over the screen).
// C: seg_1339:0000 (mouse_show), seg_1339:0016 (mouse_hide)
func (u *intro) mouseShow() { u.mouseShown = true }

func (u *intro) mouseHide() {
	if u.mouseShown {
		u.mouseShown = false
		u.waitRetrace()
	}
}

// mouseInside reports the pointer's hot spot inside r's rectangle.
// C: seg_1344:00A7 (mouse_inside)
func (u *intro) mouseInside(r surface) bool {
	return u.mouseX <= r.right && u.mouseX >= r.left && u.mouseY >= r.top && u.mouseY <= r.bottom
}

// pollInput returns a key pressed (Escape: 27) or 1 for a mouse click, 0 for none; a digit
// switches the boot intro's television channel.
// C: seg_06E9:0351 (INTRO_pollInput)
func (u *intro) pollInput() int {
	u.mousePoll()
	if u.mouseButton() {
		u.waitButtonsUp()
		return 1
	}
	if u.kbhit() {
		c := u.getch()
		if !isDigit(c) {
			return c
		}
		u.setChannel((c - '0') % channels)
	}
	return 0
}

// getKey is pollInput noting Escape.
// C: seg_06E9:033A (INTRO_getKey), seg_0C17:2FBB (TITLE_isKey)
func (u *intro) getKey() int {
	c := u.pollInput()
	if c == charEscape {
		u.escaped = true
	}
	return c
}

// waitKey waits for getKey.
func (u *intro) waitKey() {
	for u.getKey() == 0 {
	}
}

// waitInput waits for pollInput.
func (u *intro) waitInput() {
	for u.pollInput() == 0 {
	}
}

// borlandRand is Turbo C's rand.
type borlandRand struct{ seed uint32 }

const (
	randMultiplier = 22695477
	randMax        = 32767
)

// C: rand
func (r *borlandRand) next() int {
	r.seed = r.seed*randMultiplier + 1
	return int(r.seed>>16) & randMax
}

func (u *intro) rand() int { return u.random.next() }

// randomBetween picks a step between a and b by the machine's speed: a on a fast machine
// (the port's), b on a slow one. The name of the original notwithstanding, it is not random.
// C: seg_06E9:0AF0 (INTRO_rnd)
func randomBetween(a, b int) int {
	const slowest, fastest = 1500, 20000
	loops := min(max(calibratedLoops, slowest), fastest) - slowest
	span := abs(a - b)
	step := loops / ((fastest - slowest) / (span + 1))
	step = min(step, span)
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

// cString is the C string in b (up to its first NUL).
func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// strcpy copies s and its NUL into b, leaving the rest of b alone.
func strcpy(b []byte, s string) {
	n := copy(b, s)
	if n < len(b) {
		b[n] = 0
	}
}
