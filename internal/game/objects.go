package game

// The objects in memory form one linked list (Link[]), started by the link of slot
// mapList: each object on the map is followed by the objects it holds, and the objects on
// the map are sorted by y, then x (then z). Free slots are on two more lists, one for
// ordinary objects and one for monsters. areaObjects points at the first object of each
// tile of the 40x40 area around the party.

const noObject = -1

// objectLists is the state of OBJECT.C (seg_1184).
type objectLists struct {
	link [objectCount]int
	// free slots (D_D5DA/D_E6E0 for objects, D_BDD6/D_E6E2 for monsters)
	freeObjects, freeObjectCount   int
	freeMonsters, freeMonsterCount int
	// areaObjects is MapObjPtr: the first object at each tile of the area, or noObject.
	areaObjects [areaSize * areaSize]int

	// FindLoc/NextLoc cursor (D_06F4, D_06F6, D_06F8) and which part of a big object
	// was found (D_0658: 1 left, 2 top, 3 top left)
	locNext    int
	locNextRow bool
	locAt      Coord
	locPart    int
	// FindInv/NextInv cursor (D_06FB, D_06FD, D_06FF, D_0701) and its filter for
	// carried objects only (D_0656)
	invNext, invOwner int
	invType           ObjectType
	invQuality        int
	invCarriedOnly    bool
	// SearchArea/NextArea cursor (D_0703, D_0705, D_BDCE, D_BDD8)
	areaNext, areaEnd                     int
	areaX0, areaY0, areaX1, areaY1, areaZ int
	// lastMovedContent is the last object DeleteObj moved out of the deleted one (D_0707).
	lastMovedContent int
	// allowSubmissionAmulet lets the amulet of submission be moved (D_065A).
	allowSubmissionAmulet bool

	// the loaded surface regions (D_06E8), the region at the top left (D_06F0) and the
	// loaded level (D_06F2)
	loadedRegions [4]int
	topLeftRegion int
	loadedLevel   int
	// hasTmpFile says which object blocks have a .TMP file with changes (D_065C).
	hasTmpFile [objectBlocks]bool
	// diskMode is D_2E9E: 'S' while saving, 'R' to skip the region files.
	diskMode byte
	// musicianInView remembers whether a bard was in view (D_0654).
	musicianInView bool
}

// posKey orders map positions as the list does: by y, then x.
func posKey(x, y int) int { return y<<coordBits | x }

// areaIndex returns the index in areaObjects of map position (x, y), or false when it is
// outside the area.
func (g *Game) areaIndex(x, y int) (int, bool) {
	areaX, areaY := (x-g.AreaX)&coordMask, (y-g.AreaY)&coordMask
	if areaX < areaSize && areaY < areaSize {
		return areaY*areaSize + areaX, true
	}
	return 0, false
}

// listPosition finds where an object at (x, y) is in the list. With insert it returns
// the object after which one would be inserted (mapList when it goes first); otherwise
// the first object at or after (x, y).
// C: seg_1184:02FA (C_1184_02FA)
func (g *Game) listPosition(x, y int, insert bool) int {
	const lastTile = areaSize*areaSize - 1
	if j, ok := g.areaIndex(x, y); ok {
		if !insert {
			i := g.areaObjects[j]
			if i < 0 {
				for ; j < lastTile && g.areaObjects[j] == noObject; j++ {
				}
				i = g.areaObjects[j]
			}
			return i
		}
		// the first object before j and the first one after
		i := j - 1
		for ; i >= 0 && g.areaObjects[i] == noObject; i-- {
		}
		for ; j < lastTile && g.areaObjects[j] == noObject; j++ {
		}
		if i >= 0 {
			i = g.areaObjects[i]
		} else {
			i = mapList
		}
		stop := g.areaObjects[j]
		for next := g.link[i]; next >= 0 && next != stop; i, next = next, g.link[next] {
			if o := g.object(next); o.CoordUse() == OnMap {
				if o.Pos.Y > y || o.Pos.Y == y && o.Pos.X >= x {
					break
				}
			}
		}
		return i
	}
	// outside the area: walk the list from the end of the area or from the start
	prev := mapList
	lastY, lastX := (g.AreaY+areaSize-1)&coordMask, (g.AreaX+areaSize-1)&coordMask
	if lastY < y || lastY == y && lastX < x {
		i := lastTile
		for ; i >= 0 && g.areaObjects[i] == noObject; i-- {
		}
		if i >= 0 {
			prev = g.areaObjects[i]
		}
	}
	// The original compares the coordinates as 20-bit numbers of y and x.
	for next := g.link[prev]; next >= 0; prev, next = next, g.link[next] {
		if o := g.object(next); o.CoordUse() == OnMap && posKey(o.Pos.X, o.Pos.Y) >= posKey(x, y) {
			break
		}
	}
	if !insert {
		return g.link[prev]
	}
	return prev
}

