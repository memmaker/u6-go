package game

import "github.com/memmaker/u6-go/internal/sound"

// Combat: attacks, damage, death and the combat modes of the AI (seg_2337).

// Messages (seg_356A).
const (
	grazedMsg       = "`%s grazed.\n"
	killedMsg       = "`%s killed!\n"
	draggedUnderMsg = "`%s dragged under!\n"
	dissolvedMsg    = "A slug dissolves %s's %s!\n"
	stolenMsg       = "`%s has been robbed!\n"
	poisonedMsg     = "`%s poisoned!\n"
	slimeDividesMsg = "Slime divides!\n"
	vineKilledMsg   = "Tangle vine killed!\n"
	brokenMsg       = "`%s broken!\n"
	shipCannonMsg   = "with ship cannons-"
	bareHandsMsg    = "with bare hands-"
	withMsg         = "with %s-"
	outOfArrowsMsg  = "Out of arrows!\n"
	outOfBoltsMsg   = "Out of bolts!\n"
	attackPassMsg   = "pass.\n"
	interferesMsg   = "`%s interferes!\n"
	attackWithMsg   = "\nAttack with %s-"
	attackMsg       = "Attack"
	wispTeleportMsg = "Wisp teleports!\n"
	glassSwordMsg   = "Thy sword hath shattered!\n"
	yellsForHelpMsg = "`%s yells for help!\n"
)

// combat is the state of the fights.
type combat struct {
	// hideNextHit keeps showHit from flashing the next hit (D_1968 == 0)
	hideNextHit bool
	// weapons are the equipped weapons of the attacker, weaponIndex the one attacking
	// (D_EC0E, D_1A25, D_1A27)
	weapons     [maxWeapons]int
	weaponCount int
	weaponIndex int
	// lastTarget is the last thing attacked, where the next attack aims (D_04A7)
	lastTarget int
	// boltX, boltY are the targets of the three bolts of a triple crossbow (D_EC02, D_EC08)
	boltX, boltY [tripleBolts]int
	// the centre of the party and of its enemies (PartyGravityX/Y, EnemiesGravityX/Y),
	// the number of enemies (EnemiesNum)
	partyGravityX, partyGravityY int
	enemyGravityX, enemyGravityY int
	enemyCount                   int
	// the front fighters: the enemy in front and the enemies on its left and right
	// (D_EBF7, D_EBF1, D_EBF9), the party member in front and those on its left and right
	// (D_EBED, D_EBE9, D_EBF5)
	enemyFront, enemyLeft, enemyRight int
	partyFront, partyLeft, partyRight int
}

const (
	maxWeapons  = 8
	tripleBolts = 3
	// tileHit is the flash of a hit (TIL_101).
	tileHit = 257
	// tileCombatOn is the combat icon while in combat (TIL_19F).
	tileCombatOn = 415
	// missile tiles: a sling stone or thrown rock (TIL_18E), a cannon ball (TIL_18F), the
	// bolts of the wands (TIL_188, TIL_189)
	tileRock          = 398
	tileCannonBall    = 399
	tileLightningBolt = 392
	tileFireBolt      = 393
	attackMovePoints  = 10
	waitMovePoints    = 5
	opponentRange     = 8 // how far a character looks for an opponent
	nearPartyRange    = 5
	meleeRange        = 2 // CLOSE_ENOUGH_S(2): next to
	areaLast          = areaSize - 1
	swingHearing      = 6 // the distance at which the swing of a weapon is heard
	songEngage        = sound.SongEngage
)

// distance is the larger of the distances on x and y from n to (x, y).
// C: seg_2337:000B (COMBAT_getCathesus)
func (g *Game) distance(n, x, y int) int {
	p := g.object(n).Pos
	return max(abs(p.X-x), abs(p.Y-y))
}

// alignment returns the alignment of n.
func (g *Game) alignment(n int) NPCStatus { return g.NPCs[n].Status & alignmentMask }

// setAlignment changes the alignment of n.
func (g *Game) setAlignment(n int, a NPCStatus) {
	g.NPCs[n].Status = g.NPCs[n].Status&^alignmentMask | a
}

// attacksParty says whether n fights the party (Is_ATKPLR: evil or chaotic).
func (g *Game) attacksParty(n int) bool { return g.NPCs[n].Status&alignEvil != 0 }

// attacksMonsters says whether n fights the monsters (Is_ATKMON: good or chaotic).
func (g *Game) attacksMonsters(n int) bool { return g.NPCs[n].Status&alignGood != 0 }

// backupAlignment is the alignment n had before it was charmed (GetBKAlignment).
func (g *Game) backupAlignment(n int) NPCStatus {
	return NPCStatus(g.NPCs[n].Flags) & alignmentMask
}

// stopCharm ends a charm.
// C: seg_2337:005E (COMBAT_stopCharm)
func (g *Game) stopCharm(n int) {
	g.object(n).Status &^= StatusCharmed
	g.setAlignment(n, g.backupAlignment(n))
	if g.isPlayerControlled(n) {
		g.setAlignment(n, alignGood)
	}
	npc := &g.NPCs[n]
	switch {
	case g.InCombat != 0:
		npc.Mode = npc.CombatMode
	case g.SoloMember < 0:
		npc.Mode = AIFollow
	default:
		npc.Mode = AIMotionless
	}
}

// beginCombat puts the party in combat mode.
// C: seg_2337:00C8 (COMBAT_begin)
func (g *Game) beginCombat() {
	g.InCombat = 1
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		if !g.object(m).Has(StatusCharmed) {
			g.NPCs[m].Mode = g.NPCs[m].CombatMode
		}
		g.NPCs[m].Leader = byte(m)
	}
	member := int(g.SoloMember)
	if member < 0 {
		// fix: the original tests the member before the bound
		for member = 0; member < int(g.PartySize) && (g.incapacitated(int(g.Party[member])) || g.draggedUnder(int(g.Party[member]))); member++ {
		}
		if member == int(g.PartySize) {
			member = 0
		}
	}
	m := int(g.Party[member])
	g.SavedMode = int8(g.NPCs[m].Mode)
	g.NPCs[m].Mode = AICommand
	g.gr.DrawTile(tileCombatOn, combatIconX, iconsY)
	g.playSong(songEngage)
}

// breakOff ends combat mode.
// C: seg_2337:01BC (COMBAT_breakOff)
func (g *Game) breakOff() {
	g.InCombat = 0
	g.heardSomething = false
	mode := AIFollow
	if g.SoloMember >= 0 {
		mode = AIMotionless
	}
	for i := range int(g.PartySize) {
		m := int(g.Party[i])
		switch {
		case g.NPCs[m].Has(Dead):
			g.NPCs[m].Mode = AIMotionless
		case !g.object(m).Has(StatusCharmed):
			g.NPCs[m].Mode = mode
			if mode == AIFollow {
				// the original stores the index, not the character
				g.NPCs[m].Leader = byte(i)
			}
		}
	}
	i := 0
	if g.SoloMember >= 0 {
		i = int(g.SoloMember)
	}
	m := int(g.Party[i])
	g.SavedMode = int8(g.NPCs[m].Mode)
	g.NPCs[m].Mode = AICommand
	g.gr.DrawTile(tileCombatIcon, combatIconX, iconsY)
	if g.song == songEngage {
		g.playAreaSong()
	}
}

// testIntelligence says whether a resists b's magic: the cleverer, the likelier.
// C: seg_2337:0482 (COMBAT_TestIntel)
func (g *Game) testIntelligence(a, b int) bool {
	const odds, bonus = 30, 15
	return g.intelligenceOf(b)/2+bonus-g.intelligenceOf(a) > g.rand(1, odds)
}

// addExperience gives n xp experience points.
// C: seg_2337:04F4 (AddXP)
func (g *Game) addExperience(n, xp int) {
	npc := &g.NPCs[n]
	if int(npc.Experience) < maxExperience-xp {
		npc.Experience += uint16(xp)
	} else {
		npc.Experience = maxExperience
	}
}

// protection is the damage n's equipment absorbs.
// C: seg_2337:09BF (__GetProtection)
func (g *Game) protection(n int) int {
	p := 0
	for i := g.FindInv(n); i >= 0; i = g.NextInv() {
		if g.object(i).CoordUse() == Equipped {
			p += g.armorPoints(i)
		}
	}
	if g.object(n).Has(StatusCursed) {
		p -= cursedPenalty
	}
	if g.NPCs[n].Has(Protected) {
		p += cursedPenalty
	}
	return p
}

// weaponRanges are the ranges of the weapons that reach further than the next tile
// (D_1AD5, D_1AF9).
var weaponRanges = map[ObjectType]int{
	ObjMorningStar: 2, ObjHalberd: 2, ObjDagger: 3, ObjFlaskOfOil: 5, ObjSling: 4, ObjThrowingAxe: 3,
	ObjSpear: 4, ObjBow: 5, ObjCrossbow: 7, ObjSpellbook: 7, ObjCharge: 5, ObjBoomerang: 5,
	ObjTripleCrossbow: 7, ObjShip: 5, ObjZuYlem: 4, ObjMagicBow: 7, ObjLightningWand: 7, ObjFireWand: 7,
}

// weaponRange is how far weapon n reaches.
// C: seg_2337:0A14 (__GetWeaponRange)
func (g *Game) weaponRange(n int) int {
	if r, ok := weaponRanges[g.object(n).Type()]; ok {
		return r
	}
	return 1
}

