package game

import (
	"io/fs"

	"github.com/memmaker/u6-go/internal/sound"
)

// The look, get, drop and move commands (seg_27A1, its first half).

// Common messages (D_0DDC).
const (
	failedMsg        = "Failed"
	noEffectMsg      = "No effect"
	outOfRangeMsg    = "Out of range"
	blockedMsg       = "Blocked"
	nothingMsg       = "nothing"
	notPossibleMsg   = "Not possible"
	doneMsg          = "Done"
	notUsableMsg     = "Not usable"
	notAroundMsg     = "\nThis person is not around!\n"
	searchStartMsg   = "\nSearching here, you find "
	searchNoneMsg    = "\nSearching here, you find nothing.\n"
	tooHeavyMsg      = "\nThe total is too heavy.\n"
	criticalMsg      = "critical!\n"
	heavilyMsg       = "heavily"
	lightlyMsg       = "lightly"
	barelyMsg        = "barely"
	woundedMsg       = " wounded.\n"
	selectOnMap      = 1   // SelectMode: the aim cursor on the map
	aimRange         = 7   // SelectRange of the targeting commands
	lookViewEdge     = 168 // the pointer beyond this looks at nothing
	carryPerStrength = 20  // a character carries 2 stones per point of strength
	nearCrime        = 6   // citizens this close see a crime
	guardsHear       = 32  // guards this close come to arrest
	textAttrLook     = clipPaging
)

// dirNames are the eight directions (DirNames).
var dirNames = [8]string{"North", "Northeast", "East", "Southeast", "South", "Southwest", "West", "Northwest"}

// closeEnough says whether (x1, y1) is within d tiles of (x2, y2) (CLOSE_ENOUGH).
func closeEnough(d, x1, y1, x2, y2 int) bool { return abs(x1-x2) <= d && abs(y1-y2) <= d }

// qualQuan packs a quality and a quantity into an Amount (QualQuan).
func qualQuan(quality, quantity int) int { return quality<<byteBits | quantity }

// tileOf is the tile object n shows (TILE_FRAME).
func (g *Game) tileOf(n int) int { return g.tileFrame(n) }

// containerTypes can hold objects (D_1C00).
var containerTypes = []ObjectType{ObjOpenChest, ObjBackpack, ObjSpellbook, ObjOpenBarrel, ObjBag,
	ObjBasket, ObjOpenCrate, ObjDeadBody, ObjVortexCube}

// isContainer says whether objects of type t hold others.
// C: seg_27A1:0082 (C_27A1_0082)
func isContainer(t ObjectType) bool { return hasType(containerTypes, t) }

func hasType(list []ObjectType, t ObjectType) bool {
	for _, l := range list {
		if l == t {
			return true
		}
	}
	return false
}

// isOpenable is a chest, barrel or crate, which only takes objects when open.
func isOpenable(t ObjectType) bool {
	return t == ObjOpenChest || t == ObjOpenBarrel || t == ObjOpenCrate
}

// canPutInto says whether object n may go into container c.
// C: seg_27A1:00A9 (C_27A1_00A9)
func (g *Game) canPutInto(c, n int) bool {
	if c == noObject {
		return false
	}
	t := g.object(c).Type()
	if !isContainer(t) || n < npcCount || t == ObjSpellbook || t == ObjDeadBody {
		return false
	}
	return (!isOpenable(t) || g.object(c).Frame() == 0) && !isOpenable(g.object(n).Type())
}

// breakableTypes break when thrown (D_1C12).
var breakableTypes = []ObjectType{ObjFlaskOfOil, ObjSilverSnakeVenom, ObjCrystalBall, ObjMirror, ObjWine,
	ObjMead, ObjAle, ObjWineglass, ObjPlate, ObjMug, ObjHoneyJar, ObjJarOfHoney, ObjBluePotion,
	ObjWaterVase, ObjDragonEgg}

// okToGet says whether n and what it holds may be taken without stealing.
// C: seg_27A1:0153 (IsOkToGet)
func (g *Game) okToGet(n int) bool {
	if !g.object(n).Has(StatusOwned) {
		return false
	}
	for _, i := range g.heldBy(n) {
		if !g.object(i).Has(StatusOwned) {
			return false
		}
	}
	return true
}

// setOkToGet lets n and what it holds be taken.
// C: seg_27A1:019E (SetOkToGet)
func (g *Game) setOkToGet(n int) {
	g.object(n).Status |= StatusOwned
	for _, i := range g.heldBy(n) {
		g.object(i).Status |= StatusOwned
	}
}

// isLight says whether objects of type t weigh a tenth each.
// C: seg_155D:063A (C_155D_063A)
func isLight(t ObjectType) bool { return hasType(lightTypes, t) }

// printArticle prints "a ", "an " or "the " before the name of tile.
// C: seg_27A1:061E (C_27A1_061E)
func (g *Game) printArticle(tile int) {
	const a, an = 64, 128
	switch g.TileFlag2[tile] & tileArticle {
	case a:
		g.printf("a ")
	case an:
		g.printf("an ")
	case tileArticle:
		g.printf("the ")
	}
}