// rightOf and below return c moved one tile, carrying into the next field as the
// original's bit fields do.
func rightOf(c Coord) Coord {
	c.X++
	c.Y |= c.X >> coordBits
	c.X &= coordMask
	return c
}

func below(c Coord) Coord {
	c.Y++
	c.Z |= c.Y >> coordBits
	c.Y &= coordMask
	return c
}

// NextLoc returns the next object at the position given to FindLoc, including the parts
// of big objects (two tiles wide or high) whose base is to the right of or below it.
// C: seg_1184:053F (NextLoc)
func (g *Game) NextLoc() int {
	found := noObject
	at := g.locAt
	right, down := rightOf(at), below(at)
	downRight := below(right)
	for g.locNext >= 0 && found < 0 {
		n := g.locNext
		o := g.object(n)
		if o.CoordUse() == OnMap && o.Pos.Z == at.Z {
			flags := g.TileFlag[g.tileFrame(n)]
			switch o.Pos {
			case at:
				found = n
			case right:
				if flags&tileDoubleWidth != 0 {
					found, g.locPart = n, 1
				}
			case down:
				if flags&tileDoubleHeight != 0 {
					found, g.locPart = n, 1
					if flags&tileDoubleWidth != 0 {
						g.locPart = 2
					}
				}
			case downRight:
				if flags&(tileDoubleWidth|tileDoubleHeight) == tileDoubleWidth|tileDoubleHeight {
					found, g.locPart = n, 3
				}
			default:
				if !g.locNextRow {
					g.locNext = g.listPosition(at.X, at.Y+1, false)
					g.locNextRow = true
					continue
				}
				g.locNext = noObject
				return found
			}
		}
		g.locNext = g.link[n]
	}
	return found
}

// FindLoc returns the first object at (x, y, z); NextLoc returns the others.
// C: seg_1184:075A (FindLoc)
func (g *Game) FindLoc(x, y, z int) int {
	if z != 0 {
		x &= dungeonCoordMask
		y &= dungeonCoordMask
	}
	g.locPart = 0
	g.locNextRow = false
	g.locNext = g.listPosition(x, y, false)
	g.locAt = Coord{x, y, z}
	return g.NextLoc()
}

// findTypeAt returns the first object of type t on the map at (x, y) on the current level.
// C: seg_1184:07A7 (C_1184_07A7)
func (g *Game) findTypeAt(t ObjectType, x, y int) int {
	for i := g.listPosition(x, y, false); i >= 0; i = g.link[i] {
		o := g.object(i)
		if o.CoordUse() == OnMap && (o.Pos.X != x || o.Pos.Y != y) {
			break
		}
		if o.CoordUse() == OnMap && o.Pos.Z == int(g.MapZ) && o.Type() == t {
			return i
		}
	}
	return noObject
}

// FindInv returns the first object n holds; NextInv returns the others.
// C: seg_1184:0840 (FindInv)
func (g *Game) FindInv(n int) int {
	i := g.link[n]
	if i < 0 {
		return noObject
	}
	if o := g.object(i); o.CoordUse() != OnMap && o.Pos.Owner() == n {
		g.invNext = g.link[i]
		g.invOwner = n
		g.invType = o.Type()
		g.invQuality = -1
		return i
	}
	return noObject
}

// NextInv returns the next object held by the one given to FindInv.
// C: seg_1184:0899 (NextInv)
func (g *Game) NextInv() int {
	found := noObject
	for ; g.invNext >= 0 && found < 0; g.invNext = g.link[g.invNext] {
		o := g.object(g.invNext)
		if o.CoordUse() == OnMap {
			g.invNext = noObject
			break
		}
		if o.Pos.Owner() == g.invOwner && (!g.invCarriedOnly || o.CoordUse() == InInventory || o.CoordUse() == Contained) {
			found = g.invNext
		}
	}
	return found
}

