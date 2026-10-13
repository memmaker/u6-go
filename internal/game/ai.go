package game

import "github.com/memmaker/u6-go/internal/sound"

// The AI: what the characters the computer moves do each turn, the paths NPCs follow to
// their schedule, and the party's leader (seg_1E0F).

// Messages (seg_356A).
const (
	arrestMsg   = "\n\"Thou art under arrest!\"\n\n\"Wilt thou come quietly?\"\n\n:"
	goToJailMsg = "Yes\n\nThe guard strikes thee unconscious!\n\nThou dost awaken to...\n"
	defendMsg   = "No\n\n\"Then defend thyself, rogue!\"\n"
	stealMsg    = "The %s stole some gold!\n"
	darknessMsg = "\nAn unending darkness engulfs thee...\n"
	kalLorMsg   = "\nA voice in the darkness intones, \"KAL LOR!\"\n"
)

// pathfinder is the state of the path search, in the scratch buffer of the original
// (t40x40, PTH_xxx).
type pathfinder struct {
	// cost is the cost of reaching each place of the area, with fromGoal marking the
	// places reached from the goal; unreached places are unreached (PTH_map)
	cost [areaSize][areaSize]byte
	// rank chains the queued places by cost through next (PTH_rank, PTH_link), with
	// the place of each queue slot in queueX, queueY (PTH_x, PTH_y)
	rank           [queueSlots]int
	next           [queueSlots]int
	queueX, queueY [queueSlots]byte
	// resistance is the cost of crossing each place of the area (PTH_resist)
	resistance [areaSize][areaSize]byte
	// found ends the search (PTH_Found); free is the first free queue slot (PTH_Top);
	// current and highest the lowest and highest queued cost (PTH_Cur, PTH_Min)
	found            bool
	free             int
	current, highest int
	// edge is the side of the area the path leaves by, or -1 (PTH_direct); exitX, exitY
	// is where (PTH_AnX, PTH_AnY)
	edge         int
	exitX, exitY int
	// seek is the type of object looked for, or -1 (PTH_object)
	seek int
	// slot is the path slot being filled (PTH_Index)
	slot int
	// teleported and searched count the NPCs sent home and the paths searched this turn
	// (D_17A5, D_17A7)
	teleported, searched int
}

const (
	queueSlots = 256
	unreached  = 255
	fromGoal   = 128
	costMask   = fromGoal - 1
	impassable = 255
	// maxSearches is how many NPCs may search a path or be sent home in a turn.
	maxSearches = 3
	// path steps: the direction in the low 2 bits, the count above
	pathDirMask   = 3
	pathCountStep = 4
	pathMaxSteps  = pathSteps - 1
	// the sides of the area (PTH_direct): north, east, south, west
	edgeNorth = 0
	edgeEast  = 1
	edgeSouth = 2
	edgeWest  = 3
)

// the steps of the four directions of the path search (D_185A, D_185E)
var (
	pathStepX = [4]int{0, 1, 0, -1}
	pathStepY = [4]int{-1, 0, 1, 0}
)

// seekTypes are the objects an NPC walks to (D_1799).
var seekTypes = [...]ObjectType{ObjChair, ObjBed, ObjTable, ObjPullChain}

// Seeked objects (indexes into seekTypes).
const (
	seekChair = 0
	seekTable = 2
	seekChain = 3
)

// randomBias returns a small random number around 0.
// C: seg_1E0F:31C7 (C_1E0F_31C7)
func (g *Game) randomBias() int {
	n := 0
	for g.rand(0, 1) != 0 {
		n++
	}
	if g.rand(0, 1) != 0 {
		n = -n
	}
	return n
}

// loiter sometimes takes a step around (x, y).
// C: seg_1E0F:33C4 (C_1E0F_33C4)
func (g *Game) loiter(n, x, y int) {
	if g.rand(0, 7) != 0 {
		g.subMove(n, waitMovePoints)
		return
	}
	p := g.object(n).Pos
	deltaX, deltaY := p.X-x, p.Y-y
	var dir int
	if abs(deltaY)-abs(deltaX) < g.randomBias() {
		dir = dirWest
		if g.randomBias() > deltaX {
			dir = dirEast
		}
	} else {
		dir = dirNorth
		if g.randomBias() > deltaY {
			dir = dirSouth
		}
	}
	g.TryStraightMove(n, dir, true)
}

// wander sometimes takes a random step; a party member drifts back to the leader.
// C: seg_1E0F:37DB (C_1E0F_37DB)
func (g *Game) wander(n int) {
	if g.NPCs[n].Mode != AIImmobile {
		p := g.object(n).Pos
		if g.isPlayerControlled(n) && (p.X != int(g.MapX) || p.Y != int(g.MapY)) && g.rand(0, 3) == 0 {
			g.TryMoveTo(n, int(g.MapX), int(g.MapY))
			return
		}
		if g.rand(0, 7) == 0 {
			g.TryStraightMove(n, g.rand(0, 3)<<1, true)
			return
		}
	}
	g.subMove(n, waitMovePoints)
}

// scheduled returns where n's schedule entry sends it.
func (g *Game) scheduled(n int) (Coord, AIMode) {
	e := g.schedule[int(g.schedulePointer[n])+int(g.NPCs[n].ScheduleIndex)]
	return decodeCoord(e.Pos), AIMode(e.Action)
}