// books are read when looked at near; signs from anywhere (D_1CDA, D_1CE4).
var (
	bookTypes = []ObjectType{ObjBook, ObjBookOfCircles, ObjNote, ObjBalloonPlans, ObjCodex}
	signTypes = []ObjectType{ObjSign, ObjGargoyleSign, ObjPicture, ObjCross, ObjTombstone}
)

// canRead says whether the party can read object n.
// C: seg_27A1:06D7 (C_27A1_06D7)
// C: seg_27A1:066D (C_27A1_066D), inlined
// C: seg_27A1:06A2 (C_27A1_06A2), inlined
func (g *Game) canRead(n int) bool {
	t := g.object(n).Type()
	if hasType(bookTypes, t) {
		p := g.object(int(g.Party[g.Active])).Pos
		o := g.object(n).Pos
		return g.object(g.Selection.obj).CoordUse() != OnMap || closeEnough(1, o.X, o.Y, p.X, p.Y)
	}
	return hasType(signTypes, t)
}

// readBook prints the text of BOOK.DAT that object n's quality points to.
// C: seg_27A1:078F (C_27A1_078F)
func (g *Game) readBook(n int) {
	g.playSong(0)
	g.printf("\n\n")
	b, err := fs.ReadFile(g.files, "BOOK.DAT")
	i := (g.object(n).Quality() - 1) * 2
	if err == nil && i >= 0 && i+1 < len(b) {
		start := int(b[i]) | int(b[i+1])<<byteBits
		end := start
		for end < len(b) && b[end] != 0 {
			end++
		}
		g.current.attr |= textAttrLook
		g.printf("%s", string(b[start:end]))
		g.pagingDisabled = true
	}
	g.playAreaSong()
}

// printCount prints how many of object n there are, or its article; it returns the count.
// C: seg_27A1:0841 (C_27A1_0841)
func (g *Game) printCount(n int) int {
	o := g.object(n)
	switch quantityType(o.Type()) {
	case countGold:
		g.printf("%d ", int(o.Amount))
		return int(o.Amount)
	case countQuantity:
		if !o.Has(StatusLit) {
			g.printf("%d ", o.Quantity())
			return o.Quantity()
		}
	}
	g.printArticle(g.tileOf(n))
	return 1
}

// isClosed says whether n is a closed chest, barrel or crate, or the spellbook, which a
// search does not empty.
// C: seg_27A1:08D3 (C_27A1_08D3)
func (g *Game) isClosed(n int) bool {
	t := g.object(n).Type()
	return isOpenable(t) && g.object(n).Frame() != 0 || t == ObjSpellbook
}

// objectBelow returns the first object under n that is not ignored, or a character.
// C: seg_27A1:0919 (C_27A1_0919)
func (g *Game) objectBelow(n int) int {
	p := g.object(n).Pos
	i := g.FindLoc(p.X, p.Y, int(g.MapZ))
	for ; i >= 0; i = g.NextLoc() {
		if i != n && !g.isIgnored(g.tileOf(i)) && i >= npcCount {
			break
		}
	}
	return i
}

// search empties object n onto the map when the party stands next to it, listing what it
// finds.
// C: seg_27A1:09A1 (C_27A1_09A1)
func (g *Game) search(n, x, y int) {
	actor := int(g.Party[g.Active])
	ap := g.object(actor).Pos
	if !closeEnough(1, x, y, ap.X, ap.Y) || n < npcCount || g.isDamaging(g.tileOf(n)) ||
		g.object(n).Type() == ObjSpellbook {
		return
	}
	g.printf(searchStartMsg)
	found := false
	if o := g.object(n); o.Type() == ObjSecretDoor {
		if o.Frame()&1 == 0 {
			g.printf("a secret door")
			o.SetFrame(o.Frame() ^ 1)
			g.computeView()
			g.OtherAnimations()
			found = true
		}
	} else {
		if g.isIgnored(g.tileOf(n)) {
			n = g.objectBelow(n)
		}
		if !g.isClosed(n) {
			for i := g.FindInv(n); i >= 0; {
				if t := g.object(i).Type(); t == ObjCharge || t == ObjEffect {
					i = g.NextInv()
					continue
				}
				np := g.object(n).Pos
				g.MoveObj(i, np.X, np.Y, int(g.MapZ))
				g.MoveObj(actor, ap.X, ap.Y, int(g.MapZ))
				g.computeView()
				g.OtherAnimations()
				next := g.FindInv(n)
				for next >= 0 && (g.object(next).Type() == ObjCharge || g.object(next).Type() == ObjEffect) {
					next = g.NextInv()
				}
				if found && next >= 0 {
					g.printf(", ")
				} else if found {
					g.printf(" and ")
				}
				g.printCount(i)
				g.printf("%s", g.objectString(i))
				found = true
				i = next
			}
		}
	}
	if !found {
		g.printf("nothing.\n")
		return
	}
	g.printf(".\n")
}