// NextArea returns the next object on the map in the rectangle given to SearchArea.
// C: seg_1184:090D (NextArea)
func (g *Game) NextArea() int {
	found := noObject
	if g.areaNext < 0 {
		return found
	}
	for ; g.areaNext != g.areaEnd && g.areaNext >= 0 && found < 0; g.areaNext = g.link[g.areaNext] {
		o := g.object(g.areaNext)
		if o.CoordUse() != OnMap {
			continue
		}
		x, y := o.Pos.X&g.coordMask, o.Pos.Y&g.coordMask
		if x >= g.areaX0 && x <= g.areaX1 && y >= g.areaY0 && y <= g.areaY1 && o.Pos.Z == g.areaZ {
			found = g.areaNext
		}
	}
	if g.areaNext == g.areaEnd {
		g.areaNext, g.areaEnd = noObject, noObject
	}
	return found
}

// SearchArea returns the first object on the map between (x1, y1) and (x2, y2) on the
// current level; NextArea returns the others.
// C: seg_1184:09DE (SearchArea)
func (g *Game) SearchArea(x1, y1, x2, y2 int) int {
	g.areaX0, g.areaY0, g.areaX1, g.areaY1 = max(x1, 0), max(y1, 0), x2, y2
	g.areaZ = int(g.MapZ)
	g.areaNext = g.listPosition(g.areaX0, g.areaY0, false)
	g.areaEnd = g.listPosition(x2+1, y2, false)
	return g.NextArea()
}

// NextInvType returns the next object of the type given to FindInvType.
// C: seg_1184:0A37 (NextInvType)
func (g *Game) NextInvType() int {
	found := noObject
	for ; g.invNext >= 0 && found < 0; g.invNext = g.link[g.invNext] {
		o := g.object(g.invNext)
		if o.CoordUse() == OnMap {
			g.invNext = noObject
			break
		}
		if o.Pos.Owner() == g.invOwner && o.Type() == g.invType &&
			(g.invQuality < 0 || o.Quality() == g.invQuality) &&
			(!g.invCarriedOnly || o.CoordUse() == InInventory || o.CoordUse() == Contained) {
			found = g.invNext
		}
	}
	return found
}

// FindInvType returns the first object of type t (and quality, unless it is -1) that n
// holds.
// C: seg_1184:0AE2 (FindInvType)
func (g *Game) FindInvType(n int, t ObjectType, quality int) int {
	if n < 0 {
		return noObject
	}
	i := g.link[n]
	if i < 0 {
		return noObject
	}
	if o := g.object(i); o.CoordUse() != OnMap && o.Pos.Owner() == n {
		g.invNext, g.invOwner, g.invType, g.invQuality = i, n, t, quality
		return g.NextInvType()
	}
	return noObject
}

// livingAt returns the first living character from slot first on at (x, y) on the current
// level.
// C: seg_1184:0B3D (C_1184_0B3D)
func (g *Game) livingAt(first, x, y int) int {
	for i := max(first, 1); i < npcCount; i++ {
		o := g.object(i)
		if o.Shape != 0 && !g.NPCs[i].Has(Dead) && o.Pos.X == x && o.Pos.Y == y && o.Pos.Z == int(g.MapZ) {
			return i
		}
	}
	return noObject
}

// updateSlimeFrames makes each slime show which of its neighbours are slimes.
// C: seg_1184:0BC1 (C_1184_0BC1)
func (g *Game) updateSlimeFrames() {
	for slime := firstMonster; slime < npcCount; slime++ {
		o := g.object(slime)
		if o.Type() != ObjSlime || g.NPCs[slime].Has(Dead) {
			continue
		}
		frame := 0
		for dir, bit := 0, 1; dir < 8; dir, bit = dir+2, bit<<1 {
			nx, ny := o.Pos.X+dirStepX[dir], o.Pos.Y+dirStepY[dir]
			for neighbour := g.livingAt(firstMonster, nx, ny); neighbour >= 0; neighbour = g.livingAt(neighbour+1, nx, ny) {
				if g.object(neighbour).Type() == ObjSlime && !g.NPCs[neighbour].Has(Dead) {
					frame |= bit
					break
				}
			}
		}
		o.SetFrame(frame)
	}
}