// plateNear returns 1 plus the direction of a plate next to (x, y, z), or 0.
// C: seg_1E0F:2125 (C_1E0F_2125)
func (g *Game) plateNear(x, y, z int) int {
	for d := 0; d < len(dirStepX); d += 2 {
		for i := g.FindLoc(x+dirStepX[d], y+dirStepY[d], z); i >= 0; i = g.NextLoc() {
			if g.object(i).Type() == ObjPlate {
				return d + 1
			}
		}
	}
	return 0
}

// furnitureAt returns the object of kind seek under n, or -1.
// C: seg_1E0F:2184 (C_1E0F_2184)
func (g *Game) furnitureAt(n, seek int) int {
	p := g.object(n).Pos
	want := seekTypes[seek]
	for i := g.FindLoc(p.X, p.Y, p.Z); i >= 0; i = g.NextLoc() {
		t := g.object(i).Type()
		switch want {
		case ObjTable:
			if t == ObjChair && g.plateNear(p.X, p.Y, p.Z) != 0 {
				return i
			}
		case ObjChair:
			const throneSeat = 2
			if t == ObjChair || t == ObjThrone && g.object(i).Frame()-g.locPart == throneSeat {
				return i
			}
		default:
			if t == want {
				return i
			}
		}
	}
	return noObject
}

// atDestination starts what an NPC does where its schedule took it, or sends it there.
// C: seg_1E0F:2276 (__AtDestination)
func (g *Game) atDestination(n int) {
	o, npc := g.object(n), &g.NPCs[n]
	pos, action := g.scheduled(n)
	there := pos == o.Pos
	npc.Mode = action
	const bedHead, bedHeadDouble = 0, 6
	switch {
	case action == AISleep:
		there = true
		npc.Status |= Asleep
		if g.isOutOfArea(n) {
			npc.Mode = AISchedule
			break
		}
		bed := noObject
		for i := g.FindLoc(o.Pos.X, o.Pos.Y, o.Pos.Z); i >= 0; i = g.NextLoc() {
			if f := g.object(i).Frame() - g.locPart; g.object(i).Type() == ObjBed && (f == bedHead || f == bedHeadDouble) {
				o.Shape = shapeOf(ObjPersonSleeping, 0)
				if f != bedHead {
					o.SetFrame(1)
				}
				bed = i
				break
			}
		}
		if bed < 0 {
			g.setUnconscious(n)
		}
	default:
		npc.Status &^= Asleep
		o.Shape = npc.OrigShape
		switch {
		case action == AISit || action == AIPlay:
			there = true
			if g.isOutOfArea(n) {
				npc.Mode = AISchedule
				break
			}
			if chair := g.furnitureAt(n, seekChair); chair >= 0 {
				if action == AIPlay {
					const playing = 2
					g.object(chair).SetFrame(playing)
					g.turnTo(n, dirSouth)
					o.Shape = shapeOf(ObjMusician2, 0)
					const facing = 4
					o.SetFrame(facing)
				} else {
					g.turnTo(n, dirNorth)
				}
			}
		case action == AIEat:
			there = true
			if g.isOutOfArea(n) {
				npc.Mode = AISchedule
				break
			}
			if chair := g.furnitureAt(n, seekTable); chair >= 0 {
				c := g.object(chair).Pos
				plate := g.plateNear(c.X, c.Y, c.Z)
				g.object(chair).SetFrame(plate >> 1)
				g.turnTo(n, plate-1)
			} else {
				g.turnTo(n, dirSouth)
			}
		case action == AIRingBell:
			there = true
			if g.isOutOfArea(n) {
				npc.Mode = AISchedule
			} else {
				g.furnitureAt(n, seekChain)
			}
		case action >= AIStandNorth && action <= AIGuardWest:
			const sides = 3
			d := int(action-AIStandNorth) & sides << 1
			npc.SetDirection(d)
			g.turnTo(n, d)
		}
	}
	if !there {
		npc.Mode = AIFindPath
		if o.Type() == ObjMusician2 {
			o.Shape = shapeOf(ObjMusician, 0)
			g.turnTo(n, dirSouth)
		}
	}
}

// pathArea returns the top left corner of the area paths are searched in.
func (g *Game) pathArea() (int, int) {
	return (int(g.MapX) - areaMargin) & areaAlign, (int(g.MapY) - areaMargin) & areaAlign
}

func inArea(x, y int) bool { return x >= 0 && x < areaSize && y >= 0 && y < areaSize }

// tracePath writes the path through (x, y) into the path slot.
// C: seg_1E0F:25F9 (C_1E0F_25F9)
func (g *Game) tracePath(x, y int) {
	pf := &g.paths
	path := &g.Paths[pf.slot]
	startX, startY := x, y
	steps, dir := 0, 0
	path[0] = 0
	lowest := func(onlyGoal bool) int {
		best, bestDir := unreached, 0
		for i := range pathStepX {
			nx, ny := x+pathStepX[i], y+pathStepY[i]
			if !inArea(nx, ny) {
				continue
			}
			c := int(pf.cost[ny][nx])
			if onlyGoal && c < fromGoal {
				continue
			}
			if c < best || c == best && dir == i {
				best, bestDir = c, i
			}
		}
		return bestDir
	}
	// back from (x, y) to the start, building the path from its end
	for pf.cost[y][x] != 0 {
		dir = lowest(false)
		x += pathStepX[dir]
		y += pathStepY[dir]
		if steps == 0 || int(path[0])&pathDirMask != dir^2 {
			if steps < pathMaxSteps {
				steps++
			}
			copy(path[1:steps+1], path[:steps])
			path[0] = byte(dir ^ 2 + pathCountStep)
		} else {
			path[0] += pathCountStep
		}
	}
	if pf.seek < 0 && pf.edge < 0 {
		// on from (x, y) to the goal
		x, y = startX, startY
		pf.cost[y][x] = unreached
		for pf.cost[y][x] > fromGoal {
			dir = lowest(true)
			x += pathStepX[dir]
			y += pathStepY[dir]
			if steps >= pathMaxSteps {
				break
			}
			if steps == 0 || int(path[steps-1])&pathDirMask != dir {
				path[steps] = byte(dir + pathCountStep)
				steps++
			} else {
				path[steps-1] += pathCountStep
			}
		}
	}
	path[steps] = 0
}

