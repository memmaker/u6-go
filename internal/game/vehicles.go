package game

// Vehicles: boarding and leaving ships, skiffs, rafts and balloons, and riding horses
// (seg_1E0F, seg_27A1).

// board takes the party aboard vehicle n: the party becomes the vehicle's cargo and
// character 0 becomes the vehicle.
// C: seg_1E0F:14BE (Board)
func (g *Game) board(n int) {
	carried := noObject
	if g.object(n).Type() == ObjShip {
		// a ship fills two tiles: find the half the party boards by
		dir := -1
		switch g.TerrainType[g.tileFrame(n)] & terrainSides {
		case terrainNorth:
			dir = dirNorth
		case terrainEast:
			dir = dirEast
		case terrainSouth:
			dir = dirSouth
		case terrainWest:
			dir = dirWest
		}
		if dir >= 0 {
			p := g.object(n).Pos
			for n = g.FindLoc(p.X+dirStepX[dir], p.Y+dirStepY[dir], int(g.MapZ)); n >= 0; n = g.NextLoc() {
				if g.object(n).Type() == ObjShip {
					break
				}
			}
		}
	}
	if n < 0 {
		return
	}
	g.breakOff()
	if g.inVehicle() {
		carried = g.unboard(true)
		if t := g.object(carried).Type(); t != ObjSkiff && t != ObjRaft {
			carried = noObject
		}
	}
	o := g.object(n)
	g.MapX, g.MapY = int16(o.Pos.X), int16(o.Pos.Y)
	g.partyEnter(2)
	const vehicle = 0
	g.cloneObject(int(g.Party[0]), vehicle)
	v, npc := g.object(vehicle), &g.NPCs[vehicle]
	v.Status &^= StatusLocal | StatusInvisible
	npc.MovePoints = int8(npc.Dexterity)
	g.MoveObj(vehicle, int(g.MapX), int(g.MapY), int(g.MapZ))
	v.Shape = o.Shape
	npc.HitPoints = byte(o.Quantity())
	g.setAlignment(vehicle, alignGood)
	npc.Status |= PlayerControl
	npc.Mode = AICommand
	o.Shape = shapeOf(ObjGold, 0)
	g.DeleteObj(n)
	for i := range int(g.PartySize) {
		g.InsertObj(int(g.Party[i]), vehicle, InInventory)
	}
	if carried >= 0 {
		g.InsertObj(carried, vehicle, InInventory)
	}
	g.Active = g.PartySize
	g.Party[g.Active] = vehicle
	g.Names[g.Active] = g.Names[0]
	g.ShowObjects()
	g.viewChanged = true
	g.playAreaSong()
}

// unboard leaves the vehicle, on land or, at sea, into a skiff or raft aboard; it
// returns the vehicle left on the map.
// C: seg_1E0F:1745 (Unboard)
func (g *Game) unboard(onLand bool) int {
	boat := noObject
	v := int(g.Party[g.PartySize])
	left := g.AddMapObj(uint16(g.object(v).Type()), StatusOwned, uint16(g.NPCs[v].HitPoints), int(g.MapX), int(g.MapY), int(g.MapZ))
	if !onLand {
		for boat = g.FindInv(v); boat >= 0; boat = g.NextInv() {
			if t := g.object(boat).Type(); t == ObjSkiff || t == ObjRaft {
				g.MoveObj(boat, int(g.MapX), int(g.MapY), int(g.MapZ))
				break
			}
		}
	}
	if left >= 0 {
		// fix: the original clears the flag before testing that there is an object
		g.object(left).Status &^= StatusLocal
		g.object(left).Shape = g.object(v).Shape
	}
	g.object(v).Shape = shapeOf(ObjGold, 0)
	g.NPCs[v].Mode = AIMotionless
	const nowhere = 14
	g.MoveObj(v, int(g.MapX), int(g.MapY), nowhere)
	g.Active = 0
	g.partyExit(int(g.MapX), int(g.MapY), int(g.MapZ), 2)
	if boat >= 0 {
		g.board(boat)
	}
	g.playAreaSong()
	return left
}

// landNear says whether the party can step off vehicle n at (x, y): a ship only from its
// sides.
// C: seg_27A1:51C4 (C_27A1_51C4)
func (g *Game) landNear(x, y, n int) bool {
	lead := int(g.Party[0])
	if g.object(n).Type() == ObjShip {
		d := g.object(n).Frame() & 2
		return g.passable(lead, dirStepX[(d+2)&7]+x, dirStepY[(d+2)&7]+y) ||
			g.passable(lead, dirStepX[(d+6)&7]+x, dirStepY[(d+6)&7]+y)
	}
	for d := range dirStepX {
		if g.passable(lead, dirStepX[d]+x, dirStepY[d]+y) {
			return true
		}
	}
	return false
}

// useVehicle boards or leaves a ship, skiff, raft or balloon.
// C: seg_27A1:5289 (C_27A1_5289)
func (g *Game) useVehicle(n int) {
	n = g.combatHead(n)
	t := g.object(n).Type()
	actor := int(g.Party[g.Active])
	p := g.object(n).Pos
	if g.SoloMember != -1 {
		g.printf("Not in solo mode!\n")
		return
	}
	switch {
	case g.inVehicle() && actor == n:
		switch {
		case g.landNear(int(g.MapX), int(g.MapY), n):
			n = g.unboard(true)
			if t == ObjBalloon2 {
				g.object(n).Shape = shapeOf(ObjBalloon, 0)
				g.refreshInventory(g.container)
			}
		case t == ObjShip && g.FindInvType(n, ObjSkiff, 0) != noObject:
			g.unboard(false)
		default:
			g.printf("\nOnly next to land.\n")
		}
	case (t == ObjShip || t == ObjSkiff) && g.object(n).Quality() != 0 &&
		g.partyMemberHolding(ObjShipDeep, g.object(n).Quality()) == noObject:
		g.printf("\nA deed is required.\n")
	case !closeEnough(1, int(g.MapX), int(g.MapY), p.X, p.Y):
		g.printf("\n%s!\n", outOfRangeMsg)
	default:
		if t == ObjShip && g.object(actor).Type() == ObjSkiff {
			g.InsertObj(actor, n, InInventory)
		}
		g.board(n)
	}
}

// mount puts rider n on a horse at (x, y) facing frame; with existing the horse's rear
// half is already there.
// C: seg_27A1:5431 (C_27A1_5431)
func (g *Game) mount(n, x, y, frame int, existing bool) {
	g.MoveObj(n, x, y, int(g.MapZ))
	g.object(n).Shape = shapeOf(ObjMountedHorse, frame)
	g.MapX, g.MapY = int16(x), int16(y)
	d := frame & 6
	x -= dirStepX[d]
	y -= dirStepY[d]
	if existing {
		rear := g.FindLoc(x, y, int(g.MapZ))
		g.object(rear).Shape = shapeOf(ObjMountedHorse, g.object(rear).Frame())
	} else {
		const rearFrames = 8
		g.AddMapObj(shapeOf(ObjMountedHorse, frame+rearFrames), 0, 0, x, y, int(g.MapZ))
	}
	g.enterArea()
	g.OtherAnimations()
}