// Terrain edge bits name the side of a tile a big creature's next part is on.
var (
	// tailFrames is the frame of a serpent's tail for the edges of its last part (D_074E)
	tailFrames = [16]int{0, 7, 5, 12, 3, 9, 11, 0, 1, 13, 8, 0, 10, 0, 0, 0}
	// edgeOf is the edge of direction d (D_075E)
	edgeOf = [7]byte{terrainNorth, 0, terrainEast, 0, terrainSouth, 0, terrainWest}
)

// clearEdge returns the mask that clears the edge of direction d (D_0747). The original
// also indexes it with values up to 15, reading on into tailFrames (D_074E), which follows
// it in memory.
func clearEdge(d int) byte {
	edges := [7]byte{^byte(terrainNorth), 0, ^byte(terrainEast), 0, ^byte(terrainSouth), 0, ^byte(terrainWest)}
	if d < len(edges) {
		return edges[d]
	}
	return byte(tailFrames[d-len(edges)])
}

// directionFromEdges returns the direction away from the first edge set in edges, as the
// segments of a serpent follow each other, or keep when none is set.
func directionFromEdges(edges byte, keep int) int {
	switch {
	case edges&terrainNorth != 0:
		return 4
	case edges&terrainEast != 0:
		return 6
	case edges&terrainSouth != 0:
		return 0
	case edges&terrainWest != 0:
		return 2
	}
	return keep
}

// moveSerpent moves (or with remove, deletes) the body of the silver serpent n after its
// head moved from (x, y).
// C: seg_1184:0CBE (C_1184_0CBE)
func (g *Game) moveSerpent(n, x, y int, remove bool) {
	prev, last := n, n
	atX, atY := x, y
	edges := g.TerrainType[g.tileFrame(n)]
	dir := directionFromEdges(edges, 0)
	frame := dir
	head := g.object(n)
	z := head.Pos.Z
	switch {
	case head.Pos.X > x:
		frame = 2
		edges |= terrainEast
	case head.Pos.X < x:
		frame = 6
		edges |= terrainWest
	case head.Pos.Y > y:
		frame = 4
		edges |= terrainSouth
	case head.Pos.Y < y:
		frame = 0
		edges |= terrainNorth
	}
	head.SetFrame(frame)
	segment := head.Quality() + 1
	segmentFrame := 0
	for {
		part := g.FindLoc(atX-dirStepX[dir], atY-dirStepY[dir], z)
		for ; part >= 0; part = g.NextLoc() {
			o := g.object(part)
			if part >= mapList && o.Type() == ObjSilverSerpent && o.Quality() == segment && o.Quantity() == n {
				prev, last = last, part
				segmentFrame = o.Frame()
				segment++
				o.Amount = uint16(segment<<byteBits | o.Quantity())
				atX, atY = o.Pos.X, o.Pos.Y
				dir = directionFromEdges(g.TerrainType[g.tileFrame(part)]&clearEdge(dir), dir)
				if remove {
					g.DeleteObj(part)
				}
				break
			}
		}
		if part < 0 || segmentFrame&15 < 8 {
			break
		}
	}
	if remove {
		return
	}
	if prev != n {
		e := g.TerrainType[g.tileFrame(prev)] & clearEdge(g.object(last).Frame()&15^5)
		switch {
		case e&terrainNorth != 0:
			g.object(prev).SetFrame(1)
		case e&terrainEast != 0:
			g.object(prev).SetFrame(3)
		case e&terrainSouth != 0:
			g.object(prev).SetFrame(5)
		case e&terrainWest != 0:
			g.object(prev).SetFrame(7)
		}
	}
	if last == n {
		return
	}
	if prev != n {
		g.object(last).SetFrame(tailFrames[edges>>4])
	} else {
		g.object(last).SetFrame(head.Frame() + 1)
	}
	g.MoveObj(last, x, y, z)
	tail := g.object(last)
	tail.Amount = uint16((head.Quality()+1)<<byteBits | tail.Quantity())
}

// dragonPartFrames are the frames of a dragon's parts by direction (D_17CE).
var dragonPartFrames = [16]int{8, 32, 16, 24, 26, 10, 34, 18, 20, 28, 12, 36, 38, 22, 30, 14}