// visit queues place (x, y) of the area at cost; search is fromGoal for the search from
// the goal.
// C: seg_1E0F:2A74 (C_1E0F_2A74)
func (g *Game) visit(cost, x, y, search int) {
	pf := &g.paths
	if pf.found || cost > costMask || !inArea(x, y) {
		return
	}
	if c := pf.cost[y][x]; c != unreached && int(c)&fromGoal != search {
		pf.found = true
		g.tracePath(x, y)
		return
	}
	if pf.seek >= 0 {
		areaX, areaY := g.pathArea()
		for i := g.FindLoc(x+areaX, y+areaY, int(g.MapZ)); i >= 0; i = g.NextLoc() {
			t := g.object(i).Type()
			if ObjectType(pf.seek) == ObjTable && t == ObjChair && g.plateNear(x+areaX, y+areaY, int(g.MapZ)) != 0 ||
				ObjectType(pf.seek) != ObjTable && t == ObjectType(pf.seek) {
				pf.found = true
				g.tracePath(x, y)
				return
			}
		}
	}
	if pf.edge >= 0 {
		if x == 0 && pf.edge == edgeWest || x == areaSize-1 && pf.edge == edgeEast ||
			y == 0 && pf.edge == edgeNorth || y == areaSize-1 && pf.edge == edgeSouth {
			pf.found = true
			pf.exitX, pf.exitY = x, y
			g.tracePath(x, y)
			return
		}
	}
	if int(pf.cost[y][x])&costMask <= cost {
		return
	}
	slot := pf.free
	if slot < 0 {
		return
	}
	pf.highest = max(pf.highest, cost)
	pf.free = pf.next[slot]
	pf.next[slot] = pf.rank[cost]
	pf.queueX[slot], pf.queueY[slot] = byte(x), byte(y)
	pf.rank[cost] = slot
	pf.cost[y][x] = byte(cost | search)
}

// reversePath turns path slot around.
// C: seg_1E0F:2C9D (__ResetPath)
func (g *Game) reversePath(slot int) {
	path := &g.Paths[slot]
	n := 0
	for n < pathSteps && path[n] != 0 {
		n++
	}
	for i := 0; i < (n+1)/2; i++ {
		a := path[i] ^ 2
		path[i] = path[n-1-i] ^ 2
		path[n-1-i] = a
	}
}

// resistanceAt is the cost of crossing (x, y) of the area; outside it, nothing passes.
func (pf *pathfinder) resistanceAt(x, y int) int {
	if !inArea(x, y) {
		return impassable
	}
	return int(pf.resistance[y][x])
}

// findPath searches a path for n to (x, y), or, with x < 0, to an object of type y.
// C: seg_1E0F:2D37 (C_1E0F_2D37)
func (g *Game) findPath(n, x, y int) {
	pf := &g.paths
	pf.found = false
	swapped := false
	pf.edge = -1
	areaX, areaY := g.pathArea()
	p := g.object(n).Pos
	sx, sy := p.X-areaX, p.Y-areaY
	if x >= 0 {
		x -= areaX
		y -= areaY
		pf.seek = -1
		if sx == x && sy == y {
			g.atDestination(n)
			return
		}
		if !(p.Z == int(g.MapZ) && inArea(sx, sy) && inArea(x, y)) {
			if pf.teleported >= maxSearches {
				return
			}
			g.sendHome(n)
			if g.NPCs[n].Mode != AIFindPath {
				return
			}
			if !inArea(sx, sy) {
				x, sx = sx, x
				y, sy = sy, y
				swapped = true
			}
			const centre = areaSize / 2
			if abs(x-centre) > abs(y-centre) {
				pf.edge = edgeEast
				if x < centre {
					pf.edge = edgeWest
				}
			} else {
				pf.edge = edgeSouth
				if y < centre {
					pf.edge = edgeNorth
				}
			}
		}
	} else {
		pf.seek = y
	}
	pf.searched++
	g.computeResistance()
	for i := range pf.cost {
		for j := range pf.cost[i] {
			pf.cost[i][j] = unreached
		}
	}
	for i := range pf.next {
		pf.next[i] = i + 1
		pf.rank[i] = -1
	}
	pf.next[queueSlots-1] = -1
	pf.free, pf.current, pf.highest = 0, 0, 0
	g.visit(0, sx, sy, 0)
	if pf.seek == -1 && pf.edge == -1 {
		g.visit(0, x, y, fromGoal)
	}
	for pf.rank[pf.current] >= 0 && !pf.found {
		slot := pf.rank[pf.current]
		pf.rank[pf.current] = pf.next[slot]
		x, y = int(pf.queueX[slot]), int(pf.queueY[slot])
		pf.next[slot] = pf.free
		pf.free = slot
		c := int(pf.cost[y][x])
		search := c & fromGoal
		c &= costMask
		g.visit(pf.resistanceAt(x-1, y)+c, x-1, y, search)
		g.visit(pf.resistanceAt(x+1, y)+c, x+1, y, search)
		g.visit(pf.resistanceAt(x, y-1)+c, x, y-1, search)
		g.visit(pf.resistanceAt(x, y+1)+c, x, y+1, search)
		if pf.rank[pf.current] < 0 {
			for pf.current < pf.highest && pf.rank[pf.current] < 0 {
				pf.current++
			}
		}
	}
	npc := &g.NPCs[n]
	switch {
	case pf.found:
		if swapped {
			g.gameLoaded = false
			g.MoveObj(n, pf.exitX+areaX, pf.exitY+areaY, p.Z)
			g.gameLoaded = true
			g.reversePath(pf.slot)
		}
		npc.Mode = AIOnPath
		npc.Leader = byte(pf.slot)
		g.PathObject[pf.slot] = uint8(n)
		g.PathCounter[pf.slot] = 0
		g.PathTries[pf.slot] = 0
	case npc.Mode == AIFindPath:
		npc.Mode = AIPathLost
	case npc.Mode == AISeekObj:
		npc.Mode = AISchedule
	}
}