// interferes says whether n's leader stands in the way of a ranged attack.
// C: seg_2337:0A4E (C_2337_0A4E)
func (g *Game) interferes(n int) bool {
	npc := &g.NPCs[n]
	if npc.Mode <= AIFollow || npc.Mode >= AIGuard2 || int(npc.Leader) == n {
		return false
	}
	l := int(npc.Leader)
	if g.object(l).Shape == 0 || g.incapacitated(l) || g.draggedUnder(l) {
		return false
	}
	p, q := g.object(n).Pos, g.object(l).Pos
	if !closeEnough(meleeRange-1, p.X, p.Y, q.X, q.Y) {
		return false
	}
	return g.SpellFx[spellTimeStop] == 0 || g.isPlayerControlled(l)
}

// tripleDirections are the directions of the side bolts of the triple crossbow for each
// target around the shooter, the first in the high nibble (D_19AC).
var tripleDirections = [viewTiles][viewTiles]byte{
	{66, 82, 82, 82, 82, 98, 99, 99, 99, 99, 100},
	{65, 66, 82, 82, 82, 98, 99, 99, 99, 100, 116},
	{65, 65, 66, 82, 82, 98, 99, 99, 100, 116, 116},
	{65, 65, 65, 66, 82, 98, 99, 100, 116, 116, 116},
	{65, 65, 65, 65, 66, 98, 100, 116, 116, 116, 116},
	{64, 64, 64, 64, 64, 0, 4, 4, 4, 4, 4},
	{48, 48, 48, 48, 32, 38, 6, 5, 5, 5, 5},
	{48, 48, 48, 32, 39, 38, 22, 6, 5, 5, 5},
	{48, 48, 32, 39, 39, 38, 22, 22, 6, 5, 5},
	{48, 32, 39, 39, 39, 38, 22, 22, 22, 6, 5},
	{32, 39, 39, 39, 39, 38, 22, 22, 22, 22, 6},
}

const nibble = 4

// rangedAttack flies the missile of weapon w from attacker a towards (*x, *y) and uses up
// the ammunition.
// C: seg_2337:0B99 (C_2337_0B99)
func (g *Game) rangedAttack(a int, x, y *int, w int) bool {
	t := g.object(w).Type()
	rng := g.weaponRange(w)
	var tile int
	switch {
	case t == ObjBow || t == ObjMagicBow:
		tile = int(g.BaseTile[ObjArrow])
	case t == ObjCrossbow || t == ObjTripleCrossbow:
		tile = int(g.BaseTile[ObjBolt])
	case t == ObjLightningWand:
		tile = tileLightningBolt
	case t == ObjFireWand:
		tile = tileFireBolt
	case t == ObjShip:
		tile = tileCannonBall
	case t == ObjSling || rng > 1 && a == w:
		tile = tileRock
	case a != w:
		tile = g.tileFrame(w)
	}
	ap := g.object(a).Pos
	if t == ObjTripleCrossbow {
		d := tripleDirections[*y-ap.Y+viewCenter][*x-ap.X+viewCenter]
		d1, d2 := int(d>>nibble), int(d&(1<<nibble-1))
		g.boltX = [tripleBolts]int{*x, *x + dirStepX[d1], *x + dirStepX[d2]}
		g.boltY = [tripleBolts]int{*y, *y + dirStepY[d1], *y + dirStepY[d2]}
		bolts := min(g.countObj(a, ObjBolt), tripleBolts)
		g.thrower = a
		g.missiles(ap.X, ap.Y, g.boltX[:bolts], g.boltY[:bolts], rng, tile, 0)
		g.thrower = noObject
		*x, *y = g.boltX[0], g.boltY[0]
	} else {
		g.throw(a, x, y, rng, tile)
	}
	usedUp := false
	switch t {
	case ObjBow, ObjCrossbow, ObjTripleCrossbow, ObjMagicBow:
		ammo := ObjBolt
		if t == ObjBow || t == ObjMagicBow {
			ammo = ObjArrow
		}
		n := 1
		if t == ObjTripleCrossbow {
			n = tripleBolts
		}
		if i := g.FindInvType(a, ammo, -1); i >= 0 {
			if o := g.object(i); o.Quantity() > n {
				o.setQuantity(o.Quantity() - n)
			} else {
				g.DeleteObj(i)
			}
			usedUp = true
		}
	case ObjSpear, ObjThrowingAxe, ObjDagger:
		if abs(*x-ap.X) <= 1 && abs(*y-ap.Y) <= 1 {
			break
		}
		thrown := w
		for i := g.FindInv(a); i >= 0; i = g.NextInv() {
			if o := g.object(i); o.CoordUse() == InInventory && o.Type() == t {
				thrown = i
				break
			}
		}
		usedUp = true
		if g.missileHit >= 0 && g.missileHit < npcCount {
			g.DeleteObj(thrown)
			break
		}
		mx, my := g.missileX, g.missileY
		tile := g.GetTileAtXYZ(mx, my, int(g.MapZ))
		const waterFirst, waterLast = 252, 255 // TIL_0FC..TIL_0FF
		terrain := g.TerrainType[tile]
		lost := terrain&terrainWet != 0 && terrain&terrainImpassable != 0 || tile >= waterFirst && tile <= waterLast
		if lost && g.FindLoc(mx, my, int(g.MapZ)) < 0 {
			g.DeleteObj(thrown)
		} else {
			g.MoveObj(thrown, mx, my, int(g.MapZ))
			g.object(thrown).Status |= StatusOwned
		}
		g.ShowObjects()
	case ObjFlaskOfOil:
		usedUp = true
		g.takeObj(a, ObjFlaskOfOil, 1)
		if g.TerrainType[g.GetTileAtXYZ(g.missileX, g.missileY, int(g.MapZ))]&terrainWet == 0 {
			g.AddMapObj(shapeOf(ObjFireField, 0), StatusLocal, 0, g.missileX, g.missileY, int(g.MapZ))
			g.ShowObjects()
		}
	case ObjZuYlem:
		g.takeObj(a, ObjZuYlem, 1)
		if g.missileHit >= 0 && g.missileHit < npcCount {
			g.sleepSpell(a, g.missileHit)
		}
	case ObjLightningWand, ObjFireWand:
		const odds, vanish = 100, 37
		if g.rand(1, odds) == vanish {
			g.DeleteObj(w)
			g.printf(ringVanishedMsg, "wand")
			g.promptFlag = true
		}
	}
	if usedUp && g.StatusDisplay == cmdStatusEquipment && int(g.Party[g.statusMember]) == a {
		g.StatusDirty = true
	}
	return true
}

// sleepSpell puts n to sleep unless it resists caster: 1 resisted, 2 immune, -2 asleep.
// C: seg_1944:0AD0 (C_1944_0AD0)
func (g *Game) sleepSpell(caster, n int) int {
	cls := monsterClass(g.object(n).Type())
	if cls >= 0 && classHas(cls, immuneSleep) {
		return 2
	}
	if g.testIntelligence(caster, n) {
		return 1
	}
	g.showHit(n)
	g.NPCs[n].Status |= Asleep
	g.setUnconscious(n)
	return -2
}

// cloneObject makes character dst a copy of src (a slime dividing).
// C: seg_2337:115F (CloneObject)
func (g *Game) cloneObject(src, dst int) {
	so, do := g.object(src), g.object(dst)
	do.Status = so.Status
	if so.Type() != ObjSlime {
		do.Shape = so.Shape
	}
	do.Amount = so.Amount
	s, d := &g.NPCs[src], &g.NPCs[dst]
	d.Status, d.Strength, d.Dexterity, d.Intelligence = s.Status, s.Strength, s.Dexterity, s.Intelligence
	d.Magic, d.MovePoints, d.Experience, d.HitPoints = s.Magic, 0, s.Experience, s.HitPoints
	d.Mode, d.CombatMode, d.ScheduleIndex, d.Leader, d.Flags = s.Mode, s.CombatMode, s.ScheduleIndex, s.Leader, s.Flags
}

// showHit flashes a hit on n and wakes it up.
// C: seg_2337:121F (COMBAT_showHit)
func (g *Game) showHit(n int) {
	if !g.hideNextHit {
		p := g.object(n).Pos
		x, y := p.X-int(g.MapX)+viewCenter, p.Y-int(g.MapY)+viewCenter
		if x >= 0 && x < viewTiles && y >= 0 && y < viewTiles {
			g.ShowObjects()
			g.showObject(tileHit, x, y, 1)
			g.drawMap(true)
			g.sfx(sound.EffectNoise, 0)
			g.ShowObjects()
			g.viewChanged = true
		}
		if n < npcCount {
			g.NPCs[n].Flags |= flagSkipTest
		}
	}
	if n < npcCount && g.NPCs[n].Has(Asleep) {
		g.NPCs[n].Status &^= Asleep
		g.object(n).Shape = g.NPCs[n].OrigShape
		g.viewChanged = true
	}
	g.hideNextHit = false
}

// classArmour is the armour of each monster class (D_3522_0142).
var classArmour = [...]byte{
	0, 6, 3, 0, 0, 0, 0, 0, 0, 0, 4, 10, 0, 0, 12, 4, 0, 0, 0, 5, 10, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 30,
	0, 0, 0, 5, 4, 0, 0, 0, 4, 0, 3, 2, 0, 30, 15, 0, 0, 1, 0, 0, 0, 4, 4, 0, 0, 0, 4, 20, 10, 10, 0, 0,
}

