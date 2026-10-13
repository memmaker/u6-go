package game

import (
	"image"

	"github.com/memmaker/u6-go/internal/sound"
)

// Moving characters: where they can go, how they turn, the party walking and following,
// and the turn scheduler (seg_1E0F).

// turns is the state of the movement module.
type turns struct {
	// followers are the party members already placed by MoveFollowers (D_EBD5)
	followers [partyMax + 1]byte
	// steppedOnDamage is set by passable for a place that hurts (D_17A9), avoidDamage
	// makes such places impassable (D_17B4)
	steppedOnDamage bool
	avoidDamage     bool
	// allowPartySwap lets a party member walk through another (D_17B2)
	allowPartySwap bool
	// turnParts makes turnTo move the other parts of big creatures too (D_17A3)
	turnParts bool
	// warnNoQuest lets passable say why the shrine is closed (D_17AD)
	warnNoQuest bool
	// moveBlocked says the party could not move (D_17AE)
	moveBlocked bool
	// tryHarder is D_17AB, set while an NPC follows a path
	tryHarder bool
	// soloLeader is the party member that leads (D_EBF3)
	soloLeader int
	// allowNPCTeleport lets NPCs far away jump to their schedule (AllowNPCTeleport)
	allowNPCTeleport bool
	// formation is how the party lines up behind its leader (the port's, formation.go)
	formation formation
}

func (t *turns) initTurns() { t.allowPartySwap = true }

// Messages (seg_356A).
const (
	notOnQuestMsg = "\n\"Thou art not upon a Sacred Quest!\nPassage denied!\"\n"
	noWindMsg     = "Thou canst not move without wind!\n"
	argghMsg      = "\n%s:\n%>ARGH!\n"
	regurgitated  = " regurgitated!\n"
	partyModeOnly = "You must be in party mode to enter.\n"
)

// questVariable is the conversation variable 'Q', set while the avatar is on the quest.
const questVariable = 'Q' - '0' - 7

// incapacitated says whether a character is dead, asleep or paralyzed (Isbis_0016).
func (g *Game) incapacitated(n int) bool { return g.NPCs[n].Has(Dead | Asleep | Paralyzed) }

// draggedUnder says whether a corpser holds a character.
func (g *Game) draggedUnder(n int) bool { return g.NPCs[n].Flags&flagDraggedUnder != 0 }

// passable says whether character n can step on (x, y).
// C: seg_1E0F:000F (C_1E0F_000F)
func (g *Game) passable(n, x, y int) bool {
	ok := false
	g.steppedOnDamage = false
	const doubleBoth = tileDoubleWidth | tileDoubleHeight
	big := g.TileFlag[g.tileFrame(n)]&doubleBoth == doubleBoth
	// fix: while the party is raised Active is -1 and C reads before the party table.
	active := noObject
	if int(g.Active) < len(g.Party) {
		active = int(g.Party[g.Active])
	}
	o := g.object(n)
	t := o.Type()
	cls := monsterClass(t)
	flier := classHas(cls, flies)
	boat := t == ObjSkiff || t == ObjRaft
	swimmer := classHas(cls, swims) || boat
	spirit := classHas(cls, ethereal)
	const amphibian = 1 // the alligator's class
	walker := !swimmer && !flier || cls == amphibian
	terrain := g.TerrainType[g.GetTileAtXYZ(x, y, int(g.MapZ))]
	if t == ObjRotWorms {
		ok = terrain&terrainDamaging != 0
	} else {
		if walker && terrain&terrainImpassable == 0 ||
			swimmer && terrain&terrainImpassable != 0 && terrain&terrainWet != 0 ||
			boat && terrain&terrainWet != 0 ||
			flier && terrain&terrainWall == 0 ||
			spirit {
			ok = true
		}
		if terrain&terrainDamaging != 0 {
			g.steppedOnDamage = true
			if g.avoidDamage {
				return false
			}
		}
	}
	keepLooking := true
	for i := g.FindLoc(x, y, int(g.MapZ)); i >= 0; i = g.NextLoc() {
		if i == n {
			continue
		}
		it := g.object(i).Type()
		if it == ObjNothingGuardiansOfTheCodex && g.VarInt[questVariable] == 0 {
			if g.warnNoQuest {
				g.printf(notOnQuestMsg)
				g.warnNoQuest = false
				g.promptFlag = true
			}
			ok = false
			break
		}
		if (t >= ObjGiantScorpion || t == ObjShip) && i >= npcCount && it == t {
			// its own other half
			sides := g.TerrainType[g.tileFrame(n)]
			if sides&terrainNorth != 0 && o.Pos.Y > y || sides&terrainEast != 0 && o.Pos.X < x ||
				sides&terrainSouth != 0 && o.Pos.Y < y || sides&terrainWest != 0 && o.Pos.X > x {
				continue
			}
		}
		if t == ObjMouse && (it == ObjMouseHole || it == ObjBars || it == ObjPortcullis) {
			ok = true
			continue
		}
		tile := g.tileFrame(i) - g.locPart
		if big && g.locPart == 0 && g.TileFlag[tile]&doubleBoth != 0 {
			return false
		}
		if walker && g.isBreakthrough(tile) {
			ok = true
			if !g.isIgnored(tile) {
				break
			}
		}
		terr := g.TerrainType[tile]
		if terr&terrainDamaging != 0 {
			g.steppedOnDamage = true
			if g.avoidDamage {
				return false
			}
		}
		if t == ObjSilverSerpent && it == ObjSilverSerpent {
			// its own body
			if g.object(i).Quality() == o.Quality()+1 {
				keepLooking, ok = false, false
			}
		} else {
			passes := walker && terr&terrainImpassable == 0 ||
				swimmer && (terr&terrainImpassable != 0 && terr&terrainWet != 0 || g.isIgnored(tile)) ||
				boat && (terr&terrainWet != 0 || g.isIgnored(tile)) ||
				flier && terr&terrainWall == 0 && (i >= npcCount || !classHas(monsterClass(it), flies)) ||
				spirit && i >= npcCount
			if !passes {
				if g.isPlayerControlled(n) {
					if i != active && g.allowPartySwap && i < npcCount && g.isPlayerControlled(i) &&
						!(g.incapacitated(i) || g.draggedUnder(i)) {
						continue
					}
				} else if n < npcCount {
					door := it >= ObjOakenDoor && it <= ObjSteelDoor && g.object(i).Frame() < 8 ||
						it == ObjPassthrough || it == ObjPassthrough2
					passes = door && classHas(cls, opensDoors)
				}
			}
			if !passes {
				keepLooking, ok = false, false
			}
		}
		if i < npcCount && i != 0 {
			small := it == ObjBird || it == ObjMouse || it == ObjInsects || it == ObjRabbit || it == ObjCorpser
			if !small || it == t {
				ok = false
				break
			}
		}
		if terr&terrainWall != 0 && (t == ObjWingedGargoyle || t == ObjCyclops || t == ObjDragon || t == ObjHydra2) {
			keepLooking, ok = false, false
		}
		if !keepLooking {
			break
		}
	}
	return ok
}