// armourTypes and their armour points (D_1A87, D_1ABB).
var armourPoints = map[ObjectType]int{
	ObjLeatherHelm: 1, ObjChainCoif: 2, ObjIronHelm: 3, ObjSpikedHelm: 3, ObjWingedHelm: 2,
	ObjBrassHelm: 2, ObjGargoyleHelm: 3, ObjMagicHelm: 5, ObjWoodenShield: 2, ObjCurvedHeater: 3,
	ObjWingedHelm2: 3, ObjKiteShield: 3, ObjSpikedShield: 2, ObjBlackShield: 2, ObjDoorShield: 4,
	ObjMagicShield: 5, ObjClothArmor: 1, ObjLeatherArmor: 2, ObjRingMail: 3, ObjScaleMail: 4,
	ObjChainMail: 5, ObjPlateMail: 7, ObjMagicArmor: 10, ObjMainGauche: 1, ObjSpikedCollar: 2,
	ObjProtectionRing: 5,
}

// armorPoints is the damage object n absorbs.
// C: seg_2337:0983 (GetArmorPoints)
func (g *Game) armorPoints(n int) int { return armourPoints[g.object(n).Type()] }

// weaponDamage is the damage of weapons (D_1B0B, D_1B55).
var weaponDamage = map[ObjectType]int{
	ObjSpikedHelm: 4, ObjSling: 6, ObjClub: 8, ObjMainGauche: 8, ObjSpear: 10, ObjThrowingAxe: 10,
	ObjDagger: 6, ObjMace: 15, ObjMorningStar: 15, ObjBow: 10, ObjCrossbow: 12, ObjSword: 15,
	ObjTwoHandedHammer: 20, ObjTwoHandedAxe: 20, ObjTwoHandedSword: 20, ObjHalberd: 30,
	ObjBoomerang: 8, ObjTripleCrossbow: 12, ObjGlassSword: 255, ObjSpikedShield: 4, ObjCannon: 90,
	ObjFlaskOfOil: 4, ObjMagicBow: 20, ObjRollingPin: 2, ObjScythe: 6, ObjPitchfork: 4, ObjPick: 4,
	ObjShovel: 4, ObjHoe: 4, ObjCleaver: 4, ObjKnife: 4, ObjDecorativeSword: 4, ObjShip: 30,
	ObjZuYlem: 0, ObjStaff: 4, ObjLightningWand: 30, ObjFireWand: 20,
}

// classDamage is the damage of each monster class (D_3522_0182).
var classDamage = [...]byte{
	1, 10, 8, 4, 6, 3, 1, 2, 15, 3, 20, 20, 2, 3, 30, 10,
	4, 4, 6, 6, 15, 1, 12, 1, 8, 12, 3, 3, 15, 1, 4, 255,
	4, 4, 4, 15, 20, 2, 2, 6, 20, 2, 8, 30, 2, 30, 60, 6,
	4, 2, 8, 20, 4, 1, 6, 4, 1, 4, 1, 20, 10, 10, 2, 4,
}

// damagePoints is the damage creature or weapon n does, -1 for none.
// C: seg_2337:10E4 (GetDamagePoints)
func (g *Game) damagePoints(n int) int {
	t := g.object(n).Type()
	if n < npcCount {
		if cls := monsterClass(t); cls >= 0 {
			return int(classDamage[cls])
		}
	}
	if d, ok := weaponDamage[t]; ok {
		return d
	}
	return -1
}

// combatHead returns the head of a creature that fills several tiles.
// C: seg_2337:205C (COMBAT_getHead)
func (g *Game) combatHead(n int) int {
	if n < 0 {
		return noObject
	}
	o := g.object(n)
	switch t := o.Type(); {
	case t >= ObjGiantScorpion || t == ObjDragon || t == ObjShip || t == ObjSilverSerpent:
		if n >= npcCount {
			n = g.objectHead(n)
		}
	case t == ObjHydra2:
		const dirShift = 2
		d := o.Frame() >> dirShift
		for i := g.FindLoc(o.Pos.X-dirStepX[d], o.Pos.Y-dirStepY[d], int(g.MapZ)); i >= 0; i = g.NextLoc() {
			if i < npcCount && g.object(i).Type() == ObjHydra {
				return i
			}
		}
	}
	return n
}

// displaySeverity says how badly n is wounded.
// C: seg_2337:198F (COMBAT_displaySeverity)
func (g *Game) displaySeverity(n int) {
	npc := &g.NPCs[n]
	if g.object(n).Type() == ObjBalloon2 || npc.HitPoints == 0 {
		return
	}
	const quarters = 4
	q := int(npc.HitPoints) * quarters / g.maxHitPoints(n)
	if q < quarters {
		g.printf("`%s ", g.objectString(n))
		g.promptFlag = true
	}
	switch q {
	case 0:
		g.printf(criticalMsg)
		const fleeIntelligence = 5
		switch npc.Mode {
		case AIBerserk, AIImmobile, AICommand, AIFront, AIFollow, AIMotionless:
		default:
			if g.intelligenceOf(n) >= fleeIntelligence {
				npc.Mode = AIRetreat
			}
		}
	case 1:
		g.printf(heavilyMsg)
	case 2:
		g.printf(lightlyMsg)
	case 3:
		g.printf(barelyMsg)
	}
	if q != 0 && q < quarters {
		g.printf(woundedMsg)
	}
}

// subKarma lowers the karma, not below 0 (SubKarma).
// C: seg_0A33:0008 (SubKarma)
func (g *Game) subKarma(n int) { g.Karma = uint8(max(int(g.Karma)-n, 0)) }