// Monster classes with a special attack (IsMonster_xx).
const (
	classCorpser     = 8
	classGazer       = 21
	classGremlin     = 23
	classLordBritish = 31
	classSlime       = 48
	// class flags
	strengthAttack = 32   // hits with strength, not dexterity (MONSTER_0020)
	poisons        = 64   // MONSTER_0040
	disappears     = 2048 // MONSTER_0800
	bleeds         = 8192 // MONSTER_2000
	human          = 16384
)

// loseHitPoints hurts n by hp; it returns the experience for killing it.
// C: seg_2337:12F5 (LooseHP)
func (g *Game) loseHitPoints(n, hp int) int {
	if hp < 0 {
		return 0
	}
	xp := 0
	o := g.object(n)
	t := o.Type()
	if t == ObjBalloon2 {
		return 0
	}
	textOn := g.TextOn
	g.TextOn = true
	defer func() { g.TextOn = textOn }()
	switch {
	case n < npcCount:
		npc := &g.NPCs[n]
		g.showHit(n)
		cls := monsterClass(t)
		switch {
		case cls == classLordBritish:
			npc.HitPoints = byteMask
		case t == ObjSkiff || t == ObjRaft:
			victim := int(g.Party[0])
			if g.PartySize > 1 {
				victim = int(g.Party[g.rand(0, int(g.PartySize)-1)])
			}
			g.loseHitPoints(victim, hp)
			g.displaySeverity(victim)
		case int(npc.HitPoints) <= hp:
			if g.lastTarget == n {
				g.lastTarget = noObject
			}
			if t == ObjShip {
				g.moveBigCreature(n, o.Pos.X, o.Pos.Y, true)
				o.Shape = shapeOf(ObjRaft, 0)
				const raftHitPoints = 10
				npc.HitPoints = raftHitPoints
				g.viewChanged = true
				break
			}
			if npc.HitPoints != 0 {
				xp = g.kill(n)
			}
			if int(g.Party[g.Active]) == n {
				g.checkActiveMember()
			}
		default:
			npc.HitPoints -= byte(hp)
			if npc.Mode == AIShy {
				npc.Mode = AIAssault
			}
			if cls == classSlime {
				g.divideSlime(n)
			}
		}
		if g.isPlayerControlled(n) {
			g.showHitPoints(n)
		}
	case t == ObjTangleVine2:
		head := o.Quality()
		g.NPCs[head].HitPoints = 0
		g.NPCs[head].Status |= Dead
		g.object(head).Status &^= StatusInvisible
		g.NPCs[head].Mode = AIMotionless
		g.printf(vineKilledMsg)
		g.promptFlag = true
	case quantityType(t) == countSingle && o.Quantity() != 0:
		g.showHit(n)
		const openDoorFrames, emptyChest = 12, 3
		if t >= ObjOakenDoor && t <= ObjSteelDoor && o.Frame() >= openDoorFrames || t == ObjOpenChest && o.Frame() == emptyChest {
			break
		}
		if o.Quantity() > hp {
			o.setQuantity(o.Quantity() - hp)
			break
		}
		g.printf(brokenMsg, g.objectString(n))
		g.promptFlag = true
		if g.FindInvType(n, ObjEffect, trapSpell) != noObject {
			g.trap(n)
		}
		if t == ObjMirror {
			const brokenMirror = 2
			g.sfx(sound.EffectWhoosh, 0)
			o.SetFrame(brokenMirror)
			g.viewChanged = true
			g.subKarma(10)
		} else {
			g.DeleteObj(n)
			g.viewChanged = true
		}
	}
	return xp
}

// kill makes n dead; it returns the experience for killing it.
// C: seg_2337:12F5 (LooseHP), the death
func (g *Game) kill(n int) int {
	npc := &g.NPCs[n]
	xp := 0
	if g.alignment(n) == alignNeutral {
		g.subKarma(5)
	}
	npc.HitPoints = 0
	npc.Status |= Dead
	if g.attacksParty(n) {
		if cls := monsterClass(g.object(n).Type()); cls >= 0 {
			xp = (int(classStrength[cls]) + int(classDexterity[cls]) + int(classIntelligence[cls]) +
				int(classArmour[cls]) + int(classDamage[cls]) + int(classHitPoints[cls])) >> 2
		}
	}
	g.object(n).Status &^= StatusInvisible
	g.setUnconscious(n)
	npc.Mode = AIMotionless
	if g.isPlayerControlled(n) && int(g.Party[0]) != n {
		for i := 1; i < int(g.PartySize); i++ {
			if int(g.Party[i]) != n {
				continue
			}
			if int(g.SoloMember) >= i {
				g.SoloMember--
			}
			if int(g.Active) == i {
				g.SetPartyMode()
			}
			if int(g.Active) > i {
				g.Active--
			}
			copy(g.Party[i:], g.Party[i+1:])
			copy(g.Names[i:], g.Names[i+1:])
			g.PartySize--
			g.StatusDirty = true
			break
		}
	}
	g.dropBody(n)
	g.viewChanged = true
	return xp
}

// divideSlime splits a hurt slime into two.
// C: seg_2337:12F5 (LooseHP), the slime
func (g *Game) divideSlime(n int) {
	p := g.object(n).Pos
	for range 8 {
		x, y := p.X+g.rand(0, 2)-1, p.Y+g.rand(0, 2)-1
		if !g.passable(n, x, y) {
			continue
		}
		if c := g.AddMonster(g.object(n).Type(), x, y, int(g.MapZ)); c >= 0 {
			g.cloneObject(n, c)
			g.printf(slimeDividesMsg)
			g.promptFlag = true
			g.viewChanged = true
		}
		return
	}
}

// showHitPoints redraws the hit points of party member n.
// C: seg_2337:12F5 (LooseHP), the status
func (g *Game) showHitPoints(n int) {
	switch {
	case g.StatusDisplay == cmdStatusPortrait && int(g.Party[g.statusMember]) == n:
		g.StatusDirty = true
	case g.StatusDisplay == cmdStatusParty:
		const lowHitPoints = 10
		for i := g.partyScroll; i < int(g.PartySize) && i < g.partyScroll+partyListRows; i++ {
			if int(g.Party[i]) != n {
				continue
			}
			g.setClip(clipParty)
			g.gotoxy(14, (i-g.partyScroll)<<1+2)
			if g.NPCs[n].Has(Poisoned) {
				g.current.fg = g.colours.poisoned
			} else if g.NPCs[n].HitPoints < lowHitPoints {
				g.current.fg = g.colours.highlight
			}
			g.printf("%3d\n", int(g.NPCs[n].HitPoints))
			g.current.fg = g.colours.text
			g.setClip(clipMessage)
		}
	}
}

// dropBody leaves the body of n with its possessions; it returns the body.
// C: seg_1184:1B3A (C_1184_1B3A)
func (g *Game) dropBody(n int) int {
	if n < 0 || g.object(n).Shape == 0 {
		return noObject
	}
	o := g.object(n)
	x, y, z := o.Pos.X, o.Pos.Y, o.Pos.Z
	first := g.FindLoc(x, y, z)
	g.allowSubmissionAmulet = true
	defer func() { g.allowSubmissionAmulet = false }()
	body := g.AddMapObj(o.Shape, o.Status|StatusOwned, uint16(qualQuan(n, 0)), x, y, z)
	if body < 0 {
		return body
	}
	if n < firstMonster {
		g.object(body).Status &^= StatusLocal
	}
	for i := g.FindInv(n); i >= 0; i = g.FindInv(n) {
		g.InsertObj(i, body, Contained)
		g.object(i).Status |= StatusOwned
	}
	if first >= 0 && g.object(first).Type() == ObjBlood {
		g.MoveObj(first, x, y, z)
	}
	const nowhere = 15
	g.MoveObj(n, x, y, nowhere)
	return body
}

// raiseBody turns body back into the character it was; it returns the character.
// C: seg_1184:1C52 (C_1184_1C52)
func (g *Game) raiseBody(body int) int {
	if body < 0 {
		return noObject
	}
	g.allowSubmissionAmulet = true
	defer func() { g.allowSubmissionAmulet = false }()
	n := g.object(body).Quality()
	at := body
	for g.object(at).CoordUse() != OnMap {
		at = g.object(at).Pos.Owner()
	}
	p := g.object(at).Pos
	g.MoveObj(n, p.X, p.Y, p.Z)
	for i := g.FindInv(body); i >= 0; i = g.FindInv(body) {
		if g.object(i).Type() == ObjAmuletOfSubmission {
			g.InsertObj(i, n, Equipped)
		} else {
			g.InsertObj(i, n, InInventory)
		}
	}
	g.DeleteObj(body)
	return n
}

// resurrect brings body back to life next to where it lies.
// C: seg_1944:1A42 (C_1944_1A42)
func (g *Game) resurrect(body int) {
	n := g.raiseBody(body)
	if n < 0 {
		// fix: the original goes on with no body to raise
		return
	}
	npc := &g.NPCs[n]
	npc.Status &^= Dead
	npc.HitPoints = 1
	g.object(n).Shape = npc.OrigShape
	npc.Magic = byte(g.maxMagic(n))
	npc.Status &^= Dead | Poisoned | Asleep | Paralyzed | Protected
	const fullKarma, percent = 98, 100
	if g.Karma < fullKarma {
		npc.Experience = uint16(int(npc.Experience) * int(g.Karma) / percent)
	}
	p := g.object(n).Pos
	placed := false
	try := func(x, y int) bool {
		if g.passable(n, x, y) {
			g.MoveObj(n, x, y, int(g.MapZ))
			placed = true
		}
		return placed
	}
	const rings = 5
	for d := 1; d < rings && !placed; d++ {
		for k := 0; k < d<<1+1; k++ {
			if try(p.X-d+k, p.Y-d) || try(p.X-d, p.Y-d+k) || try(p.X+d-k, p.Y+d) || try(p.X+d, p.Y+d-k) {
				break
			}
		}
	}
	if !placed {
		g.MoveObj(n, p.X, p.Y, int(g.MapZ))
	}
	if g.isPlayerControlled(n) {
		g.joinParty(n)
	}
}