// isOutOfArea says whether an object is outside the area.
// C: seg_1E0F:056E (IsOutOfArea)
func (g *Game) isOutOfArea(n int) bool {
	p := g.object(n).Pos
	return !(p.Z == int(g.MapZ) && (p.X-g.AreaX)&g.coordMask <= areaSize-1 && (p.Y-g.AreaY)&g.coordMask <= areaSize-1)
}

// blockedAt says whether (x, y) is impassable, by the map or an object that is not a
// party member.
// C: seg_1E0F:05D8 (C_1E0F_05D8)
func (g *Game) blockedAt(x, y int) bool {
	if g.isImpassable(g.GetTileAtXYZ(x, y, int(g.MapZ))) {
		return true
	}
	for obj := g.FindLoc(x, y, int(g.MapZ)); obj >= 0; obj = g.NextLoc() {
		if (obj >= npcCount || !g.isPlayerControlled(obj)) && g.isImpassable(g.tileFrame(obj)-g.locPart) {
			return true
		}
	}
	return false
}

// dragonTurnFrames are the frames of a dragon's parts for each direction (D_17CE).
var dragonTurnFrames = [16]int{8, 32, 16, 24, 26, 10, 34, 18, 20, 28, 12, 36, 38, 22, 30, 14}

// shipEnds are the sides the bow and stern of a ship show for each direction (D_17DE).
var shipEnds = [4]byte{terrainSouth, terrainWest, terrainNorth, terrainEast}

// face turns a creature with four facings toward dir: straight for the main directions,
// the nearest facing for the diagonals (MACRO_A).
func face(dir, facing int) int {
	if dir&1 == 0 {
		return dir >> 1
	}
	if (dir>>1-facing+1)&3 > 1 {
		return (facing + 2) & 3
	}
	return facing
}

// walkStep advances a three-step walk: from a foot to the middle, from the middle to the
// other foot.
func (npc *NPC) walkStep(step int) int {
	switch step {
	case 0:
		npc.Flags &^= flagWalking
		return 1
	case 1:
		if npc.Flags&flagWalking != 0 {
			return 0
		}
		return 2
	case 2:
		npc.Flags |= flagWalking
		return 1
	}
	return 1
}