// reportCrime costs karma and sends the guards and the frightened citizens after the
// party.
// C: seg_2337:1B0E (C_2337_1B0E)
func (g *Game) reportCrime() {
	g.subKarma(5)
	for n := range npcCount {
		o, npc := g.object(n), &g.NPCs[n]
		if o.Shape == 0 || o.Pos.Z != int(g.MapZ) || npc.Has(Dead) || npc.Has(PlayerControl) ||
			npc.Status&alignmentMask != alignNeutral {
			continue
		}
		distance := g.distance(n, int(g.MapX), int(g.MapY))
		if o.Type() == ObjGuard {
			if distance < guardsHear {
				npc.Mode = AIArrest
			}
			continue
		}
		if npc.Mode == AIImmobile || npc.Mode == AIMotionless || g.incapacitated(n) || g.draggedUnder(n) ||
			distance >= nearCrime {
			continue
		}
		switch {
		case npc.Mode < AISchedule:
			npc.Mode = AIFear
		case npc.Mode > AIPathLost:
			if o.Type() == ObjMusician2 {
				o.Shape = npc.OrigShape
			}
			npc.Mode = AIVigilante
		}
	}
}

// pushedAway says whether character a fails to push character b.
// C: seg_1944:01EC (C_1944_01EC)
func (g *Game) pushedAway(a, b int) bool {
	const odds = 30
	return (g.strengthOf(b)/2+odds-g.strengthOf(a))/2 > g.rand(1, odds)
}

// lookAtTime prints the time of a clock or sundial.
func (g *Game) printTime() {
	g.printf("\n\nThe time is ")
	h, m := int(g.Hour), int(g.Minute)
	switch {
	case h == 0:
		g.printf("12:%02d A.M.\n", m)
	case h < noon:
		g.printf("%d:%02d A.M.\n", h, m)
	case h == noon:
		g.printf("12:%02d P.M.\n", m)
	default:
		g.printf("%d:%02d P.M.\n", h-noon, m)
	}
}

// look describes the selected object or tile.
// C: seg_27A1:0C67 (C_27A1_0C67)
func (g *Game) look() {
	read := false
	g.current.attr |= textAttrLook
	g.printf("Thou dost see ")
	if d := g.pointerDirection(g.PointerX, g.PointerY); d != -1 {
		g.turnTo(int(g.Party[g.Active]), d)
		g.computeView()
	}
	if g.selectedDark {
		g.printf("darkness.\n")
		g.pagingDisabled = true
		return
	}
	pp := g.object(int(g.Party[g.Active])).Pos
	sel := &g.Selection
	if sel.obj == noObject || g.object(sel.obj).Has(StatusInvisible) {
		if g.PointerX < lookViewEdge && g.PointerY < lookViewEdge {
			sel.x = int(g.MapX) - g.viewCenterX + g.PointerX/tileSize
			sel.y = int(g.MapY) - g.viewCenterY + g.PointerY/tileSize
			tile := g.GetTileAtXYZ(sel.x, sel.y, int(g.MapZ))
			g.printArticle(tile)
			g.printf("%s", g.tileString(tile))
		} else {
			g.printf("nothing")
		}
		g.printf(".\n")
		if closeEnough(1, sel.x, sel.y, pp.X, pp.Y) {
			g.printf(searchNoneMsg)
		}
		g.pagingDisabled = true
		return
	}
	t := g.object(sel.obj).Type()
	if sel.obj < npcCount || t == ObjHorse {
		sel.obj = g.combatHead(sel.obj)
	}
	if t == ObjStatueOfMondain || t == ObjStatueOfMinax || t == ObjStatueOfExodus || sel.obj < npcCount {
		if !g.isPlayerControlled(sel.obj) || g.inVehicle() {
			g.printCount(sel.obj)
		}
		g.printf("%s", g.objectString(sel.obj))
		g.putch('\n')
		if !g.inVehicle() {
			g.showCharacter(sel.obj)
		}
	} else {
		tile := g.tileOf(sel.obj)
		count := g.printCount(sel.obj)
		const signBottom, signRight = 1244, 1245 // TIL_4DC, TIL_4DD: the halves of a big sign
		switch tile {
		case signBottom:
			sel.obj = g.FindLoc(sel.x, sel.y-1, int(g.MapZ))
		case signRight:
			sel.obj = g.FindLoc(sel.x-1, sel.y, int(g.MapZ))
		}
		g.printf("%s", g.objectString(sel.obj))
		weight := g.weightOf(sel.obj) + g.encumbrance(sel.obj)
		weighed := weight != 0 && weight < unliftableWeight
		if weighed {
			if count == 1 {
				g.printf(". It weighs")
			} else {
				g.printf(". They weigh")
			}
			g.printf(" %d.%d stones", weight/10, weight%10)
		}
		damage, armour := g.damagePoints(sel.obj), g.armorPoints(sel.obj)
		if damage > 0 {
			switch {
			case !weighed:
				g.printf(". It")
			case armour != 0:
				g.printf(",")
			default:
				g.printf(" and")
			}
			g.printf(" can do %d point\\s of damage", damage)
		}
		if armour > 0 {
			g.printf(" and can absorb %d point\\s of damage", armour)
		}
		switch {
		case g.canRead(sel.obj):
			if g.object(sel.obj).CoordUse() != OnMap {
				if g.object(sel.obj).Quality() != 0 {
					g.printf(":")
					g.readBook(sel.obj)
					read = true
				}
			} else {
				for i := g.FindLoc(sel.x, sel.y, int(g.MapZ)); i >= 0; i = g.NextLoc() {
					if g.object(i).Quality() == 0 {
						continue
					}
					if !read {
						g.printf(":")
					}
					g.readBook(i)
					read = true
				}
			}
			g.putch('\n')
		case t == ObjSpellbook && g.FindInv(sel.obj) >= 0 &&
			(closeEnough(1, sel.x, sel.y, pp.X, pp.Y) || g.object(sel.obj).CoordUse() != OnMap):
			g.lookSpellbook(sel.obj)
		case t == ObjClock || t == ObjSundial && g.Hour >= dawnHour && g.Hour <= duskHour:
			g.printTime()
		default:
			g.printf(".\n")
		}
	}
	if !read {
		g.search(sel.obj, sel.x, sel.y)
	}
	g.pagingDisabled = true
}