// addResistance adds the cost of an object of tile at (x, y) of the area.
// C: seg_1E0F:4265 (C_1E0F_4265)
func (g *Game) addResistance(tile, x, y int) {
	r := &g.paths.resistance[y][x]
	terrain := g.TerrainType[tile]
	switch {
	case *r == impassable:
		if g.TerrainType[g.AreaTiles[y*areaSize+x]]&terrainWet != 0 && terrain&terrainImpassable == 0 {
			*r = terrain>>terrainCostShift + 1
		}
	case terrain&terrainImpassable != 0:
		*r = impassable
	case terrain&terrainWall == 0:
		*r = byte(min(int(*r)+int(terrain>>terrainCostShift), impassable))
	}
}

// computeResistance works out the cost of crossing each place of the area.
// C: seg_1E0F:436A (__ComputeResistance)
func (g *Game) computeResistance() {
	pf := &g.paths
	areaX, areaY := g.pathArea()
	for y := range areaSize {
		for x := range areaSize {
			tile := int(g.AreaTiles[y*areaSize+x])
			if g.isImpassable(tile) {
				pf.resistance[y][x] = impassable
			} else {
				pf.resistance[y][x] = g.TerrainType[tile]>>terrainCostShift + 1
			}
		}
	}
	for i := g.SearchArea(areaX, areaY, areaX+areaSize-1, areaY+areaSize-1); i >= 0; i = g.NextArea() {
		o := g.object(i)
		x, y := o.Pos.X-areaX, o.Pos.Y-areaY
		t := o.Type()
		tile := g.tileFrame(i)
		flags := g.TileFlag[tile]
		r := &pf.resistance[y][x]
		switch {
		case t >= ObjOakenDoor && t <= ObjSteelDoor:
			const lockedFrames = 8
			if o.Frame() < lockedFrames && *r < impassable {
				*r++
			} else {
				*r = impassable
			}
			if flags&tileDoubleHeight != 0 {
				if y != 0 {
					pf.resistance[y-1][x] = impassable
				}
			} else if x != 0 {
				pf.resistance[y][x-1] = impassable
			}
		case t == ObjPassthrough || t == ObjPassthrough2:
			if *r < impassable {
				*r++
			}
		case i >= npcCount:
			g.addResistance(tile, x, y)
			switch {
			case flags&tileDoubleWidth != 0:
				if x != 0 {
					g.addResistance(tile-1, x-1, y)
				}
				if flags&tileDoubleHeight != 0 {
					if y != 0 {
						g.addResistance(tile-2, x, y-1)
					}
					if x != 0 && y != 0 {
						g.addResistance(tile-3, x-1, y-1)
					}
				}
			case flags&tileDoubleHeight != 0:
				if y != 0 {
					g.addResistance(tile-1, x, y-1)
				}
			}
		}
	}
}

// sendHome puts n straight at its schedule's place when neither is in sight.
// C: seg_1E0F:291C (C_1E0F_291C)
func (g *Game) sendHome(n int) {
	p := g.object(n).Pos
	inSight := func(x, y int) bool {
		return !g.allowNPCTeleport && p.Z == int(g.MapZ) && g.onScreen(x, y)
	}
	if inSight(p.X, p.Y) {
		return
	}
	home, _ := g.scheduled(n)
	if inSight(home.X, home.Y) {
		return
	}
	npc := &g.NPCs[n]
	if npc.Mode >= AIOnPath && npc.Mode < AIPathLost {
		g.PathObject[npc.Leader] = 0
	}
	g.gameLoaded = false
	g.MoveObj(n, home.X, home.Y, home.Z)
	g.gameLoaded = true
	g.atDestination(n)
	g.subMove(n, attackMovePoints)
	g.paths.teleported++
}

