package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
)

// Only the objects near the party are in memory: the 2x2 surface regions (128x128 tiles
// each) around it, or the dungeon level it is on. When the party moves on, regions it left
// are written to SAVEGAME/OBJBLKxx.TMP and the new ones read from there (or from the saved
// OBJBLKxx when there is no .TMP yet). Saving the game turns the .TMP files into the saved
// ones.

const (
	regionSize  = 128
	regionShift = 7
	// regionEdge is how close (in tiles) the party gets to the edge of the loaded regions
	// before they move
	regionEdge = 5
	// tooManyObjects is printed when a region has more objects than there are slots.
	tooManyObjects = "Error-too many objects in region!\n"
)

// serializeObject adds object n to the records of a block, making a contained object's
// owner the index of its container in the block, and frees n's shape.
// C: seg_1184:25A2 (__ObjSerialize)
func (g *Game) serializeObject(n int, records []objectRecord, indexes []int) []objectRecord {
	o := g.object(n)
	if o.CoordUse() != OnMap {
		if o.Pos.Owner() < npcCount {
			if o.CoordUse() == Contained {
				o.SetCoordUse(InInventory)
			}
		} else {
			o.SetCoordUse(Contained)
		}
	}
	pos := o.Pos
	if o.CoordUse() == Contained {
		index := len(records)
		for indexes[index] != o.Pos.Owner() {
			index--
		}
		// The original builds this coordinate on the stack, leaving its third byte
		// undefined; the object's own is kept here.
		pos.SetOwner(index)
	}
	records = append(records, objectRecord{Status: uint8(o.Status), Pos: encodeCoord(pos), Shape: o.Shape, Amount: o.Amount})
	o.Shape = 0
	if len(records) >= objectCount {
		g.printf(tooManyObjects)
	}
	return records
}

// deleteTmpFiles deletes the .TMP files of the object blocks.
// C: seg_1184:26E2 (C_1184_26E2)
func (g *Game) deleteTmpFiles() {
	for b := range objectBlocks {
		if g.hasTmpFile[b] {
			os.Remove(g.savePath(objblkName(b) + tmpSuffix))
		}
	}
}

// writeBlock writes the objects of block b to its .TMP file. While the game is being saved
// (diskMode 'S') it turns all .TMP files into the saved ones and writes b's directly.
// C: seg_1184:2722 (C_1184_2722)
func (g *Game) writeBlock(records []objectRecord, block int) {
	g.hasTmpFile[block] = true
	tmp := g.savePath(objblkName(block) + tmpSuffix)
	if g.diskMode != 'S' {
		writeObjblk(tmp, records)
		return
	}
	// while saving, the .TMP file only gets the count
	os.WriteFile(tmp, binary.LittleEndian.AppendUint16(nil, uint16(len(records))), 0o644)
	for b := range objectBlocks {
		if g.hasTmpFile[b] {
			saved := g.savePath(objblkName(b))
			os.Remove(saved)
			if b != block {
				os.Rename(g.savePath(objblkName(b)+tmpSuffix), saved)
			}
			g.hasTmpFile[b] = false
		}
	}
	writeObjblk(g.savePath(objblkName(block)), records)
	g.diskMode = 'G'
	g.worldShown = true
}

// unloadObjects writes the objects on the map for which inBlock is true (and what they
// hold) to block b and frees their slots. The characters stay; what they hold goes.
// C: seg_1184:2888 (C_1184_2888) and seg_1184:2FB3 (C_1184_2FB3)
func (g *Game) unloadObjects(block int, inBlock func(Coord) bool) {
	var records []objectRecord
	indexes := make([]int, objectCount)
	before := mapList
	si := g.link[mapList]
	for len(records) < objectCount && si >= 0 {
		o := g.object(si)
		if o.CoordUse() != OnMap || !inBlock(o.Pos) {
			before, si = si, g.link[si]
			continue
		}
		if si < npcCount {
			before = si
		} else {
			indexes[len(records)] = si
			records = g.serializeObject(si, records, indexes)
			g.freeObjectCount++
		}
		last := si
		si = g.link[si]
		for si >= 0 && g.object(si).CoordUse() != OnMap {
			if len(records) < objectCount {
				indexes[len(records)] = si
				records = g.serializeObject(si, records, indexes)
			}
			g.freeObjectCount++
			last, si = si, g.link[si]
		}
		g.link[last] = g.freeObjects
		g.freeObjects = g.link[before]
		g.link[before] = si
	}
	g.writeBlock(records, block)
}