// hits says whether attacker a hits defender d with weapon w.
// C: seg_2337:18CF (__AttackTest)
func (g *Game) hits(a, d, w int) bool {
	if d >= npcCount {
		return true
	}
	t := g.object(w).Type()
	if t == ObjGlassSword {
		return true
	}
	var strong bool
	if w == a {
		strong = classHas(monsterClass(t), strengthAttack)
	} else {
		strong = t == ObjSpikedHelm || t == ObjClub || t == ObjMace || t == ObjTwoHandedHammer
	}
	skill := g.dexterityOf(a)
	if strong {
		skill = g.strengthOf(a)
	}
	const odds = 30
	return g.rand(1, odds) >= (g.dexterityOf(d)+odds-skill)/2
}

// dissolvable are what acid slugs eat (D_1B7A); robbable what gremlins steal (D_1BA2).
var (
	dissolvable = []ObjectType{
		ObjChainCoif, ObjIronHelm, ObjSpikedHelm, ObjWingedHelm, ObjGargoyleHelm, ObjCurvedHeater, ObjWingedHelm2,
		ObjKiteShield, ObjSpikedShield, ObjDoorShield, ObjRingMail, ObjScaleMail, ObjChainMail, ObjPlateMail,
		ObjMainGauche, ObjDagger, ObjSword, ObjCleaver, ObjKnife,
	}
	robbable = []ObjectType{
		ObjLoafOfBread, ObjPortionOfMeat, ObjRolls, ObjCake, ObjCheese, ObjHam, ObjHorseChops, ObjMeat, ObjRibs,
		ObjDeadAnimal,
	}
)

// pickItem picks one of n's dissolvable (or, with rob, robbable) things at random.
// C: seg_2337:1A99 (C_2337_1A99)
func (g *Game) pickItem(n int, rob bool) int {
	list := dissolvable
	if rob {
		list = robbable
	}
	picked := noObject
	for i := g.FindInv(n); i >= 0; i = g.NextInv() {
		for _, t := range list {
			if g.object(i).Type() == t {
				if picked < 0 || g.rand(0, 1) != 0 {
					picked = i
				}
				break
			}
		}
	}
	return picked
}

// provoke makes a neutral victim fight back against attacker; hp is the damage done.
// C: seg_2337:1C1E (C_2337_1C1E)
func (g *Game) provoke(victim, attacker, hp int) {
	if victim >= npcCount || g.alignment(victim) != alignNeutral || g.isPlayerControlled(victim) {
		return
	}
	cls := monsterClass(g.object(victim).Type())
	if g.NPCs[attacker].Mode == AICommand && cls >= 0 && classHas(cls, human) && hp > 0 {
		g.printf(yellsForHelpMsg, g.objectString(victim))
		g.reportCrime()
		g.promptFlag = true
	}
	if g.NPCs[victim].Mode >= AISchedule {
		if g.NPCs[attacker].Mode == AICommand {
			if g.object(victim).Type() == ObjMusician2 {
				g.object(victim).Shape = g.NPCs[victim].OrigShape
			}
			g.NPCs[victim].Mode = AIVigilante
		}
		return
	}
	if g.alignment(attacker) == alignGood {
		g.setAlignment(victim, alignChaotic)
	} else {
		g.setAlignment(victim, alignGood)
	}
	if victim >= firstMonster {
		g.NPCs[victim].Mode = AIAssault
	}
}

// doHit makes a hit d with up to hp damage.
// C: seg_2337:1D20 (__DoHit)
func (g *Game) doHit(a, d, hp int) {
	if d < 0 {
		return
	}
	name := g.objectString(d)
	if g.object(a).Shape == shapeOf(ObjCorpser, 0) {
		g.object(a).SetFrame(1)
		g.viewChanged = true
	}
	const deadly = 255
	if hp == -1 {
		hp = 1
	} else if hp > 1 && hp < deadly {
		hp = g.rand(1, hp)
	}
	armour := 0
	if d < npcCount {
		armour = g.protection(d)
	}
	if armour > 0 && hp < deadly {
		hp -= g.rand(1, armour)
	}
	if hp <= 0 {
		g.showHit(d)
		g.printf(grazedMsg, name)
		g.promptFlag = true
	} else {
		if d < npcCount && g.NPCs[d].Mode > AIFollow && g.NPCs[d].Mode < AIGuard2 {
			p, q := g.object(d).Pos, g.object(a).Pos
			if closeEnough(meleeRange-1, p.X, p.Y, q.X, q.Y) {
				g.NPCs[d].Leader = byte(a)
			}
		}
		g.addExperience(a, g.loseHitPoints(d, hp))
	}
	if d >= npcCount {
		return
	}
	if d == g.Selection.obj {
		g.provoke(d, a, hp)
	}
	switch t := g.object(d).Type(); {
	case g.NPCs[d].HitPoints == 0 && t != ObjBalloon2 && t != ObjSkiff && t != ObjRaft && t != ObjShip:
		g.printf(killedMsg, name)
		g.promptFlag = true
		if g.StatusDisplay == cmdStatusPortrait && int(g.Party[g.statusMember]) == a {
			g.StatusDirty = true
		}
		return
	}
	cls := monsterClass(g.object(a).Type())
	if cls == classCorpser {
		g.sfx(sound.EffectSweepShort, 0)
		g.printf(draggedUnderMsg, name)
		g.NPCs[d].Flags |= flagDraggedUnder
		g.promptFlag = true
		g.viewChanged = true
	}
	if cls == classAcidSlug {
		if i := g.pickItem(d, false); i >= 0 {
			g.sfx(sound.EffectSplatter, 0)
			g.printf(dissolvedMsg, g.objectString(d), g.objectString(i))
			g.promptFlag = true
			g.DeleteObj(i)
		}
	}
	if cls == classGremlin {
		if i := g.pickItem(d, true); i >= 0 {
			g.sfx(sound.EffectSweep, 0)
			g.printf(stolenMsg, g.objectString(d))
			g.promptFlag = true
			g.DeleteObj(i)
		}
	}
	if classHas(cls, poisons) && cls >= 0 && g.rand(0, 3) == 0 {
		g.printf(poisonedMsg, name)
		g.promptFlag = true
		g.NPCs[d].Status |= Poisoned
		if g.StatusDisplay == cmdStatusParty || g.StatusDisplay == cmdStatusPortrait && int(g.Party[g.statusMember]) == d {
			g.StatusDirty = true
		}
	}
	if hp > 0 {
		g.displaySeverity(d)
	}
}

// Results of tryHit.
const (
	hitDone       = 0
	hitInterfered = 1
	hitOutOfRange = 2
)

// rangedWeapons are always fired, even at the next tile.
var rangedWeapons = map[ObjectType]bool{
	ObjSling: true, ObjThrowingAxe: true, ObjBow: true, ObjCrossbow: true, ObjBoomerang: true,
	ObjFlaskOfOil: true, ObjTripleCrossbow: true, ObjZuYlem: true, ObjLightningWand: true, ObjFireWand: true,
}