// dragonPart returns the part of a dragon facing dir next to (x, y, z).
// C: seg_1184:10F1 (C_1184_10F1)
func (g *Game) dragonPart(x, y, z, dir, facing int) int {
	frame := dragonPartFrames[dir>>1+(facing&6)<<1]
	for part := g.FindLoc(x+dirStepX[dir], y+dirStepY[dir], z); part >= 0; part = g.NextLoc() {
		if o := g.object(part); o.Type() == ObjDragon && o.Frame()&62 == frame {
			return part
		}
	}
	return noObject
}

// moveBigCreature moves (or with remove, deletes) the other parts of the big creature n
// after it moved from (x, y).
// C: seg_1184:116D (C_1184_116D)
func (g *Game) moveBigCreature(n, x, y int, remove bool) {
	o := g.object(n)
	ox, oy, z := o.Pos.X, o.Pos.Y, o.Pos.Z
	edges := g.TerrainType[g.tileFrame(n)]
	t := o.Type()
	if t == ObjHydra {
		t = ObjHydra2
	}
	facing := 0
	if t == ObjDragon {
		facing = o.Frame() & 6 >> 1
	}
	for dir := 0; dir < 8; dir++ {
		if t != ObjHydra2 && (dir&1 != 0 || edges&edgeOf[dir] == 0) {
			continue
		}
		var other int
		if t == ObjDragon {
			other = g.dragonPart(x, y, z, dir, facing<<1)
		} else {
			for other = g.FindLoc(dirStepX[dir]+x, dirStepY[dir]+y, z); other >= 0; other = g.NextLoc() {
				part := g.object(other)
				if other < mapList || part.Type() != t {
					continue
				}
				if t == ObjHydra2 && part.Frame()>>2 == dir {
					break
				}
				if t != ObjHydra2 && g.TerrainType[g.tileFrame(other)]&edgeOf[dir^4] != 0 {
					break
				}
			}
		}
		if other < 0 {
			continue
		}
		if remove {
			g.DeleteObj(other)
		} else {
			g.MoveObj(other, dirStepX[dir]+ox, dirStepY[dir]+oy, z)
		}
	}
}

// isBigCreature says whether type t has parts on several tiles.
func isBigCreature(t ObjectType) bool {
	return t == ObjHydra || t == ObjDragon || t == ObjShip || t >= ObjGiantScorpion
}

// coordMaskFor is the coordinate range of level z.
func coordMaskFor(z int) int {
	if z != 0 {
		return dungeonCoordMask
	}
	return coordMask
}

// AddMapObj puts a new object on the map; it returns its number or noObject when there
// is no free slot.
// C: seg_1184:13A7 (AddMapObj)
// C: seg_1184:134C (AddObj), which only picks AddMapObj or AddInvObj
func (g *Game) AddMapObj(shape uint16, status ObjStatus, amount uint16, x, y, z int) int {
	x &= coordMaskFor(z)
	y &= coordMaskFor(z)
	if g.freeObjectCount < 1 {
		return noObject
	}
	n := g.freeObjects
	after := g.listPosition(x, y, true)
	g.freeObjects = g.link[g.freeObjects]
	g.freeObjectCount--
	g.Objects[n] = Object{Status: status | StatusLocal, Pos: Coord{x, y, z}, Shape: shape, Amount: amount}
	g.link[n] = g.link[after]
	g.link[after] = n
	if i, ok := g.areaIndex(x, y); ok {
		g.areaObjects[i] = n
	}
	return n
}

// AddInvObj gives owner a new object.
// C: seg_1184:14A8 (AddInvObj)
func (g *Game) AddInvObj(shape uint16, status ObjStatus, amount uint16, owner int) int {
	if g.StatusDisplay == cmdStatusEquipment && int(g.Party[g.statusMember]) == owner {
		g.StatusDirty = true
	}
	if g.freeObjectCount < 1 {
		return noObject
	}
	n := g.freeObjects
	g.freeObjects = g.link[g.freeObjects]
	g.freeObjectCount--
	o := g.object(n)
	*o = Object{Status: status, Shape: shape, Amount: amount}
	if ObjectType(shape) != ObjGargoyleLens {
		o.Status |= StatusLocal
	}
	o.Pos.SetOwner(owner)
	g.link[n] = g.link[owner]
	g.link[owner] = n
	return n
}