// turnTo turns a character toward dir and advances its walk.
// C: seg_1E0F:0664 (C_1E0F_0664)
func (g *Game) turnTo(n, dir int) {
	o := g.object(n)
	npc := &g.NPCs[n]
	onChair := false
	seat, seatType := noObject, ObjectType(0)
	if !g.isOutOfArea(n) {
		for seat = g.FindLoc(o.Pos.X, o.Pos.Y, o.Pos.Z); seat >= 0; seat = g.NextLoc() {
			seatType = g.object(seat).Type()
			const throneSeat = 2
			if seatType == ObjChair || seatType == ObjThrone && g.object(seat).Frame()-g.locPart == throneSeat {
				onChair = true
				break
			}
		}
	}
	t := o.Type()
	frame := o.Frame()
	switch {
	case t >= ObjFighter && t <= ObjWoman || t >= ObjLordBritish && t <= ObjAvatar:
		var facing int
		if onChair {
			const sitting = 3
			frame = sitting
			facing = 2
			if seatType == ObjChair {
				facing = g.object(seat).Frame()
			}
		} else {
			facing = face(dir, frame>>2)
			frame = npc.walkStep(frame & stepMask)
		}
		o.SetFrame(frame + facing<<2)
	case t == ObjWingedGargoyle:
		const steps = 12
		facing := face(dir, frame/steps)
		switch frame % steps {
		case 3:
			npc.Flags &^= flagWalking
			frame = 7
		case 7:
			frame = 11
			if npc.Flags&flagWalking != 0 {
				frame = 3
			}
		case 11:
			npc.Flags |= flagWalking
			frame = 7
		default:
			frame = 7
		}
		o.SetFrame(frame + facing*steps)
	case t == ObjBird:
		o.SetFrame(g.rand(0, 2))
	case t == ObjGargoyle:
		const steps = 3
		facing := face(dir, frame/steps)
		o.SetFrame(npc.walkStep(frame%steps) + facing*steps)
	case t == ObjSheep || t == ObjDog || t == ObjDeer || t == ObjGiantRat || t == ObjSnake ||
		t == ObjGiantSpider || t >= ObjGiantScorpion || t == ObjWolf || t == ObjSeaSerpent ||
		t >= ObjDaemon && t <= ObjMongbat || t == ObjMusician2:
		facing := frame >> 1
		if t < ObjGiantScorpion || g.turnParts {
			if dir&1 != 0 {
				x, y := o.Pos.X, o.Pos.Y
				if g.blockedAt(x, y-dirStepY[dir]) {
					dir = dir&4 + 2
				} else if g.blockedAt(x-dirStepX[dir], y) {
					dir = (dir + 1) & 4
				}
			}
			facing = face(dir, facing)
			if frame&1 == 0 {
				frame = 1
			} else {
				frame = 0
			}
		}
		frame = frame&1 + facing<<1
		if t >= ObjGiantScorpion && g.turnParts {
			if head := g.objectHead(n); head >= 0 {
				h := g.object(head)
				const headUp = 8
				if h.Frame()&headUp != 0 {
					h.SetFrame(frame&7 + headUp)
				} else {
					h.SetFrame(frame&7 ^ 4)
				}
				g.MoveObj(head, o.Pos.X-dirStepX[facing<<1], o.Pos.Y-dirStepY[facing<<1], int(g.MapZ))
			}
		}
		o.SetFrame(frame)
	case t == ObjShip && g.turnParts:
		facing := face(dir, (frame&6)>>1)
		frame = 9 + facing<<1
		x, y := o.Pos.X, o.Pos.Y
		d := o.Frame() & 6
		const bowFrames, sternFrames = 8, 15
		for k := g.FindLoc(x+dirStepX[d], y+dirStepY[d], int(g.MapZ)); k >= 0; k = g.NextLoc() {
			p := g.object(k)
			if k >= npcCount && p.Type() == ObjShip && p.Frame() < bowFrames && g.TerrainType[g.tileFrame(k)]&shipEnds[d>>1] != 0 {
				p.SetFrame(frame - 8)
				g.MoveObj(k, x+dirStepX[facing<<1], y+dirStepY[facing<<1], int(g.MapZ))
				break
			}
		}
		for k := g.FindLoc(x-dirStepX[d], y-dirStepY[d], int(g.MapZ)); k >= 0; k = g.NextLoc() {
			p := g.object(k)
			if k >= npcCount && p.Type() == ObjShip && p.Frame() > sternFrames && g.TerrainType[g.tileFrame(k)]&shipEnds[(d>>1)^2] != 0 {
				p.SetFrame(frame + 8)
				g.MoveObj(k, x-dirStepX[facing<<1], y-dirStepY[facing<<1], int(g.MapZ))
				break
			}
		}
		o.SetFrame(frame)
	case t == ObjMouse || t == ObjRabbit || t == ObjSkiff || t == ObjCat:
		o.SetFrame(face(dir, frame))
	case t == ObjCyclops:
		facing := face(dir, frame>>3)
		if frame&4 == 0 {
			frame = 7
		} else {
			frame = 3
		}
		o.SetFrame(frame + facing<<3)
		if !g.isOutOfArea(n) {
			g.cyclopsShake = g.rand(2, 4)
		}
	case t == ObjDragon && frame < 8:
		for k := range 4 {
			if part := g.dragonPart(o.Pos.X, o.Pos.Y, int(g.MapZ), k<<1, frame); part >= 0 {
				g.object(part).SetFrame(dragonTurnFrames[k+(dir&6)<<1])
			}
		}
		o.SetFrame(dir & 6)
	}
	g.turnParts = false
}

