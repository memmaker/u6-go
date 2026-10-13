package vga

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"

	"github.com/memmaker/u6-go/internal/data"
)

const (
	tileSize  = 16
	tileBytes = tileSize * tileSize
	tileCount = 2048
	// tileNumberMask keeps the tile number of a tile value; the bits above select a
	// silhouette colour (DrawTileOffscreen) or are game flags.
	tileNumberMask = tileCount - 1
	// residentTiles are the map tiles (MAPTILES.VGA); the object tiles (OBJTILES.VGA) follow.
	residentTiles = 512
	paragraph     = 16

	// mask type bits (MASKTYPE.VGA)
	maskKindBits   = 3
	maskSizedBlock = 8 // the tile's data starts with its length in paragraphs

	// silhouette selection in a tile value given to DrawTileOffscreen
	silhouetteShift = 11
	silhouetteBits  = 3
)

// TileKind says how a tile's pixels are drawn.
type TileKind byte

const (
	// Opaque tiles draw all 256 pixels.
	Opaque TileKind = iota
	// Transparent tiles leave colour 255 undrawn.
	Transparent
	// PixelBlock tiles are stored as runs of pixels; the gaps are left undrawn. Here they are
	// unpacked into 256 pixels with colour 255 in the gaps.
	PixelBlock
)

// Tile is a 16x16 tile picture.
type Tile struct {
	Kind   TileKind
	Pixels [tileBytes]byte
}

// silhouetteColors are the colours DrawTileOffscreen paints a tile's black pixels in when the
// tile value's silhouette bits are 1, 2 or 3 (MagicalHalo and the like).
var silhouetteColors = [silhouetteBits + 1]byte{0, 11, 12, 9}

// LoadTiles reads the tile pictures: MAPTILES.VGA (packed, tiles 0..511 at the paragraph offsets
// in TILEINDX.VGA), OBJTILES.VGA (tiles 512..2047, in groups), MASKTYPE.VGA (packed: each tile's
// kind, then each object tile's group and each group's paragraph offset in OBJTILES.VGA) and
// ANIMMASK.VGA (packed, the masks of AnimateCoasts).
//
// GR_2A (driver offset 42). The original gets a memory block from the game, keeps the map
// tiles in it and pages object tile groups into the rest on demand, asking for the disk when
// needed; it fails (and the game quits) when the block is smaller than 196 KB. Here all tiles
// are loaded at once.
// C: U6MCGA.DRV:25A2
func (d *Driver) LoadTiles() error {
	mapTiles, err := readPacked(d.files, "MAPTILES.VGA")
	if err != nil {
		return err
	}
	index, err := fs.ReadFile(d.files, "TILEINDX.VGA")
	if err != nil {
		return err
	}
	masks, err := readPacked(d.files, "MASKTYPE.VGA")
	if err != nil {
		return err
	}
	objTiles, err := fs.ReadFile(d.files, "OBJTILES.VGA")
	if err != nil {
		return err
	}
	if d.animMasks, err = readPacked(d.files, "ANIMMASK.VGA"); err != nil {
		return err
	}
	const (
		objectTiles = tileCount - residentTiles
		groupStart  = tileCount // in MASKTYPE.VGA
		offsetStart = groupStart + objectTiles
	)
	if len(index) < tileCount*2 || len(masks) < offsetStart {
		return errors.New("vga: short tile index or mask file")
	}

	// tiles at the same offset share their picture, as in the original
	byOffset := map[int]*Tile{}
	for t := range residentTiles {
		offset := int(binary.LittleEndian.Uint16(index[t*2:])) * paragraph
		if d.tiles[t] = byOffset[offset]; d.tiles[t] != nil {
			continue
		}
		if d.tiles[t], _, err = decodeTile(mapTiles, offset, masks[t]); err != nil {
			return fmt.Errorf("vga: map tile %d: %w", t, err)
		}
		byOffset[offset] = d.tiles[t]
	}
	// An object tile group is a run of tiles with the same group number, stored one after
	// the other from the group's offset.
	offset := 0
	for t := residentTiles; t < tileCount; t++ {
		group := int(masks[groupStart+t-residentTiles])
		if t == residentTiles || group != int(masks[groupStart+t-residentTiles-1]) {
			at := offsetStart + group*2
			if at+2 > len(masks) {
				return fmt.Errorf("vga: object tile %d: bad group %d", t, group)
			}
			offset = int(binary.LittleEndian.Uint16(masks[at:])) * paragraph
		}
		var size int
		if d.tiles[t], size, err = decodeTile(objTiles, offset, masks[t]); err != nil {
			return fmt.Errorf("vga: object tile %d: %w", t, err)
		}
		offset += size
	}
	return nil
}