// unloadRegion writes surface region r to its block.
// C: seg_1184:2888 (C_1184_2888)
func (g *Game) unloadRegion(r int) {
	if r < 0 || g.diskMode == 'R' {
		return
	}
	x, y := r%regionsPerSide<<regionShift, r/regionsPerSide<<regionShift
	const regionMask = (regionsPerSide - 1) << regionShift
	g.unloadObjects(r, func(p Coord) bool { return p.Z == 0 && p.X&regionMask == x && p.Y&regionMask == y })
}

// unloadLevel writes dungeon level z to its block.
// C: seg_1184:2FB3 (C_1184_2FB3)
func (g *Game) unloadLevel(z int) {
	if z < 0 || z > dungeonLevels || g.diskMode == 'R' {
		return
	}
	g.unloadObjects(surfaceRegions+z-1, func(p Coord) bool { return p.Z == z })
}

// moveHeldAfterHolders moves the objects held by characters, which a block stores apart
// from them, behind their holders.
// C: seg_1184:2B27 (C_1184_2B27)
func (g *Game) moveHeldAfterHolders() {
	prev := mapList
	for g.link[prev] >= 0 {
		si := g.link[prev]
		if g.object(si).Status&InInventory != 0 {
			holder := g.object(si).Pos.Owner()
			di := g.link[holder]
			for ; di >= 0 && g.object(di).CoordUse() != OnMap && di != si; di = g.link[di] {
			}
			if di != si {
				last := g.lastHeld(si)
				holderLast := g.lastHeld(holder)
				g.link[prev] = g.link[last]
				g.link[last] = g.link[holderLast]
				g.link[holderLast] = si
				continue
			}
		}
		prev = si
	}
}

// loadObjects puts the objects of a block in free slots and merges them into the list.
// C: seg_1184:2BD8 (__ObjectsDeserialize)
func (g *Game) loadObjects(records []objectRecord) {
	slots := make([]int, len(records))
	n := 0
	for ; n < len(records); n++ {
		if g.freeObjectCount == 0 {
			slots[n] = noObject
			if n > 0 {
				g.link[slots[n-1]] = noObject
			}
			break
		}
		si := g.freeObjects
		slots[n] = si
		g.freeObjects = g.link[si]
		g.freeObjectCount--
		r := records[n]
		o := g.object(si)
		*o = Object{Status: ObjStatus(r.Status), Pos: decodeCoord(r.Pos), Shape: r.Shape, Amount: r.Amount}
		if o.CoordUse() == Contained {
			o.Pos.SetOwner(slots[o.Pos.Owner()])
		}
	}
	if n == len(records) {
		g.link[slots[n-1]] = noObject
	}
	// merge the new chain, which is in list order, into the list
	si, di := mapList, slots[0]
	restart := false
	for si >= 0 && di >= 0 {
		if g.compareListOrder(g.link[si], di) < 0 {
			si = g.link[si]
		} else {
			next := g.link[si]
			g.link[si] = di
			si = next
			for g.link[di] >= 0 && g.compareListOrder(si, g.link[di]) > 0 {
				if g.compareListOrder(di, g.link[di]) > 0 {
					restart = true
					break
				}
				di = g.link[di]
			}
			next = g.link[di]
			g.link[di] = si
			di = next
			if restart {
				si, restart = mapList, false
				continue
			}
		}
		if g.link[si] < 0 {
			g.link[si] = di
			si = noObject
		}
	}
}

// readBlock reads block b from its .TMP file, or the saved one when it has none.
// C: seg_1184:2DEF (C_1184_2DEF) and seg_1184:30A6 (C_1184_30A6)
func (g *Game) readBlock(block int) {
	name := objblkName(block)
	if g.hasTmpFile[block] {
		name += tmpSuffix
	}
	raw, err := os.ReadFile(g.savePath(name))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			g.printf("DISK ERROR!!!! %v\n", err)
		}
		return
	}
	records, err := readObjblk(bytes.NewReader(raw))
	if err == nil && len(records) > 0 {
		g.loadObjects(records)
	}
}

// loadRegion reads surface region r.
// C: seg_1184:2DEF (C_1184_2DEF)
func (g *Game) loadRegion(r int) {
	if r < 0 || g.diskMode == 'R' {
		return
	}
	g.readBlock(r & (surfaceRegions - 1))
}

// loadLevel reads dungeon level z.
// C: seg_1184:30A6 (C_1184_30A6)
func (g *Game) loadLevel(z int) {
	if z < 0 || z > dungeonLevels || g.diskMode == 'R' {
		return
	}
	g.readBlock(surfaceRegions + z - 1)
}

// regionAt returns the surface region holding (x, y).
func regionAt(x, y int) int {
	return x>>regionShift&(regionsPerSide-1) + y>>(regionShift-3)&(surfaceRegions-regionsPerSide)
}

