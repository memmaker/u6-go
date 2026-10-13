package vga

import "math"

// The driver keeps 16 sprite slots of 16x16 pixels for missiles and similar effects. The game
// uses slots 0-7 to save the back buffer under a sprite (SaveUnderSprite) and 8-15 for the
// sprites themselves (MakeRotatedSprite), then draws both with DrawSprite.
const spriteSlots = 16

type sprite [tileBytes]byte

const (
	// angleSteps is the number of sprite rotation steps in a full turn.
	angleSteps = 128
	// sineScale is the value of the sine table at a quarter turn.
	sineScale = 64
	// spriteOrigin is the offset of a sprite position from the sprite's top left corner, and
	// of the map view in the back buffer.
	spriteOrigin = 8
)

// sine is the driver's table of sineScale * sin, in angleSteps steps.
var sine = func() (table [angleSteps]int) {
	for i := range table {
		table[i] = int(math.Round(sineScale * math.Sin(2*math.Pi*float64(i)/angleSteps)))
	}
	return table
}()

// MakeRotatedSprite puts tile, turned by angle (128 steps per turn) about the point (pivotX, pivotY) of the tile in 1/16 pixels, into a sprite slot.
// Pixels turned in from outside the tile are transparent. The game passes 127, 127.
//
// GR_33 (driver offset 51).
// C: U6MCGA.DRV:2779
func (d *Driver) MakeRotatedSprite(slot, tile, angle, pivotX, pivotY int) {
	if slot < 0 || slot >= spriteSlots {
		return // fix: the original used negative slots as they came
	}
	const (
		// destination pixels are walked in half pixel steps -15, -13, ..., 15 about the centre
		halfPixels = tileSize - 1
		// source coordinates are in 1/128 pixel: a half pixel step times sineScale
		sourceShift = 7
		pivotScale  = 8 // pivot 1/16 pixels to source units
	)
	source := d.Tile(tile).Pixels
	sin, cos := sine[angle&(angleSteps-1)], sine[(angle+angleSteps/4)&(angleSteps-1)]
	out := &d.sprites[slot]
	i := 0
	for offsetY := -halfPixels; offsetY <= halfPixels; offsetY += 2 {
		for offsetX := -halfPixels; offsetX <= halfPixels; offsetX += 2 {
			sx := offsetX*cos - offsetY*sin + pivotX*pivotScale
			sy := offsetX*sin + offsetY*cos + pivotY*pivotScale
			out[i] = transparent
			if sx >= 0 && sx < tileSize<<sourceShift && sy >= 0 && sy < tileSize<<sourceShift {
				out[i] = source[(sy>>sourceShift)*tileSize+(sx>>sourceShift)]
			}
			i++
		}
	}
}

// DrawSprite draws a sprite slot on the back buffer centred at (x, y) (its top left corner at
// x-8, y-8), leaving transparent pixels.
//
// GR_36 (driver offset 54).
// C: U6MCGA.DRV:2846
func (d *Driver) DrawSprite(slot, x, y int) {
	if slot < 0 || slot >= spriteSlots {
		return
	}
	drawTile(d.BackBuffer[:], BackBufferSize, &Tile{Kind: Transparent, Pixels: d.sprites[slot]},
		x-spriteOrigin, y-spriteOrigin)
}

// SaveUnderSprite copies the back buffer's 16x16 square centred at (x, y) into a sprite slot,
// so DrawSprite can later put it back. Squares starting outside the back buffer are ignored.
//
// GR_39 (driver offset 57).
// C: U6MCGA.DRV:2897
func (d *Driver) SaveUnderSprite(slot, x, y int) {
	if slot < 0 || slot >= spriteSlots {
		return
	}
	const lastStart = (BackBufferSize-tileSize)*BackBufferSize + BackBufferSize - tileSize
	start := (y-spriteOrigin)*BackBufferSize + x - spriteOrigin
	if start < 0 || start >= lastStart {
		return
	}
	for row := range tileSize {
		copy(d.sprites[slot][row*tileSize:(row+1)*tileSize], d.BackBuffer[start+row*BackBufferSize:])
	}
}