// subMove takes movement points from a character.
// C: seg_1E0F:0F54 (SubMov)
func (g *Game) subMove(n, points int) {
	if g.object(n).Type() == ObjMountedHorse {
		points >>= 1
	}
	const attacksParty = 32
	if g.SpellFx[spellHaste] != 0 && g.isPlayerControlled(n) {
		points >>= 1
	} else if g.SpellFx[spellSlow] != 0 && g.NPCs[n].Status&attacksParty != 0 {
		points <<= 1
	}
	points = max(points, 1)
	g.NPCs[n].MovePoints -= int8(points)
}

// draggedUnderTurn lets a character held by a corpser try to break free; it returns true
// when it is still held.
// C: seg_1E0F:0FA9 (C_1E0F_0FA9)
func (g *Game) draggedUnderTurn(n int) bool {
	if !g.draggedUnder(n) {
		return false
	}
	npc := &g.NPCs[n]
	if npc.Mode == AICommand {
		g.printf(argghMsg, g.objectString(n))
	}
	if g.rand(1, maxStat) < int(npc.Strength) {
		g.sfx(sound.EffectHiss, 0)
		g.printf(g.objectString(n))
		g.printf(regurgitated)
		npc.Flags &^= flagDraggedUnder
		g.viewChanged = true
		return false
	}
	g.loseHitPoints(n, g.rand(1, 15))
	npc.MovePoints = 0
	return true
}

// alreadyNear says whether (x, y) is next to one of the first count followers placed (and
// false when it is on one of them).
// C: seg_1E0F:1056 (C_1E0F_1056)
func (g *Game) alreadyNear(count, x, y int) bool {
	near := false
	isMouse := g.object(int(g.followers[count])).Type() == ObjMouse
	check := func(deltaX, deltaY int, t ObjectType) (bool, bool) {
		if deltaX == 0 && deltaY == 0 && t != ObjMouse && !isMouse {
			return false, true
		}
		return deltaX > -2 && deltaX < 2 && deltaY > -2 && deltaY < 2, false
	}
	for k := range count {
		f := g.object(int(g.followers[k]))
		t := f.Type()
		deltaX, deltaY := f.Pos.X-x, f.Pos.Y-y
		n, on := check(deltaX, deltaY, t)
		if on {
			return false
		}
		near = near || n
		if t != ObjMountedHorse {
			continue
		}
		deltaX -= dirStepX[f.Frame()&6]
		deltaY -= dirStepY[f.Frame()&6]
		if n, on = check(deltaX, deltaY, t); on {
			return false
		}
		near = near || n
	}
	return near
}

// the formation of the party behind the leader: sideways and back of each place
// (D_17B8, D_17C3)
var (
	followSide = [11]int{0, -1, 1, 0, -2, 2, -1, 1, -3, 3, 0}
	followBack = [11]int{0, 1, 1, 2, 2, 2, 3, 3, 3, 3, 1}
)