// AddMonster puts a new monster of type t on the map.
// C: seg_1184:1540 (AddMonster)
func (g *Game) AddMonster(t ObjectType, x, y, z int) int {
	x &= coordMaskFor(z)
	y &= coordMaskFor(z)
	if g.freeMonsterCount == 0 {
		return noObject
	}
	n := g.freeMonsters
	after := g.listPosition(x, y, true)
	g.freeMonsters = g.link[g.freeMonsters]
	g.freeMonsterCount--
	shape := shapeOf(t, 0)
	g.Objects[n] = Object{Status: StatusLocal | StatusOwned, Pos: Coord{x, y, z}, Shape: shape, Amount: 1}
	g.NPCs[n].Status = 0
	g.NPCs[n].Mode = AIAssault
	g.NPCs[n].OrigShape = shape
	g.link[n] = g.link[after]
	g.link[after] = n
	if i, ok := g.areaIndex(x, y); ok {
		g.areaObjects[i] = n
	}
	if t == ObjSlime {
		g.updateSlimeFrames()
	}
	return n
}

// previousInList returns the object whose link is n.
// C: seg_1184:1665 (C_1184_1665)
func (g *Game) previousInList(n int) int {
	o := g.object(n)
	var i int
	if o.CoordUse() == OnMap {
		i = g.listPosition(o.Pos.X, o.Pos.Y, true)
	} else {
		i = o.Pos.Owner()
	}
	for ; i >= 0 && g.link[i] != n; i = g.link[i] {
	}
	return i
}

// unlinkFromArea makes the area's pointer to removed (which was at (x, y)) point at next
// when next is at the same place, or nowhere.
// C: seg_1184:16D5 (C_1184_16D5)
func (g *Game) unlinkFromArea(x, y, z, removed, next int) {
	if g.object(removed).CoordUse() != OnMap {
		return
	}
	mask := coordMaskFor(z)
	areaX, areaY := (x-g.AreaX)&mask, (y-g.AreaY)&mask
	if areaX >= areaSize || areaY >= areaSize || g.areaObjects[areaY*areaSize+areaX] != removed {
		return
	}
	i := areaY*areaSize + areaX
	if next >= 0 && g.object(next).CoordUse() == OnMap && (g.object(next).Pos.X != x || g.object(next).Pos.Y != y) {
		g.areaObjects[i] = noObject
	} else {
		g.areaObjects[i] = next
	}
}

// DeleteObj removes object n, leaving what it holds where it was.
// C: seg_1184:17A5 (DeleteObj)
func (g *Game) DeleteObj(n int) {
	if n < 0 {
		return
	}
	g.lastMovedContent = noObject
	if g.StatusDisplay == cmdStatusEquipment && g.isHeldBy(n, int(g.Party[g.statusMember])) {
		g.StatusDirty = true
	}
	o := g.object(n)
	t := o.Type()
	if t == ObjShip {
		return
	}
	if n < npcCount {
		if isBigCreature(t) {
			g.moveBigCreature(n, o.Pos.X, o.Pos.Y, true)
		} else if t == ObjSilverSerpent {
			g.moveSerpent(n, o.Pos.X, o.Pos.Y, true)
		}
	}
	for {
		content := g.FindInv(n)
		if content < 0 {
			break
		}
		g.lastMovedContent = content
		if o.CoordUse() == OnMap {
			g.MoveObj(content, o.Pos.X, o.Pos.Y, o.Pos.Z)
		} else {
			g.InsertObj(content, o.Pos.Owner(), o.CoordUse())
		}
	}
	o.Shape = 0
	prev := g.previousInList(n)
	if prev >= 0 {
		g.link[prev] = g.link[n]
		if n < npcCount {
			g.link[n] = g.freeMonsters
			g.freeMonsters = n
			g.freeMonsterCount++
		} else {
			g.link[n] = g.freeObjects
			g.freeObjects = n
			g.freeObjectCount++
		}
	}
	// fix: the original reads Link[-1] when n was not in the list.
	next := noObject
	if prev >= 0 {
		next = g.link[prev]
	}
	g.unlinkFromArea(o.Pos.X, o.Pos.Y, o.Pos.Z, n, next)
	if t == ObjSlime {
		g.updateSlimeFrames()
	}
}

// regionReach is how far from the party objects stay in memory (20 tiles).
const regionReach = 20

