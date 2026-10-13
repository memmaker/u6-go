package game

// Eggs: the hidden objects that create creatures and objects when the party comes near
// (seg_2E2D).

// The stats of each monster class (D_3522_0082, _00C2, _0102, _01C2, _0202).
var (
	classStrength = [...]byte{
		5, 20, 16, 25, 5, 8, 1, 8, 17, 14, 27, 35, 11, 5, 40, 22, 10, 15, 20, 15, 25, 2, 1, 2, 30, 19, 22, 22, 26, 1, 14, 30,
		12, 10, 12, 22, 20, 2, 3, 5, 21, 3, 13, 17, 5, 30, 60, 10, 6, 3, 9, 24, 15, 20, 18, 11, 1, 8, 20, 20, 10, 10, 4, 11,
	}
	classDexterity = [...]byte{
		5, 10, 15, 25, 25, 8, 28, 8, 6, 7, 8, 26, 18, 16, 35, 22, 14, 15, 17, 32, 19, 10, 10, 26, 30, 13, 27, 27, 11, 40, 16, 30,
		16, 14, 12, 9, 27, 25, 9, 13, 20, 17, 13, 17, 10, 30, 21, 14, 6, 7, 11, 20, 18, 25, 14, 12, 40, 16, 5, 20, 10, 10, 18, 12,
	}
	classIntelligence = [...]byte{
		2, 3, 2, 25, 4, 9, 4, 9, 3, 5, 7, 31, 5, 8, 27, 13, 15, 10, 11, 33, 8, 21, 9, 7, 9, 8, 7, 7, 2, 1, 12, 30,
		14, 22, 18, 8, 13, 3, 3, 4, 16, 2, 3, 9, 4, 30, 8, 7, 2, 6, 4, 8, 12, 1, 9, 12, 20, 8, 1, 20, 10, 10, 8, 12,
	}
	classHitPoints = [...]byte{
		10, 30, 10, 30, 5, 6, 3, 6, 40, 10, 40, 100, 8, 6, 150, 50, 8, 8, 20, 50, 30, 20, 20, 5, 100, 20, 15, 15, 50, 1, 8, 255,
		8, 30, 8, 30, 30, 5, 5, 8, 30, 5, 20, 70, 6, 30, 200, 20, 15, 5, 10, 50, 15, 30, 15, 8, 40, 8, 30, 20, 10, 10, 6, 8,
	}
)

// classAlignment: N neutral, E evil, G good, C chaotic.
var classAlignment = func() (a [64]NPCStatus) {
	for i, r := range "CCCGCCNCCNEENNEENNCEEEECEENNCCCGCEECENNCCCCENCCECCCECCECCCCCCCNC" {
		a[i] = map[rune]NPCStatus{'N': alignNeutral, 'E': alignEvil, 'G': alignGood, 'C': alignChaotic}[r]
	}
	return
}()

// possessions are what the creatures of a class carry: spells (charges), weapons, armour
// and treasure; each further one of a list is half as likely (D_3522_02C2, D_3522_0342).
type possessions struct {
	spells                    []int
	weapons, armour, treasure []ObjectType
}