// MoveFollowers moves the party members that follow the leader n into formation behind
// it. keepFormation makes them take their place exactly (after a pass).
// C: seg_1E0F:1193 (MoveFollowers)
func (g *Game) MoveFollowers(n int, keepFormation bool) {
	if g.inVehicle() {
		return
	}
	g.followers[0] = byte(n)
	leader := g.object(n)
	x, y := leader.Pos.X, leader.Pos.Y
	t := leader.Type()
	var facing int
	if t >= ObjFighter && t <= ObjAvatar || t == ObjGargoyle {
		facing = (leader.Frame() >> 1) & 6
	} else {
		facing = g.NPCs[n].Direction() & 6
	}
	// where each member stood before this move, for the Follow formation's line; the
	// leader stood a step back
	var stood [partyMax]image.Point
	for k := range int(g.PartySize) {
		p := g.object(int(g.Party[k])).Pos
		stood[k] = image.Pt(p.X, p.Y)
	}
	// the port's formations other than the original's take their places exactly
	keepFormation = keepFormation || g.formation != formationDefault
	for pass := range 2 {
		place := 1
		front := image.Pt(x-dirStepX[facing], y-dirStepY[facing])
		for k := range int(g.PartySize) {
			member := int(g.Party[k])
			g.followers[place] = byte(member)
			tx, ty := g.formationPlace(place, x, y, facing, front)
			o := g.object(member)
			if member == n || o.Pos.Z != int(g.MapZ) || g.NPCs[member].Mode != AIFollow {
				continue
			}
			if g.draggedUnderTurn(member) || g.incapacitated(member) || g.draggedUnder(member) {
				continue
			}
			saved := g.NPCs[member].MovePoints
			fx, fy := o.Pos.X, o.Pos.Y
			behind := (tx-fx)*dirStepX[facing]+(ty-fy)*dirStepY[facing] > 0
			if !g.alreadyNear(place, fx, fy) ||
				pass == 1 && (keepFormation && !(tx == fx && ty == fy) || !keepFormation && behind) {
				near := g.alreadyNear(place, fx, fy)
				best, bestDir := 0, -1
				for d := range len(dirStepX) {
					nx, ny := fx+dirStepX[d], fy+dirStepY[d]
					if !g.passable(member, nx, ny) {
						continue
					}
					const keen, wary = 256, 128
					eager := keen
					if g.steppedOnDamage {
						if near {
							continue
						}
						eager = wary
					}
					if g.alreadyNear(place, nx, ny) {
						eager += keen
					} else if near {
						eager = 0
					}
					eager -= abs(nx-tx) + abs(ny-ty)
					if eager > best {
						best, bestDir = eager, d
					}
				}
				if bestDir >= 0 {
					g.MoveObj(member, fx+dirStepX[bestDir], fy+dirStepY[bestDir], int(g.MapZ))
					g.turnParts = true
					g.turnTo(member, bestDir)
				}
			}
			g.NPCs[member].MovePoints = saved
			g.subMove(int(g.Party[k]), 5)
			front = stood[k]
			place++
		}
	}
}

// the cost of sailing against the wind and rowing against the river, by the angle
// (D_180A, D_1812)
var (
	sailCost = [8]int{10, 5, 3, 4, 5, 4, 3, 5}
	rowCost  = [8]int{3, 4, 5, 7, 10, 7, 5, 4}
)

// raftDrift is where a raft goes for each river flow and wind (D_181A).
var raftDrift = [8][8]int{
	{0, 0, 1, 1, 0, 7, 7, 0},
	{1, 1, 1, 2, 2, 1, 0, 0},
	{1, 2, 2, 2, 3, 3, 2, 1},
	{2, 2, 3, 3, 3, 4, 4, 3},
	{4, 3, 3, 4, 4, 4, 5, 5},
	{6, 5, 4, 4, 5, 5, 5, 6},
	{7, 7, 6, 5, 5, 6, 6, 6},
	{7, 0, 0, 7, 6, 6, 7, 7},
}

const drunk = 3