// followPath takes n one step along its path to (x, y).
// C: seg_1E0F:387D (__DoOnPath)
func (g *Game) followPath(n, x, y int) {
	npc := &g.NPCs[n]
	slot := int(npc.Leader)
	step := int(g.Paths[slot][g.PathCounter[slot]])
	switch {
	case step == 0:
		g.PathObject[slot] = 0
		if g.distance(n, x, y) < meleeRange {
			g.atDestination(n)
		} else {
			npc.Mode = AIFindPath
		}
		g.subMove(n, waitMovePoints)
	case g.TryStraightMove(n, step&pathDirMask<<1, true):
		g.PathTries[slot]++
		if step>>2 <= int(g.PathTries[slot]) {
			g.PathCounter[slot]++
			g.PathTries[slot] = 0
		}
		npc.Mode = AIOnPath
	case npc.Mode == AIOnPath:
		npc.Mode = AIPathBlocked
		npc.MovePoints = 0
	case npc.Mode == AIPathBlocked:
		npc.Mode = AIPathBlocked2
		npc.MovePoints = 0
	case npc.Mode == AIPathBlocked2:
		npc.Mode = AIPathLost
		npc.MovePoints = 0
		g.PathObject[slot] = 0
		// fix: the original steps by the raw path byte, past the direction table
		p := g.object(n).Pos
		if d := step & 7; p.X+dirStepX[d] == x && p.Y+dirStepY[d] == y {
			g.atDestination(n)
		}
	}
}

// approachParty walks n to the nearest party member it sees; it returns that member
// when n is next to it, else -1.
// C: seg_1E0F:39B3 (C_1E0F_39B3)
func (g *Game) approachParty(n int) int {
	picked := noObject
	p := g.object(n).Pos
	const farthest = 50
	best := farthest
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		if g.object(m).Shape == 0 || g.NPCs[m].Has(Dead) || !g.canSee(n, m) {
			continue
		}
		q := g.object(m).Pos
		d := (q.X-p.X)*(q.X-p.X) + (q.Y-p.Y)*(q.Y-p.Y)
		if d < best || d == best && g.rand(0, 1) != 0 {
			best, picked = d, m
		}
	}
	if picked < 0 {
		g.TryMoveTo(n, g.partyGravityX, g.partyGravityY)
		return picked
	}
	q := g.object(picked).Pos
	g.TryMoveTo(n, q.X, q.Y)
	if p := g.object(n).Pos; !closeEnough(1, q.X, q.Y, p.X, p.Y) {
		picked = noObject
	}
	return picked
}

// Jail: where the guards take the party, its desk and its door.
var (
	jailCell = Coord{X: 231, Y: 186}
	jailDesk = Coord{X: 235, Y: 183}
	jailDoor = Coord{X: 231, Y: 184}
)

// arrest has guard n arrest the party.
// C: seg_1E0F:3B60 (C_1E0F_3B60)
func (g *Game) arrest(n int) {
	const lordBritish, metFlag = 5, 128
	if g.NPCs[lordBritish].TalkFlags&metFlag == 0 {
		g.NPCs[n].Mode = AIFindPath
		return
	}
	g.printf(arrestMsg)
	k := 0
	for k != 'N' && k != 'Y' {
		k = g.getch()
	}
	g.NPCs[n].Mode = AIFindPath
	if k == 'Y' {
		g.printf(goToJailMsg)
		g.redrawWorld()
		if g.InCombat != 0 {
			g.breakOff()
		}
		g.partyTeleport(jailCell.X, jailCell.Y, 0, 0)
		const morning, hour = 8, 60
		for g.Hour != morning {
			g.passTime(hour)
		}
		const jailKey = 9
		for i := range int(g.PartySize) {
			for _, k := range g.heldBy(int(g.Party[i])) {
				o := g.object(k)
				if o.Type() == ObjLockpick {
					g.DeleteObj(k)
				}
				if o.Type() == ObjKey2 && o.Quality() == jailKey {
					desk := g.FindLoc(jailDesk.X, jailDesk.Y, 0)
					for desk >= 0 && g.object(desk).Type() != ObjDesk {
						desk = g.NextLoc()
					}
					if desk >= 0 {
						g.InsertObj(k, desk, Contained)
					}
				}
				door := g.FindLoc(jailDoor.X, jailDoor.Y, 0)
				for door >= 0 && g.object(door).Type() != ObjSteelDoor {
					door = g.NextLoc()
				}
				if door >= 0 {
					const locked = 9
					g.object(door).SetFrame(locked)
				}
			}
		}
		g.screenFade = true
		g.ShowObjects()
		g.playAreaSong()
		g.OtherAnimations()
	} else {
		g.printf(defendMsg)
		g.reportCrime()
		g.NPCs[n].Mode = AIAssault
		g.setAlignment(n, alignEvil)
		for i := range npcCount {
			if g.NPCs[i].Mode == AIArrest {
				g.NPCs[i].Mode = AIAssault
				g.setAlignment(i, alignEvil)
			}
		}
	}
	g.promptFlag = true
}

// stealGold has thief n pick victim's pocket.
// C: seg_1E0F:3D8D (C_1E0F_3D8D)
func (g *Game) stealGold(n, victim int) {
	if g.countObj(victim, ObjGold) == 0 {
		return
	}
	const least, most, odds = 10, 30, 30
	g.takeObj(victim, ObjGold, g.rand(least, most))
	if (g.intelligenceOf(victim)+odds-g.dexterityOf(n))/2 <= g.rand(1, odds) {
		g.printf(stealMsg, g.objectString(n))
	}
}