var classPossessions = map[int]possessions{
	5:  {spells: nil, weapons: []ObjectType{ObjDagger, ObjClub}, armour: []ObjectType{ObjCloth}, treasure: nil},
	9:  {spells: nil, weapons: nil, armour: nil, treasure: []ObjectType{ObjPortionOfMeat}},
	10: {spells: nil, weapons: []ObjectType{ObjTwoHandedHammer, ObjSling, ObjClub}, armour: []ObjectType{ObjScaleMail, ObjIronHelm, ObjSpikedShield}, treasure: []ObjectType{ObjOpenChest}},
	11: {spells: []int{17, 128, 34, 80}, weapons: nil, armour: nil, treasure: nil},
	12: {spells: nil, weapons: nil, armour: nil, treasure: []ObjectType{ObjPortionOfMeat}},
	14: {spells: []int{34, 83, 68, 96, 119}, weapons: nil, armour: nil, treasure: []ObjectType{ObjOpenChest, ObjGold, ObjGold}},
	15: {spells: []int{34, 68}, weapons: nil, armour: nil, treasure: []ObjectType{ObjGold, ObjGold}},
	16: {spells: nil, weapons: []ObjectType{ObjDagger}, armour: []ObjectType{ObjCloth}, treasure: []ObjectType{ObjGold}},
	17: {spells: nil, weapons: []ObjectType{ObjHoe, ObjPitchfork}, armour: []ObjectType{ObjClothArmor}, treasure: []ObjectType{ObjGold}},
	18: {spells: nil, weapons: []ObjectType{ObjSword, ObjCrossbow}, armour: []ObjectType{ObjChainMail, ObjCurvedHeater, ObjIronHelm, ObjPlateMail}, treasure: []ObjectType{ObjGold, ObjGold}},
	19: {spells: []int{68, 65, 69}, weapons: []ObjectType{ObjTwoHandedSword, ObjBow, ObjHalberd, ObjBoomerang}, armour: []ObjectType{ObjLeatherArmor, ObjLeatherHelm}, treasure: []ObjectType{ObjSpell1}},
	20: {spells: nil, weapons: []ObjectType{ObjClub, ObjBoomerang, ObjThrowingAxe}, armour: []ObjectType{ObjLeatherHelm, ObjWoodenShield, ObjLeatherArmor, ObjScaleMail}, treasure: nil},
	21: {spells: []int{20, 68, 80}, weapons: nil, armour: nil, treasure: nil},
	22: {spells: []int{128, 99}, weapons: nil, armour: nil, treasure: nil},
	24: {spells: nil, weapons: []ObjectType{ObjCrossbow, ObjHalberd}, armour: []ObjectType{ObjPlateMail, ObjCurvedHeater, ObjIronHelm}, treasure: []ObjectType{ObjGold}},
	25: {spells: nil, weapons: []ObjectType{ObjClub, ObjSpear}, armour: []ObjectType{ObjClothArmor, ObjWoodenShield}, treasure: []ObjectType{ObjGold}},
	28: {spells: []int{69, 96, 98, 68}, weapons: nil, armour: nil, treasure: nil},
	30: {spells: nil, weapons: []ObjectType{ObjDagger, ObjSling}, armour: []ObjectType{ObjClothArmor}, treasure: []ObjectType{ObjGold}},
	32: {spells: nil, weapons: []ObjectType{ObjBow, ObjSling, ObjSword}, armour: []ObjectType{ObjRingMail, ObjLeatherHelm, ObjWoodenShield}, treasure: []ObjectType{ObjGold, ObjLute}},
	33: {spells: []int{17, 34, 20, 49, 68, 80, 69, 83}, weapons: []ObjectType{ObjDagger}, armour: []ObjectType{ObjClothArmor, ObjLeatherHelm}, treasure: []ObjectType{ObjSpell1, ObjOpenChest, ObjGold}},
	34: {spells: nil, weapons: []ObjectType{ObjDagger, ObjSword}, armour: []ObjectType{ObjClothArmor, ObjLeatherHelm}, treasure: []ObjectType{ObjGold}},
	35: {spells: []int{18, 80}, weapons: nil, armour: nil, treasure: []ObjectType{ObjOpenChest, ObjOpenChest}},
	36: {spells: nil, weapons: []ObjectType{ObjDagger, ObjSword, ObjSpear, ObjThrowingAxe}, armour: []ObjectType{ObjWoodenShield, ObjClothArmor, ObjLeatherArmor}, treasure: []ObjectType{ObjOpenChest, ObjGold}},
	40: {spells: []int{17, 80, 66, 68}, weapons: nil, armour: nil, treasure: []ObjectType{ObjSpell1, ObjGold}},
	43: {spells: []int{34, 83, 51}, weapons: nil, armour: nil, treasure: nil},
	44: {spells: nil, weapons: nil, armour: nil, treasure: []ObjectType{ObjPortionOfMeat}},
	47: {spells: nil, weapons: []ObjectType{ObjSword, ObjSpear, ObjThrowingAxe, ObjBow, ObjDagger}, armour: []ObjectType{ObjWoodenShield, ObjLeatherHelm}, treasure: []ObjectType{ObjGold}},
	50: {spells: []int{89}, weapons: nil, armour: nil, treasure: []ObjectType{ObjPortionOfMeat}},
	51: {spells: []int{17, 68}, weapons: nil, armour: nil, treasure: nil},
	52: {spells: nil, weapons: []ObjectType{ObjSword, ObjBow, ObjSling}, armour: []ObjectType{ObjRingMail, ObjLeatherHelm, ObjWoodenShield}, treasure: []ObjectType{ObjGold, ObjGold}},
	54: {spells: nil, weapons: []ObjectType{ObjThrowingAxe, ObjSpear, ObjSword, ObjBow, ObjClub}, armour: []ObjectType{ObjWoodenShield, ObjLeatherArmor, ObjLeatherHelm, ObjRingMail}, treasure: []ObjectType{ObjOpenChest, ObjGold, ObjGold}},
	55: {spells: nil, weapons: []ObjectType{ObjSword, ObjSling}, armour: []ObjectType{ObjClothArmor}, treasure: []ObjectType{ObjGold}},
	56: {spells: []int{20, 69, 129}, weapons: nil, armour: nil, treasure: nil},
	58: {spells: []int{20}, weapons: nil, armour: nil, treasure: nil}}