// advance moves the party (its active member) one step in dir.
// C: seg_1E0F:1B0E (C_1E0F_1B0E)
func (g *Game) advance(dir int) {
	g.warnNoQuest = true
	leader := int(g.Party[g.Active])
	o := g.object(leader)
	g.MapX, g.MapY = int16(o.Pos.X), int16(o.Pos.Y)
	mx, my := int(g.MapX), int(g.MapY)
	g.moveBlocked = false
	if g.DrunkCounter > drunk && g.rand(0, 1) != 0 {
		g.printf("Hic!\n")
		dir = g.rand(0, 7)
	}
	if t := o.Type(); t == ObjRaft || t == ObjBalloon2 && !sound.Settings.FreeBalloonMovement {
		flow := dir
		dir = int(g.WindDir) ^ 4
		if g.WindDir < 0 {
			dir = int(g.WindDir)
		}
		if t == ObjRaft {
			if tile := g.GetTileAtXYZ(mx, my, int(g.MapZ)); tile >= tileFlow && tile < tileFlow+8 {
				flow = tile - tileFlow
			}
			if dir < 0 {
				dir = flow
			} else {
				dir = raftDrift[flow][dir]
			}
		}
		if dir < 0 {
			g.printf(noWindMsg)
			g.promptFlag = true
			g.NPCs[leader].MovePoints = 0
			return
		}
	}
	newX := (mx + dirStepX[dir]) & surfaceMask
	newY := (my + dirStepY[dir]) & surfaceMask
	diagonal := dirStepX[dir] != 0 && dirStepY[dir] != 0
	if !g.passable(leader, newX, newY) {
		g.moveBlocked = true
		if g.mousePressed && diagonal {
			// slide along the obstacle toward the pointer
			if abs(g.mouseMapX-mx) > abs(g.mouseMapY-my) {
				if g.passable(leader, newX, my) {
					g.moveBlocked = false
					newY = my
				} else if g.passable(leader, mx, newY) {
					g.moveBlocked = false
					newX = mx
				}
			} else {
				if g.passable(leader, mx, newY) {
					g.moveBlocked = false
					newX = mx
				} else if g.passable(leader, newX, my) {
					g.moveBlocked = false
					newY = my
				}
			}
		}
	} else if diagonal && !g.passable(leader, (mx+dirStepX[dir])&surfaceMask, my) &&
		!g.passable(leader, mx, (my+dirStepY[dir])&surfaceMask) {
		g.moveBlocked = true
	}
	if g.moveBlocked {
		const bumpHz = 311
		g.sfx(sound.EffectNote, bumpHz)
		g.MapX, g.MapY = int16(o.Pos.X), int16(o.Pos.Y)
		g.warnNoQuest = false
		return
	}
	if g.object(int(g.Party[g.Active])).Type() != ObjMouse {
		// swap places with a party member in the way
		for k := range int(g.PartySize) {
			if int(g.Active) == k {
				continue
			}
			m := int(g.Party[k])
			p := g.object(m)
			if !g.NPCs[m].Has(Dead) && p.Type() != ObjMouse && p.Pos.X == newX && p.Pos.Y == newY && p.Pos.Z == int(g.MapZ) {
				if g.NPCs[m].Mode != AIFront || !g.TryMoveTo(m, newX+newX-mx, newY+newY-my) {
					g.MoveObj(m, mx, my, int(g.MapZ))
					g.turnParts = true
					g.turnTo(m, (dir+4)&7)
				}
				break
			}
		}
	}
	leader = int(g.Party[g.Active])
	switch t := g.object(leader).Type(); {
	case t == ObjShip && g.WindDir >= 0:
		g.subMove(leader, sailCost[(dir-int(g.WindDir))&7])
	case t == ObjSkiff:
		flow := dir
		if tile := g.GetTileAtXYZ(mx, my, int(g.MapZ)); tile >= tileFlow && tile < tileFlow+8 {
			flow = tile - tileFlow
		}
		g.subMove(leader, rowCost[(dir-flow)&7])
	case t == ObjRaft || t == ObjBalloon2:
		g.subMove(leader, 5)
	case diagonal:
		g.subTerrainMove(leader)
		g.subMove(leader, 5)
	default:
		g.subTerrainMove(leader)
	}
	g.MapX, g.MapY = int16(newX), int16(newY)
	g.enterArea()
	if g.MoveObj(leader, newX, newY, int(g.MapZ)) >= 0 {
		g.NPCs[leader].SetDirection(dir)
		g.turnParts = true
		g.turnTo(leader, dir)
		g.viewChanged = true
	} else {
		o := g.object(leader)
		g.MapX, g.MapY = int16(o.Pos.X), int16(o.Pos.Y)
		g.enterArea()
		g.printf("Object mismatch\n")
	}
	g.warnNoQuest = false
	g.MoveFollowers(leader, false)
	g.enterGates()
	g.warnNoQuest = false
}

// enterGates takes the party through a moongate, a red gate, or into a dungeon when it
// stepped on one.
// C: seg_1E0F:184D (C_1E0F_184D)
func (g *Game) enterGates() {
	if g.InCombat != 0 || g.inVehicle() {
		return
	}
	soloRefused := false
	for n := g.FindLoc(int(g.MapX), int(g.MapY), int(g.MapZ)); n >= 0; n = g.NextLoc() {
		if int(g.Party[g.Active]) == n {
			continue
		}
		t := g.object(n).Type()
		if (t == ObjMoongate || t == ObjRedGate) && g.locPart == 0 || t == ObjMineShaft || t == ObjHole {
			soloRefused = !g.travelThrough(n)
			break
		}
	}
	if soloRefused {
		g.printf(partyModeOnly)
		g.promptFlag = true
	}
}

// subTerrainMove takes the movement points the ground costs.
// C: seg_1E0F:31F7 (SubTerrainMov)
func (g *Game) subTerrainMove(n int) {
	points := 5
	p := g.object(n).Pos
	if tile := g.GetTileAtXYZ(p.X, p.Y, p.Z); !g.isImpassable(tile) {
		points += int(g.TerrainType[tile]) >> terrainCostShift
	}
	for obj := g.FindLoc(p.X, p.Y, p.Z); obj >= 0; obj = g.NextLoc() {
		if tile := g.tileFrame(obj); !g.isImpassable(tile) {
			points += int(g.TerrainType[tile]) >> terrainCostShift
		}
	}
	g.subMove(n, points)
}

// openDoorUnder opens (step 1) or closes (step 0) the door a door-opening monster stands
// in.
// C: seg_1E0F:2001 (C_1E0F_2001)
func (g *Game) openDoorUnder(n, step int) {
	cls := monsterClass(g.object(n).Type())
	if cls < 0 || !classHas(cls, opensDoors) || g.isPlayerControlled(n) {
		return
	}
	p := g.object(n).Pos
	for obj := g.FindLoc(p.X, p.Y, p.Z); obj >= 0; obj = g.NextLoc() {
		door := g.object(obj)
		switch t := door.Type(); {
		case t >= ObjOakenDoor && t <= ObjSteelDoor:
			door.SetFrame(step<<2 + door.Frame()&3)
			g.computeView()
			return
		case t == ObjPassthrough || t == ObjPassthrough2:
			g.TextOn = false
			g.usePassthrough(obj)
			g.TextOn = true
		}
	}
}