// supports says whether a table or the like at (x, y) takes objects.
// C: seg_27A1:1330 (C_27A1_1330)
func (g *Game) supports(x, y int) bool {
	for n := g.FindLoc(x, y, int(g.MapZ)); n >= 0; n = g.NextLoc() {
		tile := g.tileOf(n)
		if g.TileFlag2[tile]&tileSupporting != 0 {
			return true
		}
		if g.isImpassable(tile) {
			return false
		}
	}
	return false
}

// howMany asks how many of object n to take; it returns 0 for none.
// C: seg_27A1:13BE (C_27A1_13BE)
func (g *Game) howMany(n int) int {
	o := g.object(n)
	kind := quantityType(o.Type())
	if kind == countSingle || o.CoordUse() == Equipped {
		return 1
	}
	have := int(o.Amount)
	if kind == countQuantity {
		have = o.Quantity()
	}
	if have == 1 {
		return 1
	}
	sel, row := g.Selection.obj, g.current.curY
	g.printf("How many? ")
	mode := g.SelectMode
	g.drawStatusCursor(0)
	g.SelectMode = 0
	typed := g.gets(4)
	g.SelectMode = mode
	g.Selection.obj = sel
	const answerColumn = 10
	switch v := g.parseNumber(typed, 0); {
	case v == numberEmpty:
		g.gotoxy(answerColumn, row)
		g.printf("%d\n", have)
		return have
	case v > have || v == 0 || v == numberCancelled:
		g.gotoxy(answerColumn, row)
		g.printf("0   \n")
		return 0
	default:
		g.putch('\n')
		return v
	}
}

// aim switches the arrow keys to the aim cursor when they moved in the status area.
func (g *Game) aim() {
	if g.SelectMode != 0 {
		g.drawStatusCursor(0)
		g.SelectMode = selectOnMap
	}
}

// amountToMove is the Amount of count objects like n.
func (g *Game) amountToMove(n, count int) int {
	if quantityType(g.object(n).Type()) == countQuantity {
		return qualQuan(g.object(n).Quality(), count)
	}
	return count
}

// takeOff clears the slot of n when it is worn.
func (g *Game) takeOff(n, member int) {
	if g.object(n).CoordUse() != Equipped {
		return
	}
	if s := g.slotOf(n); s != noSlot {
		g.equipment[s] = emptySlot
	}
	if g.equipSlot(n) == slotTwoHanded {
		g.oneHanded[member] = false
		g.equipment[slotLeftHand] = emptySlot
	}
}