// Classes whose chests are never trapped and that carry no gold with their scrolls.
const (
	classAcidSlug       = 0
	classWingedGargoyle = 19
	trapSpell           = 16 // the Effect quality of a trapped container (SPELL_16)
	chestGoldMax        = 100
)

// food a creature may carry (D_28A3).
var creatureFood = [...]ObjectType{ObjPortionOfMeat, ObjHam, ObjLoafOfBread}

// randomAround returns about v, between v-v/2 and v+v/2.
// C: seg_2E2D:0088 (mkRandom)
func (g *Game) randomAround(v int) int {
	if half := v >> 1; half != 0 {
		v = g.rand(0, half) + v - half + g.rand(0, half)
	}
	return v
}

// equipCreature gives a new creature the possessions of its class.
// C: seg_2E2D:00BE (C_2E2D_00BE)
func (g *Game) equipCreature(n, cls int) {
	p := classPossessions[cls]
	const kept = StatusLocal | StatusOwned
	for _, s := range p.spells {
		if i := g.AddInvObj(shapeOf(ObjCharge, 0), StatusLocal|Equipped, uint16(qualQuan(s, 1)), n); i >= 0 {
			g.object(i).Status |= StatusInvisible
		}
	}
	odds := 2
	for _, t := range p.weapons {
		if g.rand(1, odds) == 1 {
			count := 1
			if t == ObjSpear || t == ObjThrowingAxe || t == ObjDagger {
				count = g.rand(6, 12)
			}
			for range count {
				if g.AddInvObj(shapeOf(t, 0), kept|Equipped, 0, n) < 0 {
					continue
				}
				const ammoMin, ammoMax = 12, 24
				switch t {
				case ObjBow, ObjMagicBow:
					g.AddInvObj(shapeOf(ObjArrow, 0), kept|InInventory, uint16(g.rand(ammoMin, ammoMax)), n)
				case ObjCrossbow, ObjTripleCrossbow:
					g.AddInvObj(shapeOf(ObjBolt, 0), kept|InInventory, uint16(g.rand(ammoMin, ammoMax)), n)
				}
			}
		}
		odds <<= 1
	}
	odds = 2
	for _, t := range p.armour {
		if g.rand(1, odds) == 1 {
			g.AddInvObj(shapeOf(t, 0), kept|Equipped, 1, n)
		}
		odds <<= 1
	}
	odds = 2
	for _, t := range p.treasure {
		if g.rand(1, odds) == 1 {
			g.giveTreasure(n, cls, t)
		}
		odds <<= 1
	}
}