func readPacked(files fs.FS, name string) ([]byte, error) {
	packed, err := fs.ReadFile(files, name)
	if err != nil {
		return nil, err
	}
	unpacked, err := data.Decompress(packed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return unpacked, nil
}

var errBadTile = errors.New("tile data out of range")

// decodeTile reads the tile at offset of file and returns it and its stored size.
//
// A pixel block is a length byte (in paragraphs) and runs of pixels, each a 16-bit distance
// from the end of the previous run (laid out 176 pixels per row, the back buffer's width), a
// count byte and the pixels, ending with a count of 0.
// C: U6MCGA.DRV:23C6
func decodeTile(file []byte, offset int, mask byte) (*Tile, int, error) {
	if offset >= len(file) {
		return nil, 0, errBadTile
	}
	size := tileBytes
	if mask&maskSizedBlock != 0 {
		size = int(file[offset]) * paragraph
	}
	if offset+size > len(file) {
		return nil, 0, errBadTile
	}
	stored := file[offset : offset+size]
	tile := &Tile{}
	if mask&maskKindBits == 0 {
		copy(tile.Pixels[:], stored)
		return tile, size, nil
	}
	if mask&maskKindBits == 1 {
		tile.Kind = Transparent
		copy(tile.Pixels[:], stored)
		return tile, size, nil
	}
	tile.Kind = PixelBlock
	for i := range tile.Pixels {
		tile.Pixels[i] = transparent
	}
	at, position := 1, 0 // position in the 176-wide layout
	for {
		if at+3 > len(stored) {
			return nil, 0, errBadTile
		}
		position += int(int16(binary.LittleEndian.Uint16(stored[at:])))
		count := int(stored[at+2])
		at += 3
		if count == 0 {
			return tile, size, nil
		}
		x, y := position%BackBufferSize, position/BackBufferSize
		if x+count > tileSize || y >= tileSize || at+count > len(stored) {
			return nil, 0, errBadTile
		}
		copy(tile.Pixels[y*tileSize+x:], stored[at:at+count])
		at += count
		position += count
	}
}

// Tile returns the picture tile currently shows (after SetTileFrame).
func (d *Driver) Tile(tile int) *Tile { return d.tiles[tile&tileNumberMask] }

// drawTile draws a tile on a picture of the given width, leaving its transparent pixels.
func drawTile(pixels []byte, width int, t *Tile, x, y int) {
	for row := range tileSize {
		for col := range tileSize {
			c := t.Pixels[row*tileSize+col]
			if t.Kind != Opaque && c == transparent {
				continue
			}
			if i := (y+row)*width + x + col; x+col >= 0 && x+col < width && i >= 0 && i < len(pixels) {
				pixels[i] = c
			}
		}
	}
}

// DrawTile draws a tile on the screen with its top left corner at (x, y).
//
// GR_2D (driver offset 45).
// C: U6MCGA.DRV:2458
func (d *Driver) DrawTile(tile, x, y int) {
	drawTile(d.Screen[:], ScreenWidth, d.Tile(tile), x, y)
}

// DrawTileOffscreen draws a tile on the back buffer with its top left corner at (x, y). When
// bits 11-12 of the tile value are set, only the tile's black (colour 0) pixels are drawn, in
// colour 11, 12 or 9.
//
// GR_42 (driver offset 66).
// C: U6MCGA.DRV:24D9
func (d *Driver) DrawTileOffscreen(tile, x, y int) {
	t := d.Tile(tile)
	if s := tile >> silhouetteShift & silhouetteBits; s != 0 {
		silhouette := &Tile{Kind: Transparent}
		for i, c := range t.Pixels {
			silhouette.Pixels[i] = transparent
			if c == 0 {
				silhouette.Pixels[i] = silhouetteColors[s]
			}
		}
		t = silhouette
	}
	drawTile(d.BackBuffer[:], BackBufferSize, t, x, y)
}

// SetTileFrame makes tile show the picture of tile frame, for the animated tiles (water,
// fire, ...). Frames from the object tiles are ignored.
//
// GR_4B (driver offset 75). The original points tile's data at frame's and copies frame's
// kind; sharing the *Tile does the same.
// C: U6MCGA.DRV:2CC2
func (d *Driver) SetTileFrame(tile, frame int) {
	if frame >= residentTiles || frame < 0 {
		return
	}
	d.tiles[tile&tileNumberMask] = d.tiles[frame]
}

// AnimateCoasts redraws the 32 coast tiles (16..47) from the water tiles' current frames: each
// coast tile takes the water pixels its mask in ANIMMASK.VGA selects and keeps its land.
//
// GR_3F (driver offset 63).
// C: U6MCGA.DRV:2D2F
func (d *Driver) AnimateCoasts() {
	const (
		firstCoastTile = 16
		maskBytes      = 64
	)
	for k, water := range coastWater {
		from := d.tiles[water].Pixels[:]
		to := d.tiles[firstCoastTile+k].Pixels[:]
		mask := d.animMasks[k*maskBytes:]
		// the mask alternates counts of pixels to copy and to keep, ending with a 0 (the
		// first count may be 0)
		at := int(mask[0])
		copy(to[:at], from)
		for i := 1; ; i++ {
			n := int(mask[i])
			if n == 0 {
				break
			}
			if i%2 == 0 {
				copy(to[at:at+n], from[at:])
			}
			at += n
		}
	}
}

// coastWater is the water tile under each coast tile.
var coastWater = [...]int{11, 11, 13, 13, 15, 15, 9, 9, 13, 15, 11, 9, 11, 13, 15, 9,
	13, 15, 15, 9, 9, 11, 11, 13, 9, 11, 15, 13, 13, 15, 9, 11}