// tryHit makes a attack (x, y) with weapon w.
// C: seg_2337:214D (__TryHit)
func (g *Game) tryHit(a, x, y, w int) int {
	ap := g.object(a).Pos
	if g.combatRange(x, y, ap.X, ap.Y) > g.weaponRange(w) {
		return hitOutOfRange
	}
	wt := g.object(w).Type()
	if (wt == ObjBow || wt == ObjMagicBow) && g.FindInvType(a, ObjArrow, -1) < 0 ||
		(wt == ObjCrossbow || wt == ObjTripleCrossbow) && g.FindInvType(a, ObjBolt, -1) < 0 {
		return hitDone
	}
	deltaX, deltaY := x-ap.X, y-ap.Y
	var dir int
	if abs(deltaX) > abs(deltaY) {
		dir = dirEast
		if deltaX < 0 {
			dir = dirWest
		}
	} else {
		dir = dirNorth
		if deltaY >= 0 {
			dir = dirSouth
		}
	}
	at := g.object(a).Type()
	if at == ObjShip {
		// a ship fires its cannons broadside
		if (g.NPCs[a].Direction()&6^dir)&2 != 0 {
			dir = g.NPCs[a].Direction()
		} else {
			dir ^= 2
		}
	}
	g.NPCs[a].SetDirection(dir)
	g.turnTo(a, dir)
	if at >= ObjSkeleton && at <= ObjMongbat {
		const dirFrames, attackFrame = 12, 2
		g.object(a).SetFrame(g.object(a).Frame()&dirFrames + attackFrame)
	}
	g.updateViewNow()
	g.missileHit = g.Selection.obj
	ranged := false
	if wt != ObjMorningStar && wt != ObjHalberd {
		ranged = deltaX > 1 || deltaX < -1 || deltaY > 1 || deltaY < -1 || rangedWeapons[wt]
	}
	if ranged && g.interferes(a) {
		return hitInterfered
	}
	bolts := 0
	if wt == ObjTripleCrossbow {
		bolts = min(g.countObj(a, ObjBolt), tripleBolts)
	}
	damage := max(g.damagePoints(w), 1)
	const unknown = -1
	hit := unknown
	opponent := g.combatHead(g.missileHit)
	if wt == ObjCharge || wt == ObjSpellbook {
		return g.castInCombat(a, opponent, w)
	}
	if g.missileHit >= 0 && opponent < npcCount {
		if h := g.hits(a, opponent, w); !h && ranged {
			x += g.rand(0, 2) - 1
			y += g.rand(0, 2) - 1
			g.missileHit = g.objectAtForSelection(a, x, y)
		} else {
			hit = boolInt(h)
		}
	}
	if ranged && !g.rangedAttack(a, &x, &y, w) {
		hit = unknown
	}
	opponent = g.combatHead(g.missileHit)
	if !ranged && g.distance(a, int(g.MapX), int(g.MapY)) < swingHearing {
		g.sfx(sound.EffectSwing, 0)
	}
	if hit == unknown {
		hit = 1
		if g.missileHit >= 0 && opponent < npcCount {
			hit = boolInt(g.hits(a, opponent, w))
		}
	}
	if wt == ObjTripleCrossbow {
		for i := range bolts {
			if i > 0 {
				g.missileHit = g.objectAtForSelection(a, g.boltX[i], g.boltY[i])
				opponent = g.combatHead(g.missileHit)
				hit = boolInt(g.hits(a, opponent, w))
			}
			if hit != 0 && g.missileHit >= 0 && opponent != a {
				g.doHit(a, opponent, damage)
			}
		}
		return hitDone
	}
	if hit != 0 && g.missileHit >= 0 && opponent != a {
		if classHas(monsterClass(g.object(opponent).Type()), undead) &&
			wt != ObjGlassSword && wt != ObjTripleCrossbow && wt != ObjMagicBow {
			damage = (damage + 1) >> 1
		}
		if wt != ObjZuYlem {
			g.doHit(a, opponent, damage)
		}
		if wt == ObjGlassSword {
			g.printf(glassSwordMsg)
			g.DeleteObj(w)
			g.promptFlag = true
		}
	}
	if wt == ObjBoomerang {
		const anyRange = 100
		p := g.object(a).Pos
		g.missiles(p.X, p.Y, []int{x}, []int{y}, anyRange, int(g.BaseTile[ObjBoomerang]), missileReverse)
	}
	return hitDone
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// listWeapons lists the equipped weapons of the active character and names the first.
// C: seg_2337:27C5 (C_2337_27C5)
func (g *Game) listWeapons() {
	n := int(g.Party[g.Active])
	g.weaponCount, g.weaponIndex = 0, 0
	for i := g.FindInv(n); i >= 0 && g.weaponCount < maxWeapons; i = g.NextInv() {
		if g.object(i).CoordUse() == Equipped && g.damagePoints(i) >= 0 {
			g.weapons[g.weaponCount] = i
			g.weaponCount++
		}
	}
	switch {
	case g.weaponCount != 0:
		g.printf(withMsg, g.objectString(g.weapons[0]))
	case g.object(n).Type() == ObjShip:
		g.printf(shipCannonMsg)
	default:
		g.printf(bareHandsMsg)
	}
}

// attackWithNext attacks the selection with the current weapon and goes to the next.
// C: seg_2337:2875 (C_2337_2875)
func (g *Game) attackWithNext() {
	n := int(g.Party[g.Active])
	w := n
	if g.weaponCount != 0 {
		w = g.weapons[g.weaponIndex]
	}
	t := g.object(w).Type()
	switch {
	case (t == ObjBow || t == ObjMagicBow) && g.FindInvType(n, ObjArrow, -1) < 0:
		g.printf(outOfArrowsMsg)
	case (t == ObjCrossbow || t == ObjTripleCrossbow) && g.FindInvType(n, ObjBolt, -1) < 0:
		g.printf(outOfBoltsMsg)
	default:
		switch {
		case g.Selection.obj == n:
			g.printf(attackPassMsg)
		case g.Selection.obj >= 0:
			g.printf("%s.\n", g.objectString(g.Selection.obj))
		case g.selectedDark:
			g.printf(nothingMsg + "!\n")
		default:
			g.printf("%s.\n", g.tileString(g.GetTileAtXYZ(g.Selection.x, g.Selection.y, int(g.MapZ))))
		}
		if g.Selection.obj == n {
			break
		}
		g.printf("\n")
		g.skipNewline = true
		if g.Selection.obj < 0 {
			g.Selection.obj = g.objectAtForSelection(n, g.Selection.x, g.Selection.y)
		}
		switch g.tryHit(n, g.Selection.x, g.Selection.y, w) {
		case hitInterfered:
			g.sfx(sound.EffectSweep, 0)
			g.printf(interferesMsg, g.objectString(int(g.NPCs[n].Leader)))
		case hitOutOfRange:
			g.printf(outOfRangeMsg + "!\n")
			g.lastTarget = noObject
			g.sfx(sound.EffectSweep, 0)
		}
	}
	g.weaponIndex++
	if g.weaponIndex < g.weaponCount {
		g.SelectMode = 1
		g.SelectRange = g.weaponRange(g.weapons[g.weaponIndex])
		if g.combatRange(g.AimX, g.AimY, viewCenter, viewCenter) > g.SelectRange {
			g.AimX, g.AimY = viewCenter, viewCenter
		}
		g.printf(attackWithMsg, g.objectString(g.weapons[g.weaponIndex]))
	} else {
		g.MouseMode = 0
		g.SelectMode = 0
		g.updatePointerShape()
	}
}

// attack is the Attack command.
// C: seg_2337:2AA9 (COMBAT_attack)
func (g *Game) attack() {
	g.printf(attackMsg)
	n := int(g.Party[g.Active])
	if g.inVehicle() && g.object(n).Type() != ObjShip {
		g.printf("-")
		g.printf(notOnShipMsg)
		return
	}
	g.printf(" ")
	g.MouseMode = 1
	g.pendingCommand = cmdAttack
	g.updatePointerShape()
	g.listWeapons()
	w := n
	if g.weaponCount != 0 {
		w = g.weapons[g.weaponIndex]
	}
	g.SelectRange = g.weaponRange(w)
	if g.combatRange(g.AimX, g.AimY, viewCenter, viewCenter) > g.SelectRange {
		g.AimX, g.AimY = viewCenter, viewCenter
	}
	ch := g.getch()
	g.lastTarget = g.Selection.obj
	if ch == cmdSelected && (g.Selection.obj < 0 || g.object(g.Selection.obj).CoordUse() == OnMap) {
		for g.MouseMode == 1 {
			g.attackWithNext()
			if g.MouseMode == 1 {
				g.RefreshStatus()
				if g.getch() != cmdSelected {
					g.printf(whatMsg)
					break
				}
			}
		}
	} else {
		g.printf(whatMsg)
	}
	g.subMove(n, attackMovePoints)
	g.MouseMode = 0
	g.SelectMode = 0
	g.updatePointerShape()
}

// combatRange is the distance used for ranges.
// C: seg_2337:2BEE (COMBAT_range)
func (g *Game) combatRange(x0, y0, x1, y1 int) int {
	deltaX, deltaY := abs(x0-x1), abs(y0-y1)
	if deltaX >= len(rangeTable) || deltaY >= len(rangeTable) {
		const outOfReach = 9
		return outOfReach
	}
	return rangeTable[deltaX][deltaY]
}

// pickWeapon picks the weapon a fights d with: sometimes a spell, else the strongest
// that reaches.
// C: seg_2337:2C43 (__PickWeapon)
func (g *Game) pickWeapon(a, d int) int {
	picked, best := a, 0
	dp, ap := g.object(d).Pos, g.object(a).Pos
	rng := g.combatRange(dp.X, dp.Y, ap.X, ap.Y)
	for i := g.FindInv(a); i >= 0; i = g.NextInv() {
		o := g.object(i)
		if o.CoordUse() != Equipped {
			continue
		}
		if t := o.Type(); t == ObjSpellbook || t == ObjCharge {
			const spellLevels = 16
			if g.rand(0, 3) == 0 && g.SpellFx[spellStormCloak] == 0 && g.rand(0, o.Quality()) < spellLevels {
				return i
			}
		}
		if dmg := g.damagePoints(i); dmg > best && g.weaponRange(i) >= rng {
			best, picked = dmg, i
		}
	}
	return picked
}

// pickOpponent returns the nearest character n wants to fight, or -1.
// C: seg_2337:2D6E (COMBAT_pickOpponent)
func (g *Game) pickOpponent(n int) int {
	picked := noObject
	align := g.alignment(n)
	p := g.object(n).Pos
	best := 1<<15 - 1
	mode := g.NPCs[n].Mode
	for i := range npcCount {
		if g.object(i).Shape == 0 || g.NPCs[i].Has(Dead) || i == n || !g.canSee(n, i) {
			continue
		}
		if mode != AIShy && mode != AIFlee && mode != AIFear && mode != AIRetreat && mode != AIBrawl {
			other := g.alignment(i)
			if align == alignNeutral && (mode != AIVigilante || other != alignGood) ||
				align == alignChaotic && other == alignChaotic || other == alignNeutral ||
				align == alignGood && !g.attacksParty(i) || align == alignEvil && !g.attacksMonsters(i) {
				continue
			}
		}
		q := g.object(i).Pos
		if g.distance(n, q.X, q.Y) > opponentRange {
			continue
		}
		if g.NPCs[i].Mode == AIRetreat && !closeEnough(nearPartyRange, q.X, q.Y, int(g.MapX), int(g.MapY)) {
			continue
		}
		d := (q.X-p.X)*(q.X-p.X) + (q.Y-p.Y)*(q.Y-p.Y)
		if d < best || d == best && g.rand(0, 1) != 0 {
			best, picked = d, i
		}
	}
	return picked
}

// inReach says whether (x, y) is within rng of (x0, y0).
func inReach(x, y, x0, y0, rng int) bool {
	deltaX, deltaY := abs(x-x0), abs(y-y0)
	return deltaX < len(rangeTable) && deltaY < len(rangeTable) && rangeTable[deltaX][deltaY] <= rng
}

// sidestep returns the place beside (x, y) a quarter turn from the selection.
func (g *Game) sidestep(x, y int) (int, int) {
	s := g.object(g.Selection.obj).Pos
	if g.rand(0, 1) != 0 {
		return x - (s.Y - y), y + (s.X - x)
	}
	return x + (s.Y - y), y - (s.X - x)
}

// fireAt makes n attack the selection with weapon w; when the way is blocked it returns
// where to step instead and true.
func (g *Game) fireAt(n, w, rng int, tx, ty *int) bool {
	if g.throw(n, tx, ty, rng, 0) {
		g.tryHit(n, *tx, *ty, w)
		g.subMove(n, attackMovePoints)
		return false
	}
	p := g.object(n).Pos
	*tx, *ty = g.sidestep(p.X, p.Y)
	return true
}

// assault attacks the nearest opponent, or walks towards it.
// C: seg_2337:2F3E (COMBAT_AI_Assault)
func (g *Game) assault(n int) {
	g.Selection.obj = g.pickOpponent(n)
	if g.Selection.obj < 0 {
		if g.NPCs[n].Mode == AIFront {
			g.subMove(n, waitMovePoints)
		} else {
			g.wander(n)
		}
		return
	}
	// fix: C picks the weapon before checking for an opponent, reading object -1.
	w := g.pickWeapon(n, g.Selection.obj)
	s, p := g.object(g.Selection.obj).Pos, g.object(n).Pos
	ox, oy := s.X, s.Y
	rng := g.weaponRange(w)
	switch {
	case closeEnough(opponentRange-1, ox, oy, p.X, p.Y) && rangeTable[abs(ox-p.X)][abs(oy-p.Y)] <= rng:
		if g.fireAt(n, w, rng, &ox, &oy) {
			g.TryMoveTo(n, ox, oy)
		}
	case g.NPCs[n].Mode == AIFront:
		g.subMove(n, waitMovePoints)
	default:
		g.TryMoveTo(n, ox, oy)
	}
}

// holdLine keeps n in the front line between the party and its enemies; it returns true
// when n should fight instead.
// C: seg_2337:310E (C_2337_310E)
func (g *Game) holdLine(n int) bool {
	player := g.isPlayerControlled(n)
	// fix: the original tests for object 0, not for no enemy in front (-1)
	if (g.enemyCount == 0 || g.enemyFront <= 0) && !player {
		g.subMove(n, waitMovePoints)
		return true
	}
	p := g.object(n).Pos
	var ox, oy, deltaX, deltaY, front int
	if player {
		lp := g.object(int(g.Party[0])).Pos
		ox, oy = lp.X, lp.Y
		deltaX, deltaY = g.enemyGravityX-ox, g.enemyGravityY-oy
		front = int(g.Party[0])
	} else {
		ox, oy = g.partyGravityX, g.partyGravityY
		fp := g.object(g.enemyFront).Pos
		deltaX, deltaY = fp.X-ox, fp.Y-oy
		front = g.enemyFront
	}
	side := (p.X-ox)*deltaY - (p.Y-oy)*deltaX
	right := side > 0
	if side == 0 {
		right = g.rand(0, 1) != 0
	}
	stepX, stepY := -1, 1
	if deltaY > 0 {
		stepX = 1
	}
	if deltaX > 0 {
		stepY = -1
	}
	if !right {
		stepX, stepY = -stepX, -stepY
	}
	fp := g.object(front).Pos
	depth := (fp.X-ox)*deltaX + (fp.Y-oy)*deltaY
	if int(g.Party[0]) == front {
		depth += abs(deltaX) + abs(deltaY)
	}
	length := max(deltaX*deltaX+deltaY*deltaY, 1)
	tx, ty := depth*deltaX/length+ox, depth*deltaY/length+oy
	g.allowPartySwap = false
	for {
		if tx < g.AreaX || g.AreaX+areaLast < tx || ty < g.AreaY || g.AreaY+areaLast < ty || p.X == tx && p.Y == ty {
			g.allowPartySwap = true
			return true
		}
		if g.livingAt(0, tx, ty) < 0 {
			break
		}
		a := (tx+stepX-ox)*deltaX + (ty-oy)*deltaY
		b := (tx-ox)*deltaX + (ty+stepY-oy)*deltaY
		if abs(a-depth) < abs(b-depth) {
			tx += stepX
		} else {
			ty += stepY
		}
	}
	g.allowPartySwap = true
	points := g.NPCs[n].MovePoints
	if !g.TryMoveTo(n, tx, ty) {
		g.NPCs[n].MovePoints = points
	} else {
		if !player {
			return false
		}
		if q := g.object(n).Pos; q.X == tx && q.Y == ty {
			return false
		}
		if g.enemyCount == 0 {
			g.NPCs[n].MovePoints = points
		}
		if g.TryMoveTo(n, tx, ty) {
			return false
		}
	}
	g.assault(n)
	return false
}

// front fights in the front line.
// C: seg_2337:3464 (COMBAT_AI_Front)
func (g *Game) front(n int) {
	if !g.holdLine(n) {
		return
	}
	if !g.isPlayerControlled(n) {
		if g.distance(n, g.partyGravityX, g.partyGravityY) >= opponentRange {
			g.subMove(n, waitMovePoints)
			return
		}
		if n == g.enemyFront && g.rand(0, 7) == 0 {
			g.TryMoveTo(n, g.partyGravityX, g.partyGravityY)
			return
		}
	}
	g.assault(n)
}

// rear stays behind the front line.
// C: seg_2337:34C9 (COMBAT_AI_Rear)
func (g *Game) rear(n int) {
	var left, right, ox, oy, deltaX, deltaY int
	if g.isPlayerControlled(n) {
		if g.enemyCount == 0 {
			g.TryMoveTo(n, int(g.MapX), int(g.MapY))
			return
		}
		left, right = g.partyLeft, g.partyRight
		ox, oy = g.partyGravityX, g.partyGravityY
		deltaX, deltaY = g.enemyGravityX-g.partyGravityX, g.enemyGravityY-g.partyGravityY
	} else {
		if g.enemyCount == 0 {
			g.subMove(n, waitMovePoints)
			return
		}
		left, right = g.enemyRight, g.enemyLeft
		if left < 0 {
			left = n
		}
		if right < 0 {
			right = n
		}
		ox, oy = g.enemyGravityX, g.enemyGravityY
		deltaX, deltaY = g.partyGravityX-g.enemyGravityX, g.partyGravityY-g.enemyGravityY
	}
	frontDepth := 1<<15 - 1
	align := g.alignment(n)
	for i := range npcCount {
		if !g.NPCs[i].Has(Dead) && g.NPCs[i].Mode == AIFront && g.alignment(i) == align {
			q := g.object(i).Pos
			frontDepth = min(frontDepth, (q.X-ox)*deltaX+(q.Y-oy)*deltaY)
		}
	}
	p := g.object(n).Pos
	depth := (p.X-ox)*deltaX + (p.Y-oy)*deltaY
	points := g.NPCs[n].MovePoints
	var stuck bool
	cross := func(i int) int { q := g.object(i).Pos; return (q.X-ox)*deltaY - (q.Y-oy)*deltaX }
	switch {
	case g.isPlayerControlled(n) && g.distance(n, int(g.MapX), int(g.MapY)) > 3:
		stuck = !g.TryMoveTo(n, int(g.MapX), int(g.MapY))
	case depth >= frontDepth:
		stuck = !g.TryMoveTo(n, p.X-deltaX, p.Y-deltaY)
	default:
		c := (p.X-ox)*deltaY - (p.Y-oy)*deltaX
		switch {
		case cross(left) < c:
			stuck = !g.TryMoveTo(n, p.X-deltaY, p.Y+deltaX)
		case cross(right) > c:
			stuck = !g.TryMoveTo(n, p.X+deltaY, p.Y-deltaX)
		default:
			stuck = true
		}
	}
	if stuck {
		g.NPCs[n].MovePoints = points
		g.assault(n)
	}
}

// fleeing says whether mode i keeps away from the fight.
func fleeing(m AIMode) bool { return m == AIRetreat || m == AIShy || m == AIFlee }

// onScreen says whether (x, y) is in the map view.
func (g *Game) onScreen(x, y int) bool {
	return closeEnough(viewCenter, x, y, int(g.MapX), int(g.MapY))
}

// flank attacks the enemy furthest on n's side.
// C: seg_2337:37F2 (COMBAT_AI_Flank)
func (g *Game) flank(n int) {
	player := g.isPlayerControlled(n)
	if g.enemyCount == 0 || !player && g.distance(n, int(g.MapX), int(g.MapY)) > 7 {
		points := g.NPCs[n].MovePoints
		if player && g.distance(n, int(g.MapX), int(g.MapY)) > 2 {
			g.TryMoveTo(n, int(g.MapX), int(g.MapY))
		}
		g.NPCs[n].MovePoints = points
		g.subMove(n, waitMovePoints)
		return
	}
	align := g.alignment(n)
	p := g.object(n).Pos
	deltaX, deltaY := g.enemyGravityX-g.partyGravityX, g.enemyGravityY-g.partyGravityY
	side := (p.X-g.partyGravityX)*deltaY - (p.Y-g.partyGravityY)*deltaX
	right := side > 0
	if side == 0 {
		right = g.rand(0, 1) != 0
	}
	sx, sy := deltaY, -deltaX
	if !right {
		sx, sy = -deltaY, deltaX
	}
	best, target := -1<<15, noObject
	for i := range npcCount {
		if g.object(i).Shape == 0 || g.NPCs[i].Has(Dead) || g.alignment(i) == align || !g.canSee(n, i) {
			continue
		}
		if align == alignGood && !g.attacksParty(i) || align == alignEvil && !g.attacksMonsters(i) {
			continue
		}
		q := g.object(i).Pos
		if fleeing(g.NPCs[i].Mode) && !g.onScreen(q.X, q.Y) {
			continue
		}
		if closeEnough(7, q.X, q.Y, int(g.MapX), int(g.MapY)) || closeEnough(5, q.X, q.Y, p.X, p.Y) {
			c := (q.X-g.partyGravityX)*deltaY - (q.Y-g.partyGravityY)*deltaX
			if !right {
				c = -c
			}
			if c > best {
				best, target = c, i
			}
		}
	}
	if target < 0 {
		g.wander(n)
		return
	}
	g.Selection.obj = target
	q := g.object(target).Pos
	tx, ty := q.X, q.Y
	w := g.pickWeapon(n, target)
	rng := g.weaponRange(w)
	step := true
	if inReach(tx, ty, p.X, p.Y, rng) {
		step = g.fireAt(n, w, rng, &tx, &ty)
	} else {
		tx += sign(sx)
		ty += sign(sy)
	}
	if step {
		points := g.NPCs[n].MovePoints
		if !g.TryMoveTo(n, tx, ty) {
			g.NPCs[n].MovePoints = points
			g.assault(n)
		}
	}
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// berserk attacks the strongest enemy in sight.
// C: seg_2337:3C3A (COMBAT_AI_Bersek)
func (g *Game) berserk(n int) {
	target, best := noObject, 0
	align := g.alignment(n)
	p := g.object(n).Pos
	for i := range npcCount {
		if g.object(i).Shape == 0 || g.NPCs[i].Has(Dead) || i == n || g.distance(i, p.X, p.Y) > opponentRange || !g.canSee(n, i) {
			continue
		}
		other := g.alignment(i)
		if align == alignGood && other != alignEvil || align == alignEvil && other != alignGood ||
			align == alignChaotic && (other == alignChaotic || other == alignNeutral) {
			continue
		}
		if s := int(g.NPCs[i].Strength) + int(g.NPCs[i].Dexterity) + int(g.NPCs[i].Intelligence); s > best {
			best, target = s, i
		}
	}
	if target < 0 {
		g.assault(n)
		g.subMove(n, waitMovePoints)
		return
	}
	q := g.object(target).Pos
	tx, ty := q.X, q.Y
	w := g.pickWeapon(n, target)
	rng := g.weaponRange(w)
	step := true
	if inReach(tx, ty, p.X, p.Y, rng) {
		g.Selection.obj = target
		step = g.fireAt(n, w, rng, &tx, &ty)
	}
	if step {
		points := g.NPCs[n].MovePoints
		if !g.TryMoveTo(n, tx, ty) {
			g.NPCs[n].MovePoints = points
			g.assault(n)
		}
	}
	g.subMove(n, waitMovePoints)
}

// retreat runs away from the nearest opponent and slowly heals.
// C: seg_2337:3ED2 (COMBAT_AI_Retreat)
func (g *Game) retreat(n int) {
	npc := &g.NPCs[n]
	if g.distance(n, int(g.MapX), int(g.MapY)) > 7 {
		npc.MovePoints = 0
		return
	}
	o := g.pickOpponent(n)
	if o < 0 {
		g.wander(n)
		return
	}
	p, q := g.object(n).Pos, g.object(o).Pos
	deltaX, deltaY := q.X-p.X, q.Y-p.Y
	keep := 3
	if npc.Mode == AIRetreat {
		keep = opponentRange
	}
	switch {
	case closeEnough(keep-1, deltaX, deltaY, 0, 0):
		points := npc.MovePoints
		if !g.TryMoveTo(n, p.X-deltaX, p.Y-deltaY) {
			npc.MovePoints = points
			if npc.Mode != AIFlee && npc.Mode != AIShy {
				g.assault(n)
			}
		}
	case g.alignment(n) == alignGood && npc.Mode != AIShy && npc.Mode != AIFlee:
		g.TryMoveTo(n, g.partyGravityX, g.partyGravityY)
	default:
		g.wander(n)
	}
	if npc.Mode == AIRetreat && g.rand(0, 3) == 0 {
		if g.maxHitPoints(n) > int(npc.HitPoints) {
			npc.HitPoints++
		}
		if int(npc.HitPoints)*4/g.maxHitPoints(n) > 0 {
			npc.Mode = npc.CombatMode
		}
	}
}

// nearParty says whether a party member is next to (x, y) or one further.
func (g *Game) nearParty(x, y int) bool {
	for i := range int(g.PartySize) {
		q := g.object(int(g.Party[i])).Pos
		if closeEnough(2, q.X, q.Y, x, y) {
			return true
		}
	}
	return false
}

// stroll walks n ten tiles in a random direction half of the time.
func (g *Game) stroll(n int) {
	p := g.object(n).Pos
	x, y := p.X, p.Y
	if g.rand(0, 1) != 0 {
		const far = 10
		d := far
		if g.rand(0, 1) == 0 {
			d = -far
		}
		if g.rand(0, 1) != 0 {
			x += d
		} else {
			y += d
		}
		g.TryMoveTo(n, x, y)
		return
	}
	g.subMove(n, waitMovePoints)
}

// like follows the party around.
// C: seg_2337:407F (COMBAT_AI_Like)
func (g *Game) like(n int) {
	p := g.object(n).Pos
	if g.nearParty(p.X, p.Y) && g.rand(0, 1) == 0 {
		g.TryMoveTo(n, g.partyGravityX, g.partyGravityY)
		return
	}
	g.stroll(n)
}

// unfriendly sometimes attacks the party when it comes near.
// C: seg_2337:41B7 (COMBAT_AI_Unfriendly)
func (g *Game) unfriendly(n int) {
	p := g.object(n).Pos
	if g.nearParty(p.X, p.Y) && g.rand(0, 7) == 0 {
		g.NPCs[n].Mode = AIAssault
		g.assault(n)
		return
	}
	g.stroll(n)
}

// immobile attacks what comes near without moving.
// C: seg_2337:42EE (COMBAT_AI_Immobile)
func (g *Game) immobile(n int) {
	align := g.alignment(n)
	if align != alignNeutral {
		p := g.object(n).Pos
		for range 16 {
			x, y := p.X+g.rand(0, 8)-4, p.Y+g.rand(0, 8)-4
			s := g.objectAtForSelection(n, x, y)
			g.Selection.obj = s
			if s >= 0 && s < npcCount && g.canSee(n, s) && g.alignment(s) != align && g.alignment(s) != alignNeutral {
				g.tryHit(n, x, y, g.pickWeapon(n, s))
				g.subMove(n, attackMovePoints)
				return
			}
		}
	}
	g.subMove(n, waitMovePoints)
}

// vineFrames are the frames of a tangle vine part by its direction, two per direction;
// the original reads past a pair into the next (D_1BB8).
var vineFrames = [...]byte{1, 3, 1, 4, 5, 0, 4, 0, 1, 2, 1, 5, 2, 0, 3, 0}

// growTangle grows tangle vine n one step towards dir.
// C: seg_2337:43D2 (__TryGrowVine)
func (g *Game) growTangle(n, dir int) bool {
	dir &= 6
	p := g.object(n).Pos
	prev := g.NPCs[n].Direction()
	if dir^prev == 4 {
		return false
	}
	if !g.TryStraightMove(n, dir, true) {
		return false
	}
	if part := g.AddMapObj(shapeOf(ObjTangleVine2, 0), StatusLocal, uint16(qualQuan(n, 0)), p.X, p.Y, int(g.MapZ)); part >= 0 {
		g.object(part).SetFrame(int(vineFrames[prev*2+dir>>1]))
		g.object(n).SetFrame(int(vineFrames[dir*2+g.rand(0, 3)]))
		return true
	}
	g.MoveObj(n, p.X, p.Y, int(g.MapZ))
	g.NPCs[n].SetDirection(prev)
	return false
}

// tangle grows a tangle vine towards its prey and strikes.
// C: seg_2337:44F6 (COMBAT_AI_Tangle)
func (g *Game) tangle(n int) {
	g.Selection.obj = g.pickOpponent(n)
	if g.Selection.obj >= 0 {
		q, p := g.object(g.Selection.obj).Pos, g.object(n).Pos
		if closeEnough(meleeRange-1, q.X, q.Y, p.X, p.Y) && g.rand(0, 1) != 0 {
			g.tryHit(n, q.X, q.Y, n)
			g.subMove(n, attackMovePoints)
			return
		}
		if closeEnough(4, q.X, q.Y, p.X, p.Y) && g.rand(0, 3) == 0 {
			deltaX, deltaY := q.X-p.X, q.Y-p.Y
			h, v := dirWest, dirNorth
			if deltaX > 0 {
				h = dirEast
			}
			if deltaY > 0 {
				v = dirSouth
			}
			if abs(deltaX) <= abs(deltaY) {
				h, v = v, h
			}
			if !g.growTangle(n, h) {
				g.growTangle(n, v)
				return
			}
		}
	}
	if g.rand(0, 3) == 0 {
		dir := g.rand(0, 3) << 1
		if dir == g.NPCs[n].Direction() {
			dir ^= 4
		}
		g.growTangle(n, dir)
	} else {
		g.NPCs[n].MovePoints = 0
	}
}

// explosionStepsX, explosionStepsY are where the sixteen flames of an explosion fly
// (D_2BDA, D_2BEA).
var (
	explosionStepsX = [maxMissiles]int{2, 1, 0, 1, -1, -1, -2, -1, 0, -1, -2, -2, 2, 2, 1, 1}
	explosionStepsY = [maxMissiles]int{-1, 2, -2, -1, 2, -1, 0, 1, 2, -2, -1, 1, 1, 0, 1, -2}
)

// Explosion tiles: a bomb (TIL_17E), gas (TIL_17C), fire (TIL_189).
const (
	tileBombBlast = 382
	tileGasCloud  = 380
)

// explosion blows up (x, y) with flames of tile; it returns the characters hit.
// C: seg_2FC1:10D2 (Explosion)
func (g *Game) explosion(x, y, tile int) []int {
	var deltaX, deltaY [maxMissiles]int
	for i := range deltaX {
		deltaX[i], deltaY[i] = explosionStepsX[i]+x, explosionStepsY[i]+y
	}
	const blastRange = 3
	g.missiles(x, y, deltaX[:], deltaY[:], blastRange, tile, missileAltSound|missileTrail)
	for i := g.FindLoc(x, y, int(g.MapZ)); i >= 0 && len(g.hitList) <= maxHits; i = g.NextLoc() {
		if i < npcCount && g.object(i).Shape != shapeOf(ObjCorpser, 0) && !g.draggedUnder(i) {
			g.hitList = append(g.hitList, i)
		}
		if g.isNoShoot(g.tileFrame(i) - g.locPart) {
			break
		}
	}
	return g.hitList
}

// afterExplosion sets off the kegs and blows the doors around (x, y).
// C: seg_27A1:3BE1 (AfterExplosion)
func (g *Game) afterExplosion(x, y int) {
	z := int(g.MapZ)
	const openFrames = 12
	for i := x - 2; i <= x+2; i++ {
		// the original stops a row short
		for j := y - 2; j < y+2; j++ {
			if keg := g.typeAt(i, j, z, ObjPowderKeg); keg >= 0 {
				g.object(keg).SetFrame(1)
				g.computeView()
				g.OtherAnimations()
				g.detonate(keg)
			}
			door := noObject
			for _, t := range []ObjectType{ObjSteelDoor, ObjOakenDoor, ObjWindowedDoor, ObjCedarDoor} {
				if door == noObject || g.object(door).Frame()&openFrames == openFrames {
					door = g.typeAt(i, j, z, t)
				}
			}
			if door >= 0 && g.object(door).Frame()&openFrames != openFrames {
				g.DeleteObj(door)
				g.computeView()
				g.OtherAnimations()
				g.printf("\nThe door is blown up!\n")
			}
		}
	}
}

// detonate blows up a powder keg.
// C: seg_27A1:3D2B (Detonate)
func (g *Game) detonate(n int) {
	at := n
	for g.object(at).CoordUse() != OnMap {
		at = g.object(at).Pos.Owner()
	}
	x, y := g.object(at).Pos.X, g.object(at).Pos.Y
	g.DeleteObj(n)
	const blastDamage = 60
	for _, i := range append([]int(nil), g.explosion(x, y, tileFireBolt)...) {
		if quantityType(g.object(i).Type()) == countSingle {
			g.loseHitPoints(i, g.rand(1, blastDamage))
		}
	}
	g.afterExplosion(x, y)
	g.computeView()
	g.OtherAnimations()
}

// Damaging tiles.
const (
	tileWeb         = 564  // TIL_234
	tilePoisonField = 1165 // TIL_48D
	tileFireField   = 1164 // TIL_48C
	tileSleepField  = 1167 // TIL_48F
	tileTangleFirst = 1458 // TIL_5B2..TIL_5B7
	tileTangleLast  = 1463
	tileSpikes      = 562  // TIL_232
	tileSpikes2     = 731  // TIL_2DB
	tileLava        = 732  // TIL_2DC
	tileLava2       = 1010 // TIL_3F2
	tileSwampFirst  = 2
	tileSwampLast   = 5
	minorDamage     = 8
	majorDamage     = 30
)

// fireTiles burn whoever stands on them (TIL_0DC..TIL_0DF, TIL_37A, TIL_464, TIL_465,
// TIL_46A, TIL_46B, TIL_48C, TIL_4A9).
var fireTiles = map[int]bool{220: true, 221: true, 222: true, 223: true, 890: true, 1124: true,
	1125: true, 1130: true, 1131: true, tileFireField: true, 1193: true}

// terrainDamage hurts a character on a web, swamp, fire or field.
// C: seg_0A33:0D1D (TerrainDamage)
func (g *Game) terrainDamage(n, tile int) {
	npc := &g.NPCs[n]
	npc.Flags |= flagSkipTest
	if npc.Has(Dead) {
		return
	}
	cls := monsterClass(g.object(n).Type())
	protected := npc.Has(Protected)
	poison := func() {
		if protected || classHas(cls, immunePoison) || npc.Has(Poisoned) {
			return
		}
		g.printf(poisonedMsg, g.objectString(n))
		g.promptFlag = true
		g.updateViewNow()
		g.showHit(n)
		npc.Status |= Poisoned
		g.StatusDirty = true
	}
	hurt := func(most int) {
		if protected {
			return
		}
		g.updateViewNow()
		g.loseHitPoints(n, g.rand(1, most))
	}
	switch {
	case tile == tileWeb:
		const odds = 30
		if !protected && g.strengthOf(n) < g.rand(1, odds) && g.object(n).Type() != ObjGiantSpider {
			npc.Status |= Paralyzed
			g.printf("`%s is stuck in a web!\n", g.objectString(n))
			g.promptFlag = true
		}
	case tile >= tileSwampFirst && tile <= tileSwampLast:
		for b := g.FindInvType(n, ObjSwampBoots, -1); b >= 0; b = g.NextInvType() {
			if g.object(b).CoordUse() == Equipped {
				return
			}
		}
		poison()
	case tile == tilePoisonField:
		poison()
	case fireTiles[tile]:
		if protected || classHas(cls, immuneFire) {
			return
		}
		dmg := g.rand(1, minorDamage)
		if tile == tileFireField {
			dmg = g.rand(1, majorDamage)
		}
		if classHas(cls, doubleFireDamage) {
			dmg <<= 1
		}
		g.updateViewNow()
		g.loseHitPoints(n, dmg)
	case tile >= tileTangleFirst && tile <= tileTangleLast:
		if g.object(n).Type() != ObjTangleVine2 {
			hurt(minorDamage)
		}
	case tile == tileSpikes || tile == tileSpikes2:
		hurt(minorDamage)
	case tile == tileLava || tile == tileLava2:
		hurt(majorDamage)
	case tile == tileSleepField:
		if classHas(cls, immuneSleep) || protected || npc.Has(Asleep) {
			return
		}
		g.updateViewNow()
		g.showHit(n)
		npc.Status |= Asleep
		g.setUnconscious(n)
	}
}

// castInCombat casts the spell of the charge or spellbook w at opponent for the monster a.
// C: seg_2337 (inside C_2337_0D8D, the spell branch)
func (g *Game) castInCombat(a, opponent, w int) int {
	spell := g.object(w).Quality()
	if spell == castWispTeleport && g.object(a).Type() == ObjWisp {
		x, y := g.object(a).Pos.X, g.object(a).Pos.Y
		g.MoveObj(a, int(g.MapX), int(g.MapY), int(g.MapZ))
		if g.tryTeleport(a, false) {
			g.printf(wispTeleportMsg)
			g.promptFlag = true
		} else {
			g.MoveObj(a, x, y, int(g.MapZ))
		}
		return hitDone
	}
	if g.object(w).Type() == ObjSpellbook {
		cost := circleOf(spell)
		if int(g.NPCs[a].Magic) < cost || int(g.NPCs[a].Level) < cost {
			return hitDone
		}
		g.NPCs[a].Magic -= byte(cost)
	}
	active := g.Active
	g.Active = partyMax
	g.npcCaster = a
	g.Selection.obj = opponent
	if opponent >= 0 { // fix: C reads the position of object -1 when the spell aims at nothing
		g.Selection.x, g.Selection.y = g.object(opponent).Pos.X, g.object(opponent).Pos.Y
	}
	g.TextOn = false
	g.castSpell(spell, true)
	g.TextOn = true
	if spell == castCharm && g.spellResult == resultHit {
		g.printf("%s is charmed.\n", g.objectString(g.Selection.obj))
	} else if spell == castParalyze && g.spellResult == resultSuccess {
		g.printf("%s is paralyzed.\n", g.objectString(g.Selection.obj))
	}
	g.Active = active
	return hitDone
}