// giveTreasure gives creature n one treasure t of its class.
func (g *Game) giveTreasure(n, cls int, t ObjectType) {
	const kept = StatusLocal | StatusOwned
	add := func(t ObjectType, frame int, where ObjStatus, amount, owner int) int {
		return g.AddInvObj(shapeOf(t, frame), kept|where, uint16(amount), owner)
	}
	const foodMax, torchesMax, gemsMax, potionColours = 10, 6, 4, 7
	switch t {
	case ObjOpenChest:
		chest := add(t, 0, Contained, g.rand(1, chestGoldMax), n)
		if chest < 0 {
			return
		}
		g.object(chest).SetFrame(1)
		if g.rand(0, 1) != 0 && cls != classAcidSlug {
			add(ObjEffect, 0, Contained, qualQuan(trapSpell, 0), chest)
		}
		if g.rand(0, 3) != 0 {
			add(ObjGold, 0, Contained, g.rand(1, chestGoldMax), chest)
		}
		if g.rand(0, 3) != 0 {
			add(creatureFood[g.rand(0, len(creatureFood)-1)], 0, Contained, g.rand(1, foodMax), chest)
		}
		if g.rand(0, 3) == 0 {
			add(ObjTorch, 0, Contained, g.rand(1, torchesMax), chest)
		}
		if g.rand(0, 10) == 0 {
			add(ObjGem, 0, Contained, g.rand(1, gemsMax), chest)
		}
	case ObjSpell1:
		if g.rand(0, 3) != 0 && cls != classWingedGargoyle {
			add(ObjGold, 0, InInventory, g.rand(1, chestGoldMax), n)
		}
		if g.rand(0, 3) == 0 {
			add(creatureFood[g.rand(0, len(creatureFood)-1)], 0, InInventory, g.rand(1, foodMax), n)
		}
		if g.rand(0, 3) == 0 {
			add(ObjBluePotion, g.rand(0, potionColours), InInventory, 1, n)
		}
	case ObjGold:
		const goldDie = 30
		add(t, 0, InInventory, g.rand(1, goldDie)+g.rand(1, goldDie), n)
	default:
		add(t, 0, InInventory, 1, n)
	}
}

// mutantFlag marks a two-headed creature in the type given to generate.
const mutantFlag = 1 << 15

// generate creates a creature of type t (with mutantFlag, two-headed) at (x, y, z).
// C: seg_2E2D:0499 (EGG_generate)
func (g *Game) generate(t int, x, y, z int) int {
	mutant := t&mutantFlag != 0
	typ := ObjectType(t &^ mutantFlag)
	n := g.AddMonster(typ, x, y, z)
	if n < 0 {
		return n
	}
	npc := &g.NPCs[n]
	cls := monsterClass(typ)
	if cls >= 0 {
		npc.Strength = byte(g.randomAround(int(classStrength[cls])))
		npc.Dexterity = byte(g.randomAround(int(classDexterity[cls])))
		npc.Intelligence = byte(g.randomAround(int(classIntelligence[cls])))
		npc.HitPoints = byte(g.randomAround(int(classHitPoints[cls])))
		const hitPointsPerLevel = 30
		npc.Level = byte((int(npc.HitPoints) + hitPointsPerLevel - 1) / hitPointsPerLevel)
		npc.Status = classAlignment[cls]
	} else {
		const plain = 10
		npc.Strength, npc.Dexterity, npc.Intelligence = plain, plain, plain
		npc.HitPoints, npc.Level, npc.Status = 30, 1, alignEvil
	}
	npc.Magic = byte(g.maxMagic(n))
	npc.MovePoints = int8(npc.Dexterity)
	npc.Experience = 100
	npc.Mode, npc.CombatMode = AIAssault, AIAssault
	part := func(shape uint16, x, y int) {
		if i := g.AddMapObj(shape, StatusOwned, 0, x, y, z); i >= 0 {
			g.object(i).Status &^= StatusLocal
		}
	}
	o := g.object(n)
	switch {
	case typ == ObjWingedGargoyle || typ == ObjCyclops:
		const standingFrame = 19
		o.SetFrame(standingFrame)
	case typ >= ObjGiantScorpion:
		const headFrame, tailFrame, mutantTail = 6, 14, 2
		o.SetFrame(headFrame)
		if mutant {
			part(shapeOf(typ, mutantTail), x+1, y)
		} else {
			part(shapeOf(typ, tailFrame), x+1, y)
		}
	case typ == ObjDragon:
		const head, tail, leftWing, rightWing = 8, 16, 24, 32
		o.SetFrame(0)
		part(shapeOf(typ, head), x, y-1)
		part(shapeOf(typ, tail), x, y+1)
		part(shapeOf(typ, leftWing), x-1, y)
		part(shapeOf(typ, rightWing), x+1, y)
	case typ == ObjHydra:
		const headsFrameShift = 2 // a head's frame is its direction << 2
		for d := range dirStepX {
			part(shapeOf(ObjHydra2, d<<headsFrameShift), x+dirStepX[d], y+dirStepY[d])
		}
	}
	g.equipCreature(n, cls)
	return n
}