// TryStraightMove moves a character one step in dir when it can.
// C: seg_1E0F:32D6 (TryStraightMove)
func (g *Game) TryStraightMove(n, dir int, avoidDamage bool) bool {
	dir &= 7
	g.subTerrainMove(n)
	p := g.object(n).Pos
	newX, newY := p.X+dirStepX[dir], p.Y+dirStepY[dir]
	const combatReach = 8
	if !g.isPlayerControlled(n) || g.InCombat == 0 ||
		abs(int(g.MapX)-newX) <= combatReach && abs(int(g.MapY)-newY) <= combatReach {
		g.avoidDamage = avoidDamage
		if g.passable(n, newX, newY) {
			g.avoidDamage = false
			g.openDoorUnder(n, 1)
			g.MoveObj(n, newX, newY, int(g.MapZ))
			g.NPCs[n].SetDirection(dir)
			g.turnParts = true
			g.turnTo(n, dir)
			g.openDoorUnder(n, 0)
			return true
		}
		g.avoidDamage = false
	}
	return false
}

// tryDiagonalMove moves a character diagonally between hDir and vDir when it can.
// C: seg_1E0F:3470 (__TryDiagMove)
func (g *Game) tryDiagonalMove(n, hDir, vDir int) bool {
	if g.object(n).Type() == ObjSilverSerpent {
		return false
	}
	dir := (vDir + hDir) >> 1
	if vDir == 0 && hDir == 6 {
		dir = 7
	}
	g.subTerrainMove(n)
	p := g.object(n).Pos
	newX, newY := p.X+dirStepX[dir], p.Y+dirStepY[dir]
	g.allowPartySwap = true
	g.avoidDamage = true
	sidestep := g.passable(n, newX, p.Y)
	if !sidestep {
		sidestep = g.passable(n, p.X, newY)
	}
	g.allowPartySwap = false
	if sidestep && g.passable(n, newX, newY) {
		g.avoidDamage = false
		g.openDoorUnder(n, 1)
		g.MoveObj(n, newX, newY, int(g.MapZ))
		g.NPCs[n].SetDirection(dir)
		g.turnParts = true
		g.turnTo(n, dir)
		g.openDoorUnder(n, 0)
		return true
	}
	g.avoidDamage = false
	return false
}

// TryMoveTo moves a character one step toward (x, y); it returns false when it could
// not move.
// C: seg_1E0F:35A7 (TryMoveTo)
func (g *Game) TryMoveTo(n, x, y int) bool {
	avoid := !g.tryHarder
	moved := true
	p := g.object(n).Pos
	deltaX, deltaY := x-p.X, y-p.Y
	if deltaX == 0 && deltaY == 0 || g.NPCs[n].Mode == AIImmobile {
		g.subMove(n, 5)
		return false
	}
	hDir := 2
	switch {
	case deltaX == 0:
		if g.rand(0, 1) == 0 {
			hDir = 6
		}
	case deltaX < 0:
		hDir = 6
	}
	vDir := 4
	switch {
	case deltaY == 0:
		if g.rand(0, 1) != 0 {
			vDir = 0
		}
	case deltaY < 0:
		vDir = 0
	}
	g.allowPartySwap = false
	var vertical bool
	const near = 4
	if abs(deltaX) < near && abs(deltaY) < near {
		vertical = abs(deltaX) < abs(deltaY) || abs(deltaX) == abs(deltaY) && g.rand(0, 1) != 0
	} else {
		vertical = g.rand(1, abs(deltaX)+abs(deltaY)) > abs(deltaX)
	}
	first, second := hDir, vDir
	if vertical {
		first, second = vDir, hDir
	}
	if !g.TryStraightMove(n, first, avoid) && !g.tryDiagonalMove(n, hDir, vDir) &&
		!g.TryStraightMove(n, second, avoid) && (g.rand(0, 1) != 0 || !g.TryStraightMove(n, second^4, true)) {
		g.subTerrainMove(n)
		moved = false
	}
	g.allowPartySwap = true
	return moved
}

