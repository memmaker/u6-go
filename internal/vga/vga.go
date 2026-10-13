// Package vga reimplements U6MCGA.DRV, Ultima VI's driver for the 320x200 256-colour VGA/MCGA
// mode. GAME.EXE calls the driver through a jump table of far entries GR_00..GR_4E (gr.h); each
// entry is a method of Driver here. DRIVER.md maps the entries to the methods.
//
// The driver draws on two pictures: the screen and the back buffer, in which the game composes
// the map view before copying it to the screen. The port's screen is taller than the 320x200
// shown: the game keeps its panels there (see the game's layout.go) and composes what is shown.
package vga

import (
	"io/fs"

	"github.com/memmaker/u6-go/internal/host"
)

const (
	ScreenWidth  = host.ScreenWidth
	ScreenHeight = host.ScreenHeight
	// CanvasHeight is the height of the screen picture: the shown 320x200 and room below it.
	CanvasHeight = 2 * ScreenHeight
	// BackBufferWidth and BackBufferHeight are the size of the back buffer, where the game
	// draws the port's map view: 21x13 tiles of 16 pixels on a 320x200 screen (the
	// original's was 11x11 tiles), up to a tile more than the widest and tallest screen.
	BackBufferWidth  = (host.MaxScreenWidth/tileSize + 1) * tileSize
	BackBufferHeight = (host.MaxScreenHeight/tileSize + 1) * tileSize
	// packedTileStride is the row length the packed tile positions in MAPTILES.VGA count in,
	// the original's 176 pixel back buffer.
	packedTileStride = 176

	paletteBytes = 768
	// transparent is the colour that tile, sprite and pointer pictures leave undrawn.
	transparent = 255
)

// Driver is the state of U6MCGA.DRV. The exported fields are the parts of the driver's
// shared structure (tGr in gr.h) that GAME.EXE writes directly.
type Driver struct {
	Screen     [ScreenWidth * CanvasHeight]byte
	BackBuffer [BackBufferWidth * BackBufferHeight]byte
	// Palette is what the VGA DAC holds: 256 colours of three 6-bit components. The game
	// loads it from U6PAL and animates it itself; the driver only shows it.
	Palette [paletteBytes]byte

	// Font is the 8x8 character set used by DrawChar (tGr._06, loaded by the game from U6.CH).
	Font *Font
	// TextForeground and TextBackground are DrawChar's colours (tGr._0a and _0b).
	TextForeground, TextBackground byte
	// XorMode makes PutPixel and FillRect combine with exclusive or (tGr._0c).
	XorMode bool
	// Offscreen sends PutPixel and FillRect to the back buffer instead of the screen (tGr._0f).
	Offscreen bool

	// Present, when set, is called during the long screen effects (FadeIn, Vortex) so the
	// host can show them in progress. The original ran them as fast as the CPU allowed.
	Present func()

	files fs.FS
	color byte // set by SetColor

	tiles     [tileCount]*Tile
	animMasks []byte
	sprites   [spriteSlots]sprite

	pointer pointerState
}

// Frame returns the screen for the host to present.
func (d *Driver) Frame() *host.Frame {
	frame := &host.Frame{Pixels: d.Screen[:ScreenWidth*ScreenHeight], Palette: &d.Palette}
	d.pointer.addTo(frame)
	return frame
}

func (d *Driver) present() {
	if d.Present != nil {
		d.Present()
	}
}

// New sets up the driver, which reads its files (tiles, masks, pointer) from files.
//
// GR_00 (driver offset 0). The original saves the BIOS video mode, switches to mode 19
// (320x200x256) and returns its shared structure, into which the game stores the file
// callbacks it should use; files replaces those callbacks. The mode argument is ignored.
// C: U6MCGA.DRV:2198
func New(files fs.FS) *Driver {
	return &Driver{
		files: files,
		// the values tGr holds in the driver image
		TextForeground: initialTextForeground,
		TextBackground: initialTextBackground,
	}
}

const (
	initialTextForeground = 72
	initialTextBackground = 49
)

// Shutdown restores the video mode found by New.
//
// GR_03 (driver offset 3). Nothing to do on the host.
// C: U6MCGA.DRV:21B1
func (d *Driver) Shutdown() {}

// Unused1B is an empty entry that takes two arguments.
//
// GR_1B (driver offset 27). GAME.EXE never calls it.
// C: U6MCGA.DRV:35E0
func (d *Driver) Unused1B(a, b int) {}

// Unused21 is an empty entry.
//
// GR_21 (driver offset 33). GAME.EXE never calls it.
// C: U6MCGA.DRV:35E3
func (d *Driver) Unused21() {}

// InitPalette is where the drivers for other adapters set up their colours; the game calls it
// instead of loading U6PAL when it is not on VGA.
//
// GR_24 (driver offset 36). Empty in the VGA driver.
// C: U6MCGA.DRV:2D65
func (d *Driver) InitPalette() {}

// AnimatePalette is the other adapters' version of the game's fire and water palette cycling.
//
// GR_27 (driver offset 39). Empty in the VGA driver (which also leaves the argument on the
// stack; the game only calls it on other adapters).
// C: U6MCGA.DRV:2D66
func (d *Driver) AnimatePalette(step int) {}

// Unused3C is an empty entry.
//
// GR_3C (driver offset 60). GAME.EXE never calls it.
// C: U6MCGA.DRV:28EC
func (d *Driver) Unused3C() {}