// tryTeleport moves n to a free random spot near it, off the view with offScreen.
// C: seg_2337:02DC (COMBAT_TryTeleport)
func (g *Game) tryTeleport(n int, offScreen bool) bool {
	if n < 0 {
		return false
	}
	p := g.object(n).Pos
	const tries, spread = 8, 4
	for range tries {
		x := g.rand(1, spread) + g.rand(1, spread) + p.X - spread - 1
		y := g.rand(1, spread) + g.rand(1, spread) + p.Y - spread - 1
		if offScreen && closeEnough(viewCenter, x, y, int(g.MapX), int(g.MapY)) {
			continue
		}
		if g.passable(n, x, y) {
			g.MoveObj(n, x, y, p.Z)
			return true
		}
	}
	return false
}

// Egg qualities: the tens say when it hatches, the units the alignment + 1.
const (
	eggDay               = 1
	eggNight             = 2
	statusHatched        = StatusCursed // the same bit (HATCHED, MUTANT)
	eggSure              = 100          // a quantity of 100 hatches all the creatures
)

// silver serpent bodies (D_28A9..D_28B5)
var (
	serpentCurves = [4]int{13, 10, 11, 12}
	serpentTails  = [4]int{1, 3, 5, 7}
	serpentStepX  = [4]int{0, -1, -1, 0}
	serpentStepY  = [4]int{1, 1, 0, 0}
)

// hatch creates what egg holds.
// C: seg_2E2D:0760 (EGG_hatches)
func (g *Game) hatch(egg int) {
	if g.armageddon {
		return
	}
	eo := g.object(egg)
	h := int(g.Hour)
	const dawn, dusk = 6, 18
	switch eo.Quality() / 10 {
	case eggDay:
		if h < dawn || h > dusk {
			return
		}
	case eggNight:
		if h > dawn-1 && h < dusk+1 {
			return
		}
	}
	offScreen := !eo.Has(StatusLocal) && !g.forceHatching
	gargoylesGraze := false
	for i := range int(g.PartySize) {
		t := g.object(int(g.Party[i])).Type()
		if t == ObjGargoyle || t == ObjWingedGargoyle {
			gargoylesGraze = true
			break
		}
		if g.FindInvType(avatar, ObjAmuletOfSubmission, -1) >= 0 {
			gargoylesGraze = true
		}
	}
	if !eo.Has(statusHatched) {
		g.hatchCreatures(egg, offScreen, gargoylesGraze)
	}
	eo.Status |= statusHatched | StatusInvisible
}

func (g *Game) hatchCreatures(egg int, offScreen, gargoylesGraze bool) {
	eo := g.object(egg)
	const alignShift = 5
	align := NPCStatus((eo.Quality()%10 - 1) << alignShift)
	if g.rand(1, 100) > eo.Quantity() {
		return
	}
	attacks, first := false, true
	mode := AIMotionless
	for embryo := g.FindInv(egg); embryo >= 0; embryo = g.NextInv() {
		eb := g.object(embryo)
		t := eb.Type()
		param := int(t)
		if eb.Has(statusHatched) {
			param |= mutantFlag
		}
		count := eb.Quantity()
		if eo.Quantity() != eggSure {
			count = g.rand(1, count)
		}
		if t == ObjSilverSerpent {
			count = 1
		}
		if (t == ObjGargoyle || t == ObjWingedGargoyle) && gargoylesGraze {
			mode = AIGraze
		}
		for range count {
			var n int
			if t < ObjGiantRat && t != ObjOpenChest {
				amount := 0
				if quantityType(t) != countSingle {
					amount = 1
				}
				n = g.AddMapObj(shapeOf(t, 0), StatusOwned, uint16(amount), eo.Pos.X, eo.Pos.Y, int(g.MapZ))
			} else if n = g.generate(param, eo.Pos.X, eo.Pos.Y, int(g.MapZ)); n >= 0 {
				if eo.Quality()%10 != 0 {
					g.NPCs[n].Status = g.NPCs[n].Status&^alignmentMask | align
				}
				if g.NPCs[n].Status&alignEvil != 0 && !eo.Has(StatusLocal) {
					attacks = true
				}
			}
			if n < 0 {
				continue
			}
			if !first {
				if !g.tryTeleport(n, offScreen) {
					g.DeleteObj(n)
				}
			} else {
				first = false
			}
			switch t {
			case ObjSilverSerpent:
				g.growSerpent(n, eb.Quantity())
			case ObjTangleVine:
				g.growVine(n)
			}
			if n >= npcCount {
				continue
			}
			if mode == AIMotionless {
				mode = AIMode(eb.Quality())
			}
			npc := &g.NPCs[n]
			npc.Mode, npc.CombatMode = mode, mode
			switch mode {
			case AIGuard:
				npc.SetDirection(2)
			case AIGuard2:
				npc.SetDirection(0)
			}
			if (t == ObjGargoyle || t == ObjWingedGargoyle) && npc.Mode == AIBerserk {
				npc.Strength <<= 1
				npc.Dexterity <<= 1
				npc.Intelligence <<= 1
			}
		}
	}
	if !g.heardSomething && attacks {
		x, y := eo.Pos.X-int(g.MapX), eo.Pos.Y-int(g.MapY)
		d := g.pointerDirection(x*tileSize+g.pointerCentreX, y*tileSize+g.pointerCentreY)
		if g.rand(0, 3) != 0 && g.shaminoNear() {
			g.printf("Shamino says, \"I hear something to the %s!\"\n", dirNames[d])
			g.promptFlag = true
		}
		g.heardSomething = true
	}
}