// runTurns gives turns to the characters until it is the turn of the one the player
// commands, and makes that one active.
// C: seg_1E0F:4E0A (C_1E0F_4E0A)
func (g *Game) runTurns() {
	g.checkActiveMember()
	g.computeGravity()
	g.schedulePaths()
	n := int(g.Party[g.Active])
	if g.NPCs[n].Mode > AIFollow && g.NPCs[n].Mode < AIGuard2 {
		g.NPCs[n].Leader = byte(n)
	}
	mask := surfaceMask
	if g.MapZ != 0 {
		mask = dungeonMask
	}
	for {
		n = noObject
		bestPoints, bestDex := 0, 1
		for bestPoints <= 0 {
			areaX := (int(g.MapX) - areaMargin) & areaAlign
			areaY := (int(g.MapY) - areaMargin) & areaAlign
			for i := 1; i < int(g.PartySize); i++ {
				if npc := &g.NPCs[g.Party[i]]; npc.Mode == AIFollow && npc.MovePoints <= 0 {
					g.NPCs[g.Party[0]].MovePoints = 0
				}
			}
			for i := range npcCount {
				o, npc := g.object(i), &g.NPCs[i]
				if o.Shape == 0 || npc.Mode == AIMotionless || npc.Has(Paralyzed) ||
					npc.Has(Asleep) && npc.Mode != AISchedule || npc.Has(Dead) || npc.MovePoints <= 0 ||
					o.CoordUse() != OnMap || o.Pos.Z != int(g.MapZ) {
					continue
				}
				if (o.Pos.X-areaX)&mask > areaSize-1 || (o.Pos.Y-areaY)&mask > areaSize-1 {
					if npc.Mode >= AIOnPath && npc.Mode <= AIPathLost {
						g.sendHome(i)
					}
					if npc.Mode >= AIOnPath && npc.Mode < AIPathLost {
						g.PathObject[npc.Leader] = 0
						npc.Mode = AIFindPath
					}
					npc.MovePoints = 0
				} else if npc.Mode != AIFollow {
					if npc.Mode == AISchedule {
						g.atDestination(i)
					}
					dex := int(npc.Dexterity)
					points := int(npc.MovePoints)
					better := points*bestDex - dex*bestPoints
					if points >= dex || better > 0 || better == 0 && dex > bestDex {
						n, bestPoints, bestDex = i, points, dex
					}
					if points >= dex {
						break
					}
				}
			}
			if bestPoints <= 0 {
				for i := range npcCount {
					npc := &g.NPCs[i]
					if npc.MovePoints < 0 {
						npc.MovePoints += int8(npc.Dexterity)
					} else {
						npc.MovePoints = int8(npc.Dexterity)
					}
				}
				g.passTime(1)
			}
		}
		if !g.draggedUnderTurn(n) {
			if mode := g.NPCs[n].Mode; mode != AICommand && mode != AIFollow {
				g.npcTurn(n)
				g.checkActiveMember()
				if mode := g.NPCs[n].Mode; mode > AIFollow && mode < AIGuard2 {
					g.NPCs[n].Leader = byte(n)
				}
			}
			g.updateViewNow()
		}
		if g.NPCs[n].Mode == AICommand && !g.draggedUnder(n) {
			break
		}
	}
	if g.inVehicle() {
		n = 0
	}
	if int(g.Party[g.Active]) != n {
		for i := range int(g.PartySize) {
			if int(g.Party[i]) == n {
				g.SetActive(i)
				g.promptFlag = true
				break
			}
		}
	}
}

// initTurns readies the characters for the first turn.
// C: seg_1E0F:512A (C_1E0F_512A)
func (g *Game) startTurns() {
	for i := range npcCount {
		if g.NPCs[i].Dexterity == 0 {
			g.NPCs[i].Dexterity = 1
		}
	}
	if g.NPCs[0].Mode >= AISchedule {
		g.NPCs[0].Mode = AIMotionless
	}
	if g.SoloMember < 0 {
		g.soloLeader = int(g.Active)
	}
	g.computeGravity()
}

// followSchedules sends the NPCs whose schedule changes this hour on their way.
// C: seg_1E0F:5165 (C_1E0F_5165)
func (g *Game) followSchedules() {
	const hourMask, dayShift = 31, 5
	for npcID := 2; npcID < firstMonster; npcID++ {
		npc := &g.NPCs[npcID]
		if npc.Mode == AIVigilante {
			npc.Mode = AIFindPath
		}
		if g.object(npcID).Shape == 0 {
			continue
		}
		if npc.Mode < AISchedule && npc.Mode != AIRetreat {
			continue
		}
		if g.isPlayerControlled(npcID) {
			continue
		}
		first, last := int(g.schedulePointer[npcID]), int(g.schedulePointer[npcID+1])-1
		for entry := last; entry >= first; entry-- {
			e := g.schedule[entry]
			const everyDay = 0
			if int(e.Time)&hourMask == int(g.Hour) &&
				(int(e.Time)>>dayShift == (int(g.Day)-1)%daysPerWeek+1 || int(e.Time)>>dayShift == everyDay) {
				npc.ScheduleIndex = byte(entry - first)
				npc.Mode = AIFindPath
				if g.object(npcID).Type() == ObjMusician2 && AIMode(e.Action) != AIPlay {
					g.object(npcID).Shape = npc.OrigShape
					g.turnTo(npcID, 4)
				}
				break
			}
		}
	}
}