// drop throws the selected object of the status member somewhere.
// C: seg_27A1:14DA (C_27A1_14DA)
// C: seg_27A1:012C (C_27A1_012C), inlined
func (g *Game) drop() {
	sel := g.Selection.obj
	if sel == noObject || g.object(sel).CoordUse() == OnMap || sel == g.container {
		g.printf("%s\n", nothingMsg)
		return
	}
	g.printf("%s", g.objectString(sel))
	g.putch('\n')
	t := g.object(sel).Type()
	kind := quantityType(t)
	actor := int(g.Party[g.statusMember])
	if g.unreadyCheck(sel) == unreadyFree {
		count := g.howMany(sel)
		if count == 0 {
			return
		}
		g.printf("Location: ")
		g.MouseMode = 1
		g.aim()
		g.getch()
		g.putch('\n')
		x, y := g.Selection.x, g.Selection.y
		into := g.FindLoc(x, y, int(g.MapZ))
		if x == -1 || y == -1 ||
			!g.passable(sel, x, y) && !g.supports(x, y) && !g.canPutInto(into, sel) ||
			!g.throw(actor, &g.Selection.x, &g.Selection.y, missileReach+1, 0) {
			g.printf("\n%s\n", notPossibleMsg)
			return
		}
		g.stopWornEffect(sel, actor)
		g.takeOff(sel, g.statusMember)
		g.weighInventory(actor, false)
		g.subMove(actor, 3)
		lx, ly := g.Selection.x, g.Selection.y
		g.throw(actor, &lx, &ly, missileReach+1, g.tileOf(sel))
		ap := g.object(actor).Pos
		switch {
		case !closeEnough(1, g.Selection.x, g.Selection.y, ap.X, ap.Y) && hasType(breakableTypes, t):
			switch {
			case kind != countSingle:
				g.takeObj(actor, t, g.amountToMove(sel, count))
			case t == ObjDragonEgg || t == ObjMirror:
				const crackedEgg, brokenMirror = 1, 2
				if t == ObjDragonEgg {
					g.object(sel).SetFrame(crackedEgg)
				} else {
					g.object(sel).SetFrame(brokenMirror)
				}
				g.MoveObj(sel, lx, ly, int(g.MapZ))
				g.setOkToGet(sel)
			default:
				g.DeleteObj(sel)
			}
			g.sfx(sound.EffectWhoosh, 0)
			g.printf("\nIt broke!\n")
		case g.canPutInto(into, sel) && !g.supports(g.Selection.x, g.Selection.y):
			if kind != countSingle {
				amount := g.amountToMove(sel, count)
				if n := g.giveObj(into, t, amount); n != noObject {
					g.setOkToGet(n)
				}
				g.takeObj(actor, t, amount)
			} else {
				g.InsertObj(sel, into, Contained)
				g.setOkToGet(sel)
			}
		default:
			if kind != countSingle {
				amount := g.amountToMove(sel, count)
				n := g.AddMapObj(shapeOf(t, 0), 0, uint16(amount), lx, ly, int(g.MapZ))
				g.setOkToGet(n)
				g.takeObj(actor, t, amount)
			} else {
				g.MoveObj(sel, lx, ly, int(g.MapZ))
				g.setOkToGet(sel)
			}
		}
		g.computeView()
		g.StatusDirty = false
	}
	g.refreshInventory(g.container)
}

// confirmStealing asks whether to take an object that belongs to someone.
func (g *Game) confirmStealing() bool {
	g.printf("\nStealing! Take it? ")
	return g.yesNo()
}

// get puts the selected object of the map in the active character's pack.
// C: seg_27A1:18F5 (C_27A1_18F5)
func (g *Game) get() {
	actor := int(g.Party[g.Active])
	sel := &g.Selection
	if sel.obj != noObject && (sel.obj < npcCount && sel.x != -1 || g.isIgnored(g.tileOf(sel.obj))) {
		sel.obj = g.objectBelow(sel.obj)
	}
	if sel.obj == noObject || g.object(sel.obj).CoordUse() != OnMap {
		g.printf("%s\n", nothingMsg)
		return
	}
	if d := g.pointerDirection(g.PointerX, g.PointerY); d != -1 {
		g.turnTo(actor, d)
		g.computeView()
	}
	g.printCount(sel.obj)
	g.printf("%s", g.objectString(sel.obj))
	g.putch('\n')
	ap := g.object(actor).Pos
	if !closeEnough(1, sel.x, sel.y, ap.X, ap.Y) {
		g.printf("\n%s!\n", outOfRangeMsg)
		return
	}
	if tile := g.tileOf(sel.obj); g.isDamaging(tile) {
		g.printf("\n%s", notPossibleMsg)
		g.terrainDamage(actor, tile)
		g.putch('\n')
		g.displaySeverity(actor)
		return
	}
	weight := g.weightOf(sel.obj) + g.encumbrance(sel.obj)
	if int(g.Active) != g.statusMember || g.StatusDisplay != cmdStatusEquipment {
		g.weighInventory(actor, false)
	}
	o := g.object(sel.obj)
	t := o.Type()
	switch {
	case weight == 0 && !isLight(t) || weight >= unliftable || sel.obj < npcCount || t == ObjDragon:
		g.printf("\n%s\n", notPossibleMsg)
	case int(g.NPCs[actor].Strength)*carryPerStrength < weight+g.weightInven+g.weightEquip:
		g.printf(tooHeavyMsg)
	case sound.Settings.ShowStealing && !g.okToGet(sel.obj) && g.MapZ == 0 && !g.confirmStealing():
	default:
		g.subMove(actor, 3)
		switch {
		case quantityType(t) != countSingle && (t != ObjTorch || o.Frame() == 0):
			if g.giveObj(actor, t, int(o.Amount)) >= 0 {
				g.DeleteObj(sel.obj)
			}
		case t == ObjTorch && o.Frame() == litTorchFrame:
			if int(g.Active) == g.statusMember {
				switch {
				case g.equipment[slotRightHand] == emptySlot:
					g.equipment[slotRightHand] = sel.obj
				case g.equipment[slotLeftHand] == emptySlot:
					g.equipment[slotLeftHand] = sel.obj
				default:
					g.printf("No free hand\n")
					return
				}
			}
			g.InsertObj(sel.obj, actor, Equipped)
		default:
			g.InsertObj(sel.obj, actor, InInventory)
			switch {
			case t == ObjCandle || t == ObjCandelabra:
				o.SetFrame(0)
			case t == ObjOpenChest && o.Frame() == 0:
				o.SetFrame(1)
			case t == ObjMoonstone:
				g.Moonstones[o.Frame()] = [3]int16{}
			case t == ObjBalloon2:
				g.setType(sel.obj, ObjBalloon)
			}
		}
		if int(g.Active) == g.statusMember && g.StatusDisplay == cmdStatusEquipment {
			g.refreshInventory(int(g.Party[g.statusMember]))
		}
		if !g.okToGet(sel.obj) && g.MapZ == 0 {
			g.setOkToGet(sel.obj)
			g.subKarma(1)
			g.sfx(sound.EffectFailure, 0)
			seen := false
			for n := 1; n < firstMonster; n++ {
				if g.NPCs[n].Status&alignmentMask == alignNeutral && g.isOnScreen(n) && !g.NPCs[n].Has(Asleep) {
					g.printf("\n\"Stop Thief!!!\"\n")
					g.reportCrime()
					seen = true
					break
				}
			}
			if !seen {
				g.printf("\nStealing!!!\n")
			}
		}
		g.computeView()
	}
	if int(g.Active) != g.statusMember {
		g.weighInventory(int(g.Party[g.statusMember]), false)
	}
	g.MouseMode = 0
}