// growSerpent adds the body of silver serpent n.
func (g *Game) growSerpent(n, length int) {
	g.serpentPart = 0
	o := g.object(n)
	o.SetFrame(0)
	o.setQuality(g.serpentPart)
	g.serpentPart++
	x, y := o.Pos.X, o.Pos.Y
	last := n
	for i := 0; i <= length; i++ {
		part := g.AddMapObj(shapeOf(ObjSilverSerpent, serpentCurves[i&3]), StatusOwned,
			uint16(qualQuan(g.serpentPart, n)), serpentStepX[i&3]+x, serpentStepY[i&3]+y, int(g.MapZ))
		g.serpentPart++
		if part < 0 {
			if i != 0 {
				g.object(last).SetFrame(serpentTails[(i-1)&3])
			} else {
				g.DeleteObj(n)
			}
			return
		}
		g.object(part).Status &^= StatusLocal
		last = part
		if length == i {
			g.object(part).SetFrame(serpentTails[i&3])
		}
	}
}

// growVine adds the four arms of tangle vine n.
func (g *Game) growVine(n int) {
	o := g.object(n)
	o.SetFrame(0)
	x, y := o.Pos.X, o.Pos.Y
	for d := 0; d < len(dirStepX); d += 2 {
		arm := g.generate(int(ObjTangleVine2), x+dirStepX[d], y+dirStepY[d], int(g.MapZ))
		if arm < 0 {
			continue
		}
		g.NPCs[arm].SetDirection(d)
		if d&2 != 0 {
			g.object(arm).SetFrame(0)
		} else {
			g.object(arm).SetFrame(1)
		}
		g.NPCs[arm].Status = g.NPCs[arm].Status&^alignmentMask | g.NPCs[n].Status&alignmentMask
		g.NPCs[arm].Mode, g.NPCs[arm].CombatMode = AITangle, AITangle
	}
}

// hatchEggs hatches the eggs of the area that are far enough (or all, when forced).
// C: seg_2E2D:0DFE (EGG_hatchArea)
func (g *Game) hatchEggs() {
	if g.armageddon {
		return
	}
	g.serpentPart = 0
	const nearEgg = 8
	for n := g.SearchArea(g.AreaX, g.AreaY, g.AreaX+areaSize-1, g.AreaY+areaSize-1); n >= 0; n = g.NextArea() {
		o := g.object(n)
		if o.Type() != ObjEgg || o.Pos.Z != int(g.MapZ) {
			continue
		}
		if g.forceHatching || !closeEnough(nearEgg, o.Pos.X, o.Pos.Y, int(g.MapX), int(g.MapY)) || o.Has(StatusLocal) {
			g.hatch(n)
		}
	}
}

// eggState is what hatching remembers.
type eggState struct {
	// forceHatching hatches every egg of the area (ForceHatching).
	forceHatching bool
	// heardSomething keeps Shamino from warning twice (D_17B6).
	heardSomething bool
	// serpentPart numbers the parts of a silver serpent (D_28A2).
	serpentPart int
}

// shaminoNear says whether Shamino is in the party, alive and near (SHAMINO_COMMENT).
func (g *Game) shaminoNear() bool {
	const shamino, earshot = 3, 6
	return g.isPlayerControlled(shamino) && !g.NPCs[shamino].Has(Dead) &&
		g.distance(shamino, int(g.MapX), int(g.MapY)) < earshot
}
