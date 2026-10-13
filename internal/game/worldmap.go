package game

import (
	"errors"
	"os"
	"path/filepath"
)

// The map is made of chunks of 8x8 tiles (CHUNKS, 64 bytes each); MAP gives the chunk at
// each place in 12-bit numbers, two to three bytes. The surface is 128x128 chunks stored
// as 8x8 parts of 16x16 chunks; each dungeon level is 32x32 chunks.
//
// The original kept four surface parts and 16 chunks in memory (C_101C_00D7, cacheChunk);
// the port keeps both files whole.

const (
	chunkTiles      = 8
	chunkShift      = 3
	chunkBytes      = chunkTiles * chunkTiles
	partChunks      = 16
	partBytes       = partChunks * partChunks * 3 / 2 // 0x180
	partsPerSide    = 8
	surfaceMapBytes = 60 * partBytes // 0x5a00, the last four parts are not stored
	levelChunks     = 32
	levelBytes      = levelChunks * levelChunks * 3 / 2 // 0x600
	chunkNumberBits = 12
	chunkNumberMask = 1<<chunkNumberBits - 1
	// surfaceMask and dungeonMask are D_05CA: coordinates wrap at the map's size
	surfaceMask = coordMask
	dungeonMask = dungeonCoordMask
	// areaAlign rounds the area's corner to a chunk
	areaAlign = coordMask &^ (chunkTiles - 1)
	// areaMargin is how far the area starts before the party (the original's 16, grown
	// with areaSize)
	areaMargin = 28
)

// worldMap is the state of the map module (seg_101C).
type worldMap struct {
	mapData []byte // MAP
	chunks  []byte // CHUNKS
	// AreaTiles are the tiles of the area around the party, AreaX, AreaY its top
	// left corner, ViewX, ViewY the view's place in it.
	AreaTiles    [areaSize * areaSize]byte
	AreaX, AreaY int
	ViewX, ViewY int
	// coordMask is D_05CA: 1023 on the surface, 255 in dungeons.
	coordMask int
	// the area last filled (D_05DA, D_05DC, D_05DE)
	shownX, shownY, shownZ int
	// gameLoaded is D_06E6: off while the area is rebuilt.
	gameLoaded bool
	// viewChanged asks for the view to be computed again (D_17B0).
	viewChanged bool
}

func (w *worldMap) initWorldMap() {
	w.ViewX, w.ViewY = 12, 12
	w.coordMask = surfaceMask
	w.forgetArea()
}

// forgetArea makes the next refreshArea fill the area again.
func (w *worldMap) forgetArea() { w.shownX, w.shownY, w.shownZ = -1, -1, -1 }

// loadMap reads MAP and CHUNKS.
func (w *worldMap) loadMap(dir string) error {
	var err error
	if w.mapData, err = os.ReadFile(filepath.Join(dir, mapFile)); err != nil {
		return err
	}
	if w.chunks, err = os.ReadFile(filepath.Join(dir, "CHUNKS")); err != nil {
		return err
	}
	if len(w.mapData) < surfaceMapBytes+dungeonLevels*levelBytes {
		return errors.New("MAP is too short")
	}
	return nil
}

// chunkAt returns the number of the chunk holding (x, y, z).
// C: seg_101C:0CF7 (getChunkIdxAt)
func (w *worldMap) chunkAt(x, y, z int) int {
	x &= surfaceMask
	y &= surfaceMask
	var at int
	if z != 0 {
		i := (x>>chunkShift)&(levelChunks-1) + (y>>chunkShift)&(levelChunks-1)*levelChunks
		at = surfaceMapBytes + (z-1)*levelBytes + i + i>>1
	} else {
		part := x>>(chunkShift+4) + (y>>(chunkShift+4))*partsPerSide
		i := (x>>chunkShift)&(partChunks-1) + (y>>chunkShift)&(partChunks-1)*partChunks
		at = part*partBytes + i + i>>1
	}
	// fix: the four parts past the surface's 60 are not in the file; the original read
	// whatever its buffer held
	if at+1 >= len(w.mapData) {
		return 0
	}
	word := int(w.mapData[at]) | int(w.mapData[at+1])<<byteBits
	if x&chunkTiles != 0 {
		return word >> 4
	}
	return word & chunkNumberMask
}

// chunkTile returns the tile at (x, y) of chunk c.
func (w *worldMap) chunkTile(c, x, y int) byte {
	return w.chunks[c*chunkBytes+(y&(chunkTiles-1))*chunkTiles+x&(chunkTiles-1)]
}

// GetTileAtXYZ returns the map tile at (x, y, z).
// C: seg_101C:0DC9 (GetTileAtXYZ)
func (w *worldMap) GetTileAtXYZ(x, y, z int) int {
	x &= w.coordMask
	y &= w.coordMask
	areaX, areaY := (x-w.AreaX)&w.coordMask, (y-w.AreaY)&w.coordMask
	if areaX < areaSize && areaY < areaSize {
		return int(w.AreaTiles[areaY*areaSize+areaX])
	}
	return int(w.chunkTile(w.chunkAt(x, y, z), x, y))
}

// refreshArea moves the area with the party: when it moved, it puts the characters in it
// back on the map, forgets the far objects and fills AreaTiles.
// C: seg_101C:0306 (C_101C_0306)
func (g *Game) refreshArea() {
	mx, my := int(g.MapX), int(g.MapY)
	g.AreaX = (mx - areaMargin) & areaAlign
	g.ViewX = (mx - g.AreaX - g.viewCenterX) & (levelChunks - 1)
	g.AreaY = (my - areaMargin) & areaAlign
	g.ViewY = (my - g.AreaY - g.viewCenterY) & (levelChunks - 1)
	if g.MapZ != 0 {
		g.coordMask = dungeonMask
	} else {
		g.coordMask = surfaceMask
	}
	z := int(g.MapZ)
	if g.AreaX != g.shownX || g.AreaY != g.shownY || z != g.shownZ {
		g.rebuildAreaObjects()
		g.gameLoaded = false
		for i := range npcCount {
			o := g.object(i)
			areaX, areaY := (o.Pos.X-g.AreaX)&g.coordMask, (o.Pos.Y-g.AreaY)&g.coordMask
			if o.Shape != 0 && o.CoordUse() == OnMap && o.Pos.Z == z && areaX < areaSize && areaY < areaSize &&
				g.areaObjects[areaY*areaSize+areaX] != i {
				g.MoveObj(i, o.Pos.X, o.Pos.Y, z)
			}
		}
		g.gameLoaded = true
		g.forgetFarObjects(g.AreaX+areaSize/2, g.AreaY+areaSize/2, z)
		g.shownX, g.shownY, g.shownZ = g.AreaX, g.AreaY, z
		for i := 0; i < areaSize; i += chunkTiles {
			for j := 0; j < areaSize; j += chunkTiles {
				c := g.chunkAt((g.shownX+i)&surfaceMask, (g.shownY+j)&surfaceMask, z)
				for y := range chunkTiles {
					for x := range chunkTiles {
						g.AreaTiles[(j+y)*areaSize+i+x] = g.chunkTile(c, x, y)
					}
				}
			}
		}
		g.hatchEggs()
	}
	g.updateMoongates()
	g.viewChanged = true
}

// enterArea loads what the party's place needs.
// C: seg_101C:054C (C_101C_054C)
func (g *Game) enterArea() {
	g.LoadNewRegions(int(g.MapX), int(g.MapY), int(g.MapZ))
	g.refreshArea()
}