// cannotPush says whether n cannot be pushed from (x, y) toward dir; a diagonal push
// also needs one of the sides free.
// C: seg_27A1:1DAB (C_27A1_1DAB)
func (g *Game) cannotPush(n, x, y, dir int) bool {
	free := func(d int) bool {
		tx, ty := x+dirStepX[d&7], y+dirStepY[d&7]
		return g.passable(n, tx, ty) || g.supports(tx, ty)
	}
	if !free(dir) {
		return true
	}
	if dir&1 == 0 {
		return false
	}
	return !free(dir-1) && !free(dir+1)
}

// carrierOf returns the object on the map that holds n.
// C: seg_27A1:1E69 (C_27A1_1E69)
func (g *Game) carrierOf(n int) int {
	for {
		n = g.object(n).Pos.Owner()
		if g.object(n).CoordUse() == OnMap {
			return n
		}
	}
}

// resetPointerCentre puts the mouse directions back on the party.
func (g *Game) resetPointerCentre() {
	g.pointerCentreX, g.pointerCentreY = g.mapCentre()
	g.updatePointerShape()
}

// move pushes the selected object on the map, or moves an object of the status member
// to another member or container.
// C: seg_27A1:1E8B (C_27A1_1E8B)
func (g *Game) move() {
	sel := &g.Selection
	if sel.obj == noObject || sel.obj == g.container {
		g.printf("%s\n", nothingMsg)
		return
	}
	g.printf("%s", g.objectString(sel.obj))
	g.putch('\n')
	t := g.object(sel.obj).Type()
	kind := quantityType(t)
	if g.object(sel.obj).CoordUse() == OnMap {
		g.push()
	} else {
		g.moveCarried(t, kind)
	}
	g.StatusDirty = false
	g.MouseMode = 0
}

// push is move on the map.
func (g *Game) push() {
	sel := &g.Selection
	if !closeEnough(1, sel.x, sel.y, int(g.MapX), int(g.MapY)) {
		g.printf("\n%s!\n", outOfRangeMsg)
		return
	}
	actor := int(g.Party[g.Active])
	if d := g.pointerDirection(g.PointerX, g.PointerY); d != -1 {
		g.turnTo(actor, d)
		g.computeView()
	}
	o := g.object(sel.obj)
	x, y := o.Pos.X, o.Pos.Y
	g.MouseMode = 0
	centreX, centreY := g.mapCentre()
	g.pointerCentreX = centreX + (x-int(g.MapX))*tileSize
	g.pointerCentreY = centreY + (y-int(g.MapY))*tileSize
	g.printf("To ")
	defer g.resetPointerCentre()
	if g.getch() != cmdAdvance {
		g.printf("nowhere.\n")
		return
	}
	dir := g.AdvanceDir
	g.printf("%s.\n", dirNames[dir])
	weight := g.weightOf(sel.obj) + g.encumbrance(sel.obj)
	if g.TypeWeight[o.Type()] == 0 || o.Type() == ObjDragon {
		g.printf("\n%s\n", notPossibleMsg)
		return
	}
	const lyingFrame = 3
	if sel.obj < npcCount && (g.pushedAway(actor, sel.obj) || actor == sel.obj ||
		o.Frame()&lyingFrame == lyingFrame && o.Type() != ObjDeadBody) {
		g.printf("\n%s\n", failedMsg)
		return
	}
	g.subMove(actor, 5)
	tx, ty := x+dirStepX[dir], y+dirStepY[dir]
	// the original's test, ((frame != dir>>1) & 3), is true whenever the cannon does not
	// face dir
	turnCannon := o.Type() == ObjCannon && o.Frame() != dir>>1
	if into := g.FindLoc(tx, ty, int(g.MapZ)); into != noObject && g.canPutInto(into, sel.obj) && weight < unliftableWeight {
		g.InsertObj(sel.obj, into, Contained)
	} else if g.cannotPush(sel.obj, x, y, dir) {
		g.printf("\n%s\n", blockedMsg)
		if turnCannon {
			o.SetFrame(dir >> 1 & 3)
		}
		return
	} else if turnCannon {
		o.SetFrame(dir >> 1 & 3)
	} else {
		g.MoveObj(sel.obj, tx, ty, int(g.MapZ))
	}
	g.computeView()
}