// forgetFarObjects deletes the temporary monsters and objects more than 20 tiles away from
// (x, y, z) and resets the hatched eggs there.
// C: seg_1184:19AA (C_1184_19AA)
func (g *Game) forgetFarObjects(x, y, z int) {
	x &= g.coordMask
	y &= g.coordMask
	far := func(p Coord) bool {
		return x-regionReach > p.X || x+regionReach <= p.X || y-regionReach > p.Y || y+regionReach <= p.Y || p.Z != z
	}
	for obj := firstMonster; obj < npcCount; obj++ {
		if o := g.object(obj); o.Shape != 0 && far(o.Pos) && o.Has(StatusLocal) {
			g.DeleteObj(obj)
		}
	}
	for obj := g.link[mapList]; obj >= 0; {
		next := g.link[obj]
		if o := g.object(obj); o.CoordUse() == OnMap && far(o.Pos) {
			if o.Has(StatusLocal) || o.Shape == 0 {
				g.DeleteObj(obj)
				if g.lastMovedContent >= 0 {
					next = g.lastMovedContent
				}
			} else if o.Type() == ObjEgg && o.Has(StatusCursed) {
				o.Status &^= StatusCursed
			}
		}
		obj = next
	}
}

// isHeldBy says whether n is held by holder, directly or inside something.
// C: seg_1184:1D2E (C_1184_1D2E)
func (g *Game) isHeldBy(n, holder int) bool {
	for g.object(n).CoordUse() != OnMap {
		n = g.object(n).Pos.Owner()
		if n == holder {
			return true
		}
	}
	return false
}

// lastHeld returns the last object in the list that n holds (n itself when none).
// C: seg_1184:1D60 (C_1184_1D60)
func (g *Game) lastHeld(n int) int {
	if n < 0 {
		return noObject
	}
	last := n
	for i := n; g.link[i] >= 0 && g.object(g.link[i]).CoordUse() != OnMap; {
		i = g.link[i]
		if g.isHeldBy(i, n) {
			last = i
		}
	}
	return last
}

// MoveObj puts object n (and what it holds) on the map at (x, y, z); it returns n or
// noObject when it cannot be moved.
// C: seg_1184:1DAF (MoveObj)
func (g *Game) MoveObj(n, toX, toY, toZ int) int {
	if n < 0 {
		return noObject
	}
	if g.StatusDisplay == cmdStatusEquipment && g.isHeldBy(n, int(g.Party[g.statusMember])) {
		g.StatusDirty = true
	}
	o := g.object(n)
	t := o.Type()
	if t == ObjAmuletOfSubmission {
		return noObject
	}
	if t == ObjSilverSerpent && n < npcCount {
		deltaX, deltaY := o.Pos.X-toX, o.Pos.Y-toY
		facing := o.Frame()
		// fix: the original indexes the direction tables with the frame unchecked
		if deltaX != 0 && deltaY != 0 || deltaX == 0 && deltaY == 0 || abs(deltaX) > 1 || abs(deltaY) > 1 ||
			facing < 8 && dirStepX[facing] == deltaX && dirStepY[facing] == deltaY {
			return noObject
		}
	}
	toX &= coordMaskFor(toZ)
	toY &= coordMaskFor(toZ)
	curX, curY := o.Pos.X, o.Pos.Y
	prev := g.previousInList(n)
	if prev < 0 {
		return noObject
	}
	mx, my := int(g.MapX), int(g.MapY)
	if mx-viewCenter <= curX && mx+viewCenter >= curX && my-viewCenter <= curY && my+viewCenter >= curY ||
		mx-viewCenter <= toX && mx+viewCenter >= toX && my-viewCenter <= toY && my+viewCenter >= toY {
		g.viewChanged = true
	}
	last := g.lastHeld(n)
	g.link[prev] = g.link[last]
	g.unlinkFromArea(o.Pos.X, o.Pos.Y, o.Pos.Z, n, g.link[prev])
	after := g.listPosition(toX, toY, true)
	g.link[last] = g.link[after]
	g.link[after] = n
	o.SetCoordUse(OnMap)
	o.Pos = Coord{toX, toY, toZ}
	if i, ok := g.areaIndex(toX, toY); ok {
		g.areaObjects[i] = n
	}
	switch {
	case t == ObjSlime:
		g.updateSlimeFrames()
	case t == ObjSilverSerpent && n < npcCount:
		g.moveSerpent(n, curX, curY, false)
	case n < npcCount && isBigCreature(t):
		g.moveBigCreature(n, curX, curY, false)
	case t == ObjCorpser:
		o.SetFrame(0)
	}
	if n < npcCount && g.gameLoaded {
		g.NPCs[n].Flags &^= flagSkipTest
		g.checkStepOn(n)
	}
	return n
}