// newPathSlot returns a free path slot, or pathSlots when there is none.
// C: seg_1E0F:3E27 (__NewPathIndex)
func (g *Game) newPathSlot() int {
	i := 0
	for ; i < pathSlots; i++ {
		if n := g.PathObject[i]; n != 0 {
			if m := g.NPCs[n].Mode; m < AIOnPath || m > AIPathLost {
				g.PathObject[i] = 0
			}
		}
		if g.PathObject[i] == 0 {
			break
		}
	}
	return i
}

// npcTurn is the turn of a character the computer moves.
// C: seg_1E0F:3E6A (C_1E0F_3E6A)
func (g *Game) npcTurn(n int) {
	npc := &g.NPCs[n]
	if g.isPlayerControlled(n) && !g.object(n).Has(StatusCharmed) {
		g.stopCharm(n)
	}
	points := npc.MovePoints
	if g.SpellFx[spellTimeStop] != 0 || g.object(n).Pos.Z != int(g.MapZ) {
		npc.MovePoints = 0
		return
	}
	home, _ := g.scheduled(n)
	sx, sy := home.X, home.Y
	switch npc.Mode {
	case AIFront:
		g.front(n)
	case AIRear:
		g.rear(n)
	case AIFlank:
		g.flank(n)
	case AIBerserk:
		g.berserk(n)
	case AIAssault, AIVigilante:
		g.assault(n)
	case AIImmobile:
		g.immobile(n)
	case AITangle:
		g.tangle(n)
	case AIRetreat, AIShy, AIFear, AIFlee:
		g.retreat(n)
	case AILike:
		g.like(n)
	case AIUnfriendly:
		g.unfriendly(n)
	case AIArrest:
		if g.approachParty(n) >= 0 {
			g.arrest(n)
		}
	case AIBrawl:
		if g.rand(0, 3) == 0 {
			if o := g.pickOpponent(n); o >= 0 && g.distance(o, sx, sy) < nearPartyRange {
				if q := g.object(o).Pos; g.distance(n, q.X, q.Y) < meleeRange {
					g.showHit(o)
					g.TryMoveTo(n, sx, sy)
				} else {
					g.TryMoveTo(n, q.X, q.Y)
				}
				break
			}
		}
		g.loiter(n, sx, sy)
	case AIConverse, AIThief:
		if !closeEnough(nearPartyRange-1, sx, sy, int(g.MapX), int(g.MapY)) {
			g.loiter(n, sx, sy)
			break
		}
		if m := g.approachParty(n); m >= 0 {
			// the original changes the mode before testing it, so thieves only talk
			npc.Mode = AILoiter
			if npc.Mode == AIThief {
				g.stealGold(n, m)
			} else {
				g.talkToParty(n, m)
			}
		}
	case AIOnPath, AIPathBlocked, AIPathBlocked2:
		g.followPath(n, sx, sy)
	case AISeekObj:
		g.paths.slot = g.newPathSlot()
		if g.paths.slot < pathSlots {
			g.findPath(n, -1, int(seekTypes[npc.Leader]))
			g.subMove(n, waitMovePoints)
		} else {
			npc.MovePoints = 0
		}
	case AIGraze, AIWander:
		g.wander(n)
	case AILoiter, AIFarm:
		g.loiter(n, sx, sy)
	case AIGuard, AIGuard2, AIGuardNorth, AIGuardNorth + 1, AIGuardNorth + 2, AIGuardWest:
		if g.rand(0, 1) == 0 {
			g.subMove(n, waitMovePoints)
			break
		}
		dir := npc.Direction()
		if p := g.object(n).Pos; npc.Mode >= AIGuardNorth && p.X == sx && p.Y == sy {
			dir = int(npc.Mode-AIGuardNorth) << 1
		}
		before := npc.MovePoints
		if !g.TryStraightMove(n, dir, true) {
			npc.MovePoints = before
			g.TryStraightMove(n, dir^4, true)
		}
	case AIRingBell:
		if g.MustRingBell != 0 {
			g.MustRingBell = 0
			const strokes = 12
			g.bellRings = int(g.Hour) % strokes
			if g.bellRings == 0 {
				g.bellRings = strokes
			}
			g.setTileAnimation(int(g.BaseTile[ObjBell]), animForward)
		}
		g.subMove(n, waitMovePoints)
	default:
		g.subMove(n, waitMovePoints)
	}
	if npc.MovePoints == points {
		g.subMove(n, attackMovePoints)
	}
}

// talkToParty has n talk to party member m.
// C: seg_1E0F:3E08 (C_1E0F_3E08)
func (g *Game) talkToParty(n, m int) {
	// fix: the original passes the character as the party index
	for i := range int(g.PartySize) {
		if int(g.Party[i]) == m {
			g.talkTo(i, n, false)
		}
	}
	g.playAreaSong()
	g.promptFlag = true
}