// moveCarried is move on an object the status member carries.
func (g *Game) moveCarried(t ObjectType, kind int) {
	sel := &g.Selection
	member := g.statusMember
	owner := int(g.Party[member])
	if g.unreadyCheck(sel.obj) != unreadyFree {
		g.refreshInventory(g.container)
		return
	}
	if !g.isOnScreen(owner) {
		g.printf(notAroundMsg)
		return
	}
	count := g.howMany(sel.obj)
	if count == 0 {
		return
	}
	g.printf("To ")
	n := sel.obj
	if t == ObjAmuletOfSubmission {
		g.printf("\n\n%s\n", amuletMsg)
		return
	}
	if g.SelectMode != 0 {
		g.SelectRange = aimRange
		g.drawStatusCursor(0)
		g.SelectMode = selectOnMap
	}
	if g.getch() != cmdSelected || sel.obj == noObject {
		g.printf("nobody.\n")
		return
	}
	dest := sel.obj
	if g.object(dest).CoordUse() == OnMap && (dest < npcCount && !g.isOnScreen(dest) ||
		dest >= npcCount && !closeEnough(1, sel.x, sel.y, int(g.MapX), int(g.MapY))) {
		g.printf("\n\n%s!\n", outOfRangeMsg)
		return
	}
	switch {
	case dest < npcCount && int(g.Party[member]) == dest:
		g.printf("yourself.\n")
		g.turnToPointer()
		g.subMove(int(g.Party[g.Active]), 5)
	case dest < npcCount:
		g.printf("%s", g.objectString(dest))
		g.printf(".\n")
		if !g.isOnScreen(dest) {
			g.printf(notAroundMsg)
			return
		}
		if !g.isPlayerControlled(dest) {
			g.printf("\nOnly within the party!\n")
			return
		}
		g.weighInventory(dest, false)
		var weight int
		switch {
		case kind != countSingle && isLight(t):
			weight = int(g.TypeWeight[t]) * count / 10
		case kind != countSingle:
			weight = int(g.TypeWeight[t]) * count
		default:
			weight = g.weightOf(n) + g.encumbrance(n)
		}
		if int(g.NPCs[dest].Strength)*carryPerStrength < weight+g.weightInven+g.weightEquip {
			g.printf("\nCan't carry!\n")
			g.weighInventory(owner, false)
			return
		}
		g.stopWornEffect(n, owner)
		g.takeOff(n, member)
		if kind != countSingle {
			amount := g.amountToMove(n, count)
			g.giveObj(dest, t, amount)
			g.takeObj(g.object(n).Pos.Owner(), t, amount)
		} else {
			g.InsertObj(n, dest, InInventory)
		}
		g.weighInventory(owner, false)
		g.turnToPointer()
		g.subMove(int(g.Party[g.Active]), 5)
	default:
		if !g.intoContainer(n, dest, t, kind, count) {
			return
		}
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.refreshInventory(g.container)
	}
}

// turnToPointer turns the active character toward the pointer.
func (g *Game) turnToPointer() {
	if d := g.pointerDirection(g.PointerX, g.PointerY); d != -1 {
		g.turnTo(int(g.Party[g.Active]), d)
		g.computeView()
	}
}

// intoContainer moves count of carried object n into (or out of) container c.
func (g *Game) intoContainer(n, c int, t ObjectType, kind, count int) bool {
	ct := g.object(c).Type()
	switch {
	case g.object(c).CoordUse() == OnMap:
		g.printf("nobody.\n")
	case !isContainer(ct):
		g.printf("\n\nnot a container\n")
	case g.carrierOf(c) != g.carrierOf(n):
		g.printf("\n\nCan't move to another person's bag.\n")
	case n < npcCount:
		g.printf("\n\nIt's too big to fit inside a container.\n")
	case g.object(n).CoordUse() == Equipped:
		g.printf("\n\nYou have to unready it first.\n")
	case c == n:
		g.printf("\n\nHow can a container go into itself!\n")
	case ct == ObjSpellbook && t != ObjSpell1:
		g.printf("\n\nOnly spells can go into the spellbook.\n")
	case ct == ObjVortexCube && t != ObjMoonstone:
		g.printf("\n\nOnly moonstones can go into the vortex cube.\n")
	case g.object(n).Pos.Owner() == c && ct == ObjSpellbook && t == ObjSpell1:
		g.printf("\n\nSpells are not suppose to be taken out from the Spellbook.\n")
	default:
		to := c
		if g.object(n).Pos.Owner() == c {
			g.printf("out of ")
			to = g.object(c).Pos.Owner()
		} else {
			g.printf("into ")
		}
		g.subMove(int(g.Party[g.Active]), 5)
		if kind != countSingle {
			o := g.object(n)
			amount := count
			if kind == countQuantity {
				amount = qualQuan(o.Quality(), count)
				if o.Quantity() == count {
					g.DeleteObj(n)
				} else {
					o.setQuantity(o.Quantity() - count)
				}
			} else if int(o.Amount) == count {
				g.DeleteObj(n)
			} else {
				o.Amount -= uint16(count)
			}
			g.giveObj(to, t, amount)
		} else {
			g.InsertObj(n, to, Contained)
		}
		g.invenSize--
		g.printf("%s", g.objectString(c))
		g.putch('\n')
		return true
	}
	return false
}