// LoadNewRegions makes the objects around (x, y, z) be in memory: the 2x2 surface regions
// nearest to it, or its dungeon level.
// C: seg_1184:3176 (LoadNewRegions)
func (g *Game) LoadNewRegions(x, y, z int) {
	unloaded, loaded := 0, 0
	if g.loadedLevel > 0 && g.loadedLevel != z {
		g.forgetFarObjects(x, y, z)
		g.unloadLevel(g.loadedLevel)
		unloaded++
	}
	var regions [4]int
	if z == 0 {
		if g.topLeftRegion == noObject {
			regions[0] = regionAt(x-regionSize/2, y-regionSize/2)
		} else {
			// keep the regions until the party is near their edge
			dx := 0
			rx := x - g.topLeftRegion%regionsPerSide<<regionShift
			for ; rx < regionEdge; rx, dx = rx+regionSize, dx-1 {
			}
			for ; rx >= 2*regionSize-regionEdge; rx, dx = rx-regionSize, dx+1 {
			}
			dy := 0
			ry := y - g.topLeftRegion/regionsPerSide<<regionShift
			for ; ry < regionEdge; ry, dy = ry+regionSize, dy-regionsPerSide {
			}
			for ; ry >= 2*regionSize-regionEdge; ry, dy = ry-regionSize, dy+regionsPerSide {
			}
			const rowMask = surfaceRegions - regionsPerSide
			regions[0] = (g.topLeftRegion%regionsPerSide+dx)&(regionsPerSide-1) + (g.topLeftRegion&rowMask+dy)&rowMask
		}
		const rowMask = surfaceRegions - regionsPerSide
		regions[1] = (regions[0]+1)&(regionsPerSide-1) + regions[0]&rowMask
		regions[2] = (regions[0] + regionsPerSide) & (surfaceRegions - 1)
		regions[3] = (regions[1] + regionsPerSide) & (surfaceRegions - 1)
		g.topLeftRegion = regions[0]
	} else {
		regions = [4]int{noObject, noObject, noObject, noObject}
		g.topLeftRegion = noObject
	}
	contains := func(list [4]int, r int) bool {
		for _, v := range list {
			if v == r {
				return true
			}
		}
		return false
	}
	if g.loadedLevel == 0 {
		for si, r := range g.loadedRegions {
			if contains(regions, r) || r == noObject {
				continue
			}
			if unloaded == 0 {
				g.forgetFarObjects(x, y, z)
			}
			g.unloadRegion(r)
			unloaded++
			g.loadedRegions[si] = noObject
		}
	}
	if z == 0 {
		for _, r := range regions {
			if contains(g.loadedRegions, r) {
				continue
			}
			si := 0
			for ; si < 4 && g.loadedRegions[si] != noObject; si++ {
			}
			g.loadRegion(r)
			loaded++
			// fix: the original writes past the four entries when none is free
			if si < 4 {
				g.loadedRegions[si] = r
			}
		}
	} else if g.loadedLevel != z {
		g.loadLevel(z)
		loaded++
	}
	g.loadedLevel = z
	if loaded > 0 {
		g.moveHeldAfterHolders()
	}
}

// rebuildAreaObjects points areaObjects at the first object of each tile of the area.
// C: seg_1184:2ECC (C_1184_2ECC)
func (g *Game) rebuildAreaObjects() {
	for i := range g.areaObjects {
		g.areaObjects[i] = noObject
	}
	for si := g.link[mapList]; si >= 0; si = g.link[si] {
		if o := g.object(si); o.CoordUse() == OnMap {
			if i, ok := g.areaIndex(o.Pos.X, o.Pos.Y); ok && g.areaObjects[i] == noObject {
				g.areaObjects[i] = si
			}
		}
	}
}

// commitObjects writes the objects in memory to their blocks and makes the .TMP files the
// saved ones, then loads the regions again.
// C: seg_1184:33CA (C_1184_33CA)
func (g *Game) commitObjects() {
	if g.MapZ != 0 {
		g.unloadLevel(int(g.MapZ))
	} else {
		for _, r := range g.loadedRegions {
			g.unloadRegion(r)
		}
	}
	for b := range objectBlocks {
		if g.hasTmpFile[b] {
			saved := g.savePath(objblkName(b))
			os.Remove(saved)
			os.Rename(saved+tmpSuffix, saved)
			g.hasTmpFile[b] = false
		}
	}
	for i := range g.loadedRegions {
		g.loadedRegions[i] = noObject
	}
	g.topLeftRegion, g.loadedLevel = noObject, noObject
	g.LoadNewRegions(int(g.MapX), int(g.MapY), int(g.MapZ))
	g.rebuildAreaObjects()
}