// schedulePaths searches paths for the NPCs that must go somewhere.
// C: seg_1E0F:464A (C_1E0F_464A)
func (g *Game) schedulePaths() {
	pf := &g.paths
	idle := true
	pf.slot = g.newPathSlot()
	if pf.slot == pathSlots {
		return
	}
	pf.teleported, pf.searched = 0, 0
	for n := 2; n < firstMonster; n++ {
		if g.NPCs[n].Mode != AIFindPath {
			continue
		}
		if pf.teleported < maxSearches {
			g.sendHome(n)
		}
		if g.NPCs[n].Mode == AIFindPath && pf.searched < maxSearches {
			idle = false
			home, _ := g.scheduled(n)
			g.findPath(n, home.X, home.Y)
			for pf.slot < pathSlots && g.PathObject[pf.slot] != 0 {
				pf.slot++
			}
			if pf.slot == pathSlots {
				break
			}
		}
	}
	if idle {
		for n := 2; n < firstMonster; n++ {
			if g.NPCs[n].Mode == AIPathLost {
				g.NPCs[n].Mode = AIFindPath
				if pf.teleported < maxSearches {
					g.sendHome(n)
				}
			}
		}
	}
}

// computeGravity finds the centres of the party and of its enemies and their front
// fighters.
// C: seg_1E0F:4746 (C_1E0F_4746)
func (g *Game) computeGravity() {
	sumX, sumY, count := 0, 0, 0
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		if g.NPCs[m].Mode != AIFlank && g.NPCs[m].Mode != AIBerserk {
			count++
			sumX += g.object(m).Pos.X
			sumY += g.object(m).Pos.Y
		}
	}
	if count != 0 {
		g.partyGravityX = (int(g.PartySize)>>1 + sumX) / count
		g.partyGravityY = (int(g.PartySize)>>1 + sumY) / count
	} else {
		g.partyGravityX, g.partyGravityY = int(g.MapX), int(g.MapY)
	}
	sumX, sumY, count = 0, 0, 0
	const sightRange = 24
	for i := 1; i < npcCount; i++ {
		if g.object(i).Shape == 0 || g.NPCs[i].Has(Dead) || !g.attacksParty(i) {
			continue
		}
		q := g.object(i).Pos
		if g.NPCs[i].Mode == AIRetreat && !closeEnough(nearPartyRange, q.X, q.Y, int(g.MapX), int(g.MapY)) {
			continue
		}
		if closeEnough(sightRange-1, q.X, q.Y, int(g.MapX), int(g.MapY)) {
			count++
			sumX += q.X
			sumY += q.Y
		}
	}
	g.enemyCount = count
	lead := g.NPCs[g.Party[0]].Direction()
	if count != 0 {
		g.enemyGravityX = (count>>1 + sumX) / count
		g.enemyGravityY = (count>>1 + sumY) / count
		deltaX, deltaY := g.enemyGravityX-g.partyGravityX, g.enemyGravityY-g.partyGravityY
		g.enemyFront, g.enemyLeft, g.enemyRight = noObject, noObject, noObject
		bestFront := -1 << 15
		left, right, enemyLeft, enemyRight := 0, 0, 0, 0
		for i := range npcCount {
			if g.NPCs[i].Has(Dead) || g.NPCs[i].Mode != AIFront && g.NPCs[i].Mode != AICommand {
				continue
			}
			q := g.object(i).Pos
			depth := (q.X-g.partyGravityX)*deltaX + (q.Y-g.partyGravityY)*deltaY
			side := (q.X-g.partyGravityX)*deltaY - (q.Y-g.partyGravityY)*deltaX
			if g.isPlayerControlled(i) {
				if depth > bestFront {
					bestFront, g.partyFront = depth, i
				}
				if side > left {
					left, g.partyLeft = side, i
				}
				if side < right {
					right, g.partyRight = side, i
				}
			} else {
				if g.enemyFront < 0 {
					g.enemyFront = i
				}
				if side > enemyLeft {
					enemyLeft, g.enemyLeft = side, i
				}
				if side < enemyRight {
					enemyRight, g.enemyRight = side, i
				}
			}
		}
	} else {
		g.heardSomething = false
		g.enemyGravityX = int(g.MapX) + dirStepX[lead]
		g.enemyGravityY = int(g.MapY) + dirStepY[lead]
	}
	if g.enemyGravityX == g.partyGravityX && g.enemyGravityY == g.partyGravityY {
		g.enemyGravityX += dirStepX[lead]
		g.enemyGravityY += dirStepY[lead]
	}
}

// requiem are the notes played when the Avatar dies (D_1862).
var requiem = [...]int{300, 281, 300, 250, 281, 222, 250, 250}

// avatarDied plays the death of the Avatar and wakes the party at Lord British's.
// C: seg_1E0F:4AFA (C_1E0F_4AFA)
func (g *Game) avatarDied() {
	if int(g.Active) == partyMax {
		return
	}
	g.playSong(0)
	g.printf(darknessMsg)
	for _, f := range requiem {
		const units, steps, width, delta = 1, 20000, 1, 3
		speaker(sound.PlayWavedNote(f, units, steps, width, delta))
	}
	g.worldShown = true
	g.redrawWorld()
	const pause = 30
	g.osiDelay(pause)
	g.printf(kalLorMsg)
	g.resurrectParty()
}

// partyFails says whether n can not lead.
func (g *Game) partyFails(n int) bool {
	return g.incapacitated(n) || g.draggedUnder(n) || g.object(n).Has(StatusCharmed)
}