// InsertObj puts object n (and what it holds) into dest, as how (Contained, InInventory or
// Equipped).
// C: seg_1184:20DD (InsertObj)
func (g *Game) InsertObj(n, dest int, how ObjStatus) int {
	if n < 0 || dest < 0 {
		return noObject
	}
	o := g.object(n)
	if o.Type() == ObjAmuletOfSubmission && !g.allowSubmissionAmulet {
		return noObject
	}
	if n == dest || g.isHeldBy(dest, n) {
		return noObject
	}
	prev := g.previousInList(n)
	last := g.lastHeld(n)
	g.link[prev] = g.link[last]
	g.link[last] = g.link[dest]
	g.link[dest] = n
	g.unlinkFromArea(o.Pos.X, o.Pos.Y, o.Pos.Z, n, g.link[prev])
	o.SetCoordUse(how)
	o.Pos.SetOwner(dest)
	return n
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// resetObjectLists empties the object list and puts every slot on the free lists.
// C: seg_1184:3B1D (C_1184_3B1D)
func (g *Game) resetObjectLists() {
	for i := range objectCount - 1 {
		g.link[i] = i + 1
	}
	g.link[objectCount-1] = noObject
	g.link[npcCount-1] = noObject
	g.freeMonsters, g.freeMonsterCount = firstMonster, monsterSlots
	g.freeObjects, g.freeObjectCount = firstFreeObject, objectCount-firstFreeObject
	g.link[mapList] = noObject
	for i := range g.loadedRegions {
		g.loadedRegions[i] = noObject
	}
	g.topLeftRegion, g.loadedLevel = noObject, noObject
}

// linkCharacters puts the characters of a restored game in the list and takes the
// monster slots in use off the free list.
// C: seg_1184:3B7D (C_1184_3B7D)
func (g *Game) linkCharacters() {
	lastFree := mapList
	for slot := range npcCount {
		if g.object(slot).Shape == 0 {
			if slot >= firstMonster {
				lastFree = slot
			}
			continue
		}
		if slot >= firstMonster {
			if lastFree == mapList {
				g.freeMonsters = slot + 1
			} else {
				g.link[lastFree] = slot + 1
			}
			g.freeMonsterCount--
		}
		after := mapList
		for ; g.link[after] >= 0 && g.compareListOrder(g.link[after], slot) < 0; after = g.link[after] {
		}
		g.link[slot] = g.link[after]
		g.link[after] = slot
	}
}

// compareListOrder orders two objects as the list keeps them (negative when a comes first),
// placing held objects by their holder; noObject goes last.
// C: seg_1184:29C4 (C_1184_29C4)
func (g *Game) compareListOrder(a, b int) int {
	if a == noObject {
		return 1
	}
	if b == noObject {
		return -1
	}
	ha, hb := a, b
	for g.object(ha).CoordUse() == Contained {
		ha = g.object(ha).Pos.Owner()
	}
	for g.object(hb).CoordUse() == Contained {
		hb = g.object(hb).Pos.Owner()
	}
	if g.object(ha).CoordUse() != OnMap && g.object(b).CoordUse() == OnMap {
		return -1
	}
	if g.object(hb).CoordUse() != OnMap && g.object(a).CoordUse() == OnMap {
		return 1
	}
	if g.object(ha).Status&InInventory != 0 {
		ha = g.object(ha).Pos.Owner()
	}
	if g.object(hb).Status&InInventory != 0 {
		hb = g.object(hb).Pos.Owner()
	}
	pa, pb := g.object(ha).Pos, g.object(hb).Pos
	d := pa.Y - pb.Y
	if d == 0 {
		d = pa.X - pb.X
	}
	if d == 0 {
		d = pb.Z - pa.Z
	}
	if ha == hb {
		if g.object(ha).CoordUse() == OnMap {
			d = -1
		} else if g.object(hb).CoordUse() == OnMap {
			d = 1
		}
	}
	return d
}