// checkActiveMember hands the lead to another member when the active one can not lead,
// and back to the Avatar when it can again.
// C: seg_1E0F:4B6A (C_1E0F_4B6A)
func (g *Game) checkActiveMember() {
	if g.Active > g.PartySize {
		return
	}
	if g.inVehicle() {
		g.object(0).Status &^= StatusCursed | StatusCharmed
		g.NPCs[0].Status &^= Dead | Poisoned | Asleep | Paralyzed
		g.NPCs[0].Flags &^= flagDraggedUnder
		g.setAlignment(0, alignGood)
	}
	const avatar = 1
	if g.NPCs[avatar].Has(Dead) {
		g.avatarDied()
	}
	if g.SoloMember >= 0 {
		if g.partyFails(int(g.Party[g.Active])) {
			g.SetPartyMode()
		}
		return
	}
	if g.partyFails(int(g.Party[g.Active])) {
		for i := range int(g.PartySize) {
			m := int(g.Party[i])
			if g.partyFails(m) || g.NPCs[m].Mode == AIMotionless {
				continue
			}
			g.NPCs[g.Party[g.Active]].Mode = AIFollow
			g.NPCs[m].Mode = AICommand
			g.Active = byte(i)
			g.soloLeader = i
			g.promptFlag = true
			p := g.object(m).Pos
			g.MapX, g.MapY = int16(p.X), int16(p.Y)
			g.enterArea()
			break
		}
	}
	if g.Active < g.PartySize && g.Active != 0 && g.SoloMember < 0 {
		lead := int(g.Party[0])
		if g.partyFails(lead) {
			return
		}
		if g.InCombat != 0 {
			npc := &g.NPCs[g.Party[g.Active]]
			npc.Mode = npc.CombatMode
		} else {
			g.NPCs[g.Party[g.Active]].Mode = AIFollow
			g.Active = 0
			g.soloLeader = 0
			g.promptFlag = true
			p := g.object(lead).Pos
			g.MapX, g.MapY = int16(p.X), int16(p.Y)
			g.enterArea()
		}
		g.NPCs[lead].Mode = AICommand
	}
}

// resurrectParty raises the dead party members and wakes the party at Lord British's.
// C: seg_101C:0ACC (C_101C_0ACC)
func (g *Game) resurrectParty() {
	// no one leads while the party is raised (-1 in the original)
	const noMember = byteMask
	g.Active = noMember
	const avatar = 1
	for n := range firstMonster {
		if g.isPlayerControlled(n) && g.NPCs[n].Has(Dead) {
			for b := g.link[npcCount]; b >= 0; b = g.link[b] {
				if t := g.object(b).Type(); (t == ObjDeadBody || t == ObjMouse) && g.object(b).Quality() == n {
					g.Selection.obj = b
					if n == avatar {
						g.raiseBody(b)
						npc := &g.NPCs[avatar]
						npc.Status &^= Dead
						g.object(avatar).Shape = npc.OrigShape
						npc.Magic = npc.Intelligence
						npc.Status &^= Dead | Poisoned | Asleep | Paralyzed | Protected
						const fullKarma, percent = 98, 100
						if g.Karma < fullKarma {
							npc.Experience = uint16(int(npc.Experience) * int(g.Karma) / percent)
						}
					} else {
						g.gameLoaded = false
						g.resurrect(b)
					}
					break
				}
			}
		}
		if g.NPCs[n].Mode == AIVigilante {
			g.NPCs[n].Mode = AIFindPath
		}
	}
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		npc := &g.NPCs[m]
		if npc.Has(Asleep) {
			npc.Status &^= Asleep
			g.object(m).Shape = npc.OrigShape
		}
		g.setAlignment(m, alignGood)
		g.object(m).Status &^= StatusCursed | StatusCharmed | StatusInvisible
		npc.Status &^= Dead | Poisoned | Asleep | Paralyzed
		npc.HitPoints = byte(g.maxHitPoints(m))
		npc.MovePoints = int8(npc.Dexterity)
	}
	for n := firstMonster; n < npcCount; n++ {
		if g.object(n).Type() == ObjGuard {
			g.DeleteObj(n)
		}
	}
	g.SpellFx = [len(g.SpellFx)]uint8{}
	g.hitList = g.hitList[:0]
	g.SetPartyMode()
	g.breakOff()
	const castleX, castleY = 307, 352
	g.partyTeleport(castleX, castleY, 0, 0)
	g.allowNPCTeleport = true
	const morningFrom, morningTo, afternoonFrom, afternoonTo = 8, 11, 14, 17 // the court hours
	for g.Hour < morningFrom || g.Hour > morningTo && g.Hour < afternoonFrom || g.Hour > afternoonTo {
		const quarter = 15
		g.passTime(quarter)
		g.schedulePaths()
	}
	g.allowNPCTeleport = false
	g.screenFade = true
	g.ShowObjects()
	g.sfx(sound.EffectFanfare, 0)
	g.OtherAnimations()
	g.playAreaSong()
	g.StatusDirty = true
	g.gameLoaded = true
}

// redrawWorld blanks the view and draws it again from scratch.
// C: seg_2FC1:19A1 (C_2FC1_19A1)
// C: seg_2FC1:193C (C_2FC1_193C), inlined
func (g *Game) redrawWorld() {
	const blank = 255 // TIL_0FF
	for y := range g.tileView {
		for x := range g.tileView[y] {
			g.tileView[y][x] = blank
			g.objectView[y][x] = 0
		}
	}
	for i := range g.objChain {
		g.objChain[i] = byte(i + 1)
	}
	g.viewChanged = false
	g.screenFade = true
	g.drawMap(true)
	g.refreshArea()
	g.computeView()
}
