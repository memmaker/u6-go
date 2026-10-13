package game

import (
	"strings"

	"github.com/memmaker/u6-go/internal/sound"
)

// The use command and its handlers (seg_27A1, its second half), rest (seg_3200) and the
// travel between places (seg_101C).

// usableTypes can be used (D_1C30).
var usableTypes = []ObjectType{
	ObjAle, ObjBalloon2, ObjBalloonPlans, ObjOpenBarrel, ObjBell, ObjBeehive, ObjBrazier, ObjLoafOfBread, ObjBucketOfMilk, ObjButter,
	ObjCake, ObjCampfire, ObjCandle, ObjCandelabra, ObjCannon, ObjCedarDoor, ObjCheese, ObjOpenChest, ObjHorseChops, ObjChurn,
	ObjCow, ObjCrank, ObjOpenCrate, ObjCrystalBall, ObjBalloon, ObjFan, ObjFireplace, ObjFish, ObjFishingPole, ObjFountain,
	ObjGargishVocabulary, ObjGem, ObjBunchOfGrapes, ObjHam, ObjHarp, ObjHarpsichord, ObjHorse, ObjMountedHorse, ObjJarOfHoney, ObjLockpick,
	ObjLadder, ObjLever, ObjLute, ObjMead, ObjMeat, ObjMoonstone, ObjPortionOfMeat, ObjOakenDoor, ObjOrbOfTheMoons, ObjPanpipes,
	ObjPassthrough, ObjPassthrough2, ObjPick, ObjBluePotion, ObjPowderKeg, ObjPullChain, ObjRaft, ObjRibs, ObjRolls, ObjRubberDuckyG,
	ObjRuneOfHonesty, ObjRuneOfCompassion, ObjRuneOfValor, ObjRuneOfJustice, ObjRuneOfSacrifice, ObjRuneOfHonor, ObjRuneOfSpirituality, ObjRuneOfHumility, ObjSecretDoor, ObjSextant,
	ObjShip, ObjShovel, ObjSilverHorn, ObjSkiff, ObjStaff, ObjSteelDoor, ObjSwitch, ObjTelescope, ObjTorch, ObjKey2,
	ObjVortexCube, ObjWell, ObjWindowedDoor, ObjWine, ObjXylophone,
}

// vehicleTypes cannot be used from a ship (D_1D59).
var vehicleTypes = []ObjectType{ObjBalloon2, ObjBalloonPlans, ObjCannon, ObjBalloon, ObjHorse, ObjLadder,
	ObjRaft, ObjShip, ObjSkiff, ObjStaff}

// isDoor (D_0EDA, C_1944_0AA9).
func isDoor(t ObjectType) bool {
	return t == ObjOakenDoor || t == ObjWindowedDoor || t == ObjCedarDoor || t == ObjSteelDoor
}

// The frames of doors and chests.
const (
	doorOpenBit     = 4
	doorLockBit     = 8
	doorLockedBoth  = 12 // locked and magically locked
	doorClosedFirst = 4
	doorLockedFirst = 8
	doorMagicFirst  = 12
	chestOpen       = 0
	chestClosed     = 1
	chestLocked     = 2
)

// vehicleBlocked says whether n cannot be used from the vehicle.
// C: seg_27A1:60F5 (C_27A1_60F5)
func (g *Game) vehicleBlocked(n int) bool {
	t := g.object(n).Type()
	if !hasType(vehicleTypes, t) {
		return false
	}
	actor := int(g.Party[g.Active])
	return !(g.inVehicle() && actor == n || g.object(actor).Type() == ObjSkiff && t == ObjShip)
}

// use uses the selected object.
// C: seg_27A1:6179 (C_27A1_6179)
func (g *Game) use() {
	sel := &g.Selection
	sel.obj = g.combatHead(sel.obj)
	var t ObjectType
	if sel.obj >= 0 {
		t = g.object(sel.obj).Type()
		if sel.obj < npcCount && sel.x != -1 && t != ObjHorse && t != ObjMountedHorse && t != ObjCow && !g.inVehicle() ||
			g.isIgnored(g.tileOf(sel.obj)) {
			sel.obj = g.objectBelow(sel.obj)
		}
	}
	if sel.obj == noObject {
		g.printf("%s\n", nothingMsg)
		return
	}
	sel.obj = g.combatHead(sel.obj)
	if t == ObjMountedHorse {
		g.printf("horse\n")
	} else {
		g.printf("%s", g.objectString(sel.obj))
		g.putch('\n')
	}
	o := g.object(sel.obj)
	t = o.Type()
	if !hasType(usableTypes, t) {
		g.printf("\n%s\n", notUsableMsg)
		return
	}
	if g.inVehicle() && g.vehicleBlocked(sel.obj) {
		g.printf("\n")
		g.printf(notOnShipMsg)
		return
	}
	if o.CoordUse() == OnMap {
		g.turnToPointer()
		p := g.object(int(g.Party[g.Active])).Pos
		if !closeEnough(1, sel.x, sel.y, p.X, p.Y) {
			g.printf("\n%s!\n", outOfRangeMsg)
			return
		}
	}
	under := g.FindLoc(o.Pos.X, o.Pos.Y, int(g.MapZ))
	var underType ObjectType
	if under >= 0 {
		underType = g.object(under).Type()
	}
	if isDoor(t) && !isDoor(underType) && !g.isIgnored(g.tileOf(under)) || t == ObjSecretDoor && underType != ObjSecretDoor {
		g.printf("\nNot now!\n")
		return
	}
	if int(g.Active) != g.statusMember && o.CoordUse() == Equipped && t != ObjTorch {
		g.printf("\nMust be in your hand.\n")
		return
	}
	g.useObject(sel.obj, t)
	g.subMove(int(g.Party[g.Active]), 5)
	g.computeView()
	g.OtherAnimations()
}

// notUsable prints "Not usable".
func (g *Game) notUsable() { g.printf("\n%s\n", notUsableMsg) }

// useObject runs the handler of type t.
func (g *Game) useObject(n int, t ObjectType) {
	o := g.object(n)
	frame := o.Frame()
	switch t {
	case ObjWine, ObjMead, ObjAle:
		g.drink(n)
	case ObjShip, ObjSkiff, ObjRaft, ObjBalloon2:
		if o.CoordUse() != OnMap {
			g.notUsable()
		} else {
			g.useVehicle(n)
		}
	case ObjBalloonPlans:
		g.buildBalloon()
	case ObjOpenBarrel, ObjOpenCrate:
		if frame == 1 {
			o.SetFrame(0)
		} else {
			g.notUsable()
		}
		g.search(n, g.Selection.x, g.Selection.y)
	case ObjBell, ObjPullChain:
		g.ringBell()
	case ObjBeehive:
		g.fill(ObjHoneyJar, ObjJarOfHoney, "\nYou need a honey jar.\n")
	case ObjCandle, ObjCandelabra, ObjFireplace, ObjBrazier, ObjCampfire:
		const brazierCold = 2
		if t == ObjBrazier && frame&brazierCold != 0 {
			g.notUsable()
		} else {
			g.light(n)
		}
	case ObjBunchOfGrapes, ObjButter, ObjLoafOfBread, ObjPortionOfMeat, ObjRolls, ObjCake, ObjCheese, ObjHam,
		ObjHorseChops, ObjBucketOfMilk, ObjJarOfHoney, ObjMeat, ObjRibs, ObjFish:
		g.eat(n)
	case ObjCannon:
		g.fireCannon(n)
	case ObjOakenDoor, ObjWindowedDoor, ObjCedarDoor, ObjSteelDoor:
		g.useDoor(n, false, false, false)
	case ObjOpenChest:
		if o.CoordUse() != OnMap {
			g.notUsable()
		} else {
			g.useChest(n, false, false, false)
		}
	case ObjChurn:
		g.churn()
	case ObjCow:
		g.fillBucket(ObjBucketOfMilk, "\nThe bucket of milk is too heavy for you.\n")
	case ObjWell:
		g.fillBucket(ObjBucketOfWater, "\nThe bucket of water is too heavy for you.\n")
	case ObjCrank:
		g.turnCrank(n)
	case ObjCrystalBall:
		g.gazeCrystalBall()
	case ObjBalloon:
		if g.MapZ != 0 && g.MapZ != gargoyleLevel {
			g.notUsable()
		} else {
			g.inflateBalloon(n)
		}
	case ObjFan:
		g.WindDir = (g.WindDir + 1) & 7
		g.printf("\nYou feel a breeze.\n")
		g.passTime(0)
	case ObjFishingPole:
		g.fish()
	case ObjFountain:
		g.wish()
	case ObjGargishVocabulary:
		g.printf("\nYou study the scroll!\n")
		g.SpeakGargish = 1
	case ObjGem:
		g.useGem(n)
	case ObjHarp:
		g.playInstrument(sound.EffectWavedLow)
	case ObjHarpsichord:
		g.playInstrument(sound.EffectWaved)
	case ObjLute:
		g.playInstrument(sound.EffectPluck)
	case ObjPanpipes:
		g.playInstrument(sound.EffectLowNote)
	case ObjXylophone:
		g.playInstrument(sound.EffectChime)
	case ObjHorse:
		g.mountHorse(n)
	case ObjMountedHorse:
		g.dismount(n)
	case ObjLockpick, ObjKey2:
		g.useKey(n)
	case ObjLadder:
		if g.SoloMember == -1 {
			g.climb(n)
		} else {
			g.printf("\nNot in solo mode.\n")
		}
	case ObjLever:
		g.switchLever(n, ObjPortcullis, "\nSwitch the lever, ")
	case ObjSwitch:
		g.switchLever(n, ObjElectricField, "\nOperate the switch, ")
	case ObjMoonstone:
		g.buryMoonstone(n)
	case ObjOrbOfTheMoons:
		if o.CoordUse() == OnMap {
			g.notUsable()
		} else {
			g.useOrb()
		}
	case ObjPassthrough, ObjPassthrough2:
		g.usePassthrough(n)
	case ObjPick, ObjShovel:
		if o.CoordUse() == Equipped {
			g.dig()
		} else {
			g.printf("\nNot readied.\n")
		}
	case ObjBluePotion:
		g.drinkPotion(n)
	case ObjPowderKeg:
		g.lightKeg(n)
	case ObjRubberDuckyG:
		g.printf("\nSqueak!\n")
		g.sfx(sound.EffectSqueak, 0)
	case ObjRuneOfHonesty, ObjRuneOfCompassion, ObjRuneOfValor, ObjRuneOfJustice, ObjRuneOfSacrifice,
		ObjRuneOfHonor, ObjRuneOfSpirituality, ObjRuneOfHumility:
		g.useRune(n)
	case ObjSecretDoor:
		o.SetFrame(toggleLit(frame))
	case ObjSextant:
		if g.MapZ != 0 && g.MapZ != gargoyleLevel {
			g.notUsable()
		} else {
			// TODO(port): C_1944_42AC (the sextant, with the spells)
		}
	case ObjSilverHorn:
		g.blowHorn()
	case ObjStaff:
		switch o.CoordUse() {
		case OnMap:
			g.notUsable()
		case Equipped:
			// TODO(port): C_27A1_3537 (cast the staff's charge, C_1944_43D5)
		default:
			g.printf("\nNot readied.\n")
		}
	case ObjTelescope:
		g.lookThrough(n)
	case ObjTorch:
		g.lightTorch(n)
	case ObjVortexCube:
		// TODO(port): C_27A1_5FAC (the vortex cube ends the game)
		g.printf("\n%s\n", noEffectMsg)
	}
}

// gargoyleLevel is the map level of the gargoyles' land, which is also outdoors.
const gargoyleLevel = 5

// toggleLit flips the lit (or open) bit of a frame and keeps the other bits but bit 1.
// C: seg_27A1:32FA (C_27A1_32FA)
func toggleLit(frame int) int {
	if frame&1 != 0 {
		return frame & 2
	}
	return frame | 1
}

// trap springs the trap of n.
// C: seg_27A1:28A3 (C_27A1_28A3)
func (g *Game) trap(n int) {
	var victim, x, y int
	if g.object(n).CoordUse() == OnMap {
		victim = g.user()
		x, y = g.userX(n), g.userY(n)
	} else {
		victim = g.memberUser(g.statusMember)
		x, y = g.userX(victim), g.userY(victim)
	}
	const damage = 20
	switch g.rand(0, 3) {
	case 0:
		g.printf("Acid!\n")
		g.loseHitPoints(victim, g.rand(1, damage))
		g.putch('\n')
		g.displaySeverity(victim)
	case 1:
		g.printf("Poison!\n")
		g.NPCs[victim].Status |= Poisoned
		g.showHit(victim)
	case 2:
		g.printf("Bomb!\n")
		for _, i := range append([]int(nil), g.explosion(x, y, tileBombBlast)...) {
			if i < npcCount {
				g.loseHitPoints(i, g.rand(1, damage))
				g.putch('\n')
				g.displaySeverity(i)
			}
		}
	case 3:
		g.printf("Gas!\n")
		for _, i := range append([]int(nil), g.explosion(x, y, tileGasCloud)...) {
			if i < npcCount {
				g.NPCs[i].Status |= Poisoned
				g.showHit(i)
			}
		}
	}
	g.DeleteObj(g.heldOfQuality(n, ObjEffect, trapSpell))
	if g.StatusDisplay == cmdStatusParty || g.StatusDisplay == cmdStatusPortrait {
		g.StatusDirty = true
	}
}

// useDoor opens or closes door n, or locks it with a key (lock) or a spell.
// C: seg_27A1:2A44 (C_27A1_2A44)
func (g *Game) useDoor(n int, lock, magicLock, magicUnlock bool) {
	g.putch('\n')
	o := g.object(n)
	f := o.Frame()
	switch {
	case f < doorClosedFirst:
		if !lock && !magicLock && !magicUnlock {
			g.printf("closed!\n")
			o.SetFrame(f ^ doorOpenBit)
		} else {
			g.printf("Can't (Un)lock an opened door\n")
		}
	case f < doorLockedFirst:
		switch {
		case lock:
			g.printf("locked!\n")
			o.SetFrame(f ^ doorLockedBoth)
		case magicLock:
			g.printf("magically locked!\n")
			o.SetFrame(f ^ doorLockBit)
		default:
			if g.FindInvType(n, ObjEffect, trapSpell) != noObject {
				g.trap(n)
			} else {
				g.printf("opened!\n")
			}
			o.SetFrame(f ^ doorOpenBit)
		}
	case f < doorMagicFirst:
		switch {
		case lock:
			g.printf("unlocked!\n")
			o.SetFrame(f ^ doorLockedBoth)
		case magicLock:
			g.printf("magically locked!\n")
			o.SetFrame(f ^ doorOpenBit)
		default:
			g.printf("locked\n")
		}
	default:
		if magicUnlock {
			g.printf("unlocked!\n")
			o.SetFrame(f ^ doorLockBit)
		} else {
			g.printf("magically locked!\n")
		}
	}
}

// useChest opens or closes chest n, or locks it.
// C: seg_27A1:2BBC (C_27A1_2BBC)
func (g *Game) useChest(n int, lock, magicLock, magicUnlock bool) {
	g.putch('\n')
	o := g.object(n)
	plain := !lock && !magicLock && !magicUnlock
	const chestMagic = 3
	switch o.Frame() {
	case chestOpen:
		if plain {
			g.printf("closed!\n")
			o.SetFrame(chestClosed)
		} else {
			g.printf("%s\n", noEffectMsg)
		}
	case chestClosed:
		switch {
		case plain:
			if g.FindInvType(n, ObjEffect, trapSpell) != noObject {
				g.trap(n)
			} else {
				g.printf("opened!\n")
			}
			o.SetFrame(chestOpen)
			g.search(n, o.Pos.X, o.Pos.Y)
		case lock:
			g.printf("locked!\n")
			o.SetFrame(chestLocked)
		case magicLock:
			g.printf("magically locked!\n")
			o.SetFrame(chestMagic)
		default:
			g.printf("%s\n", noEffectMsg)
		}
	case chestLocked:
		switch {
		case lock:
			g.printf("unlocked!\n")
			o.SetFrame(chestClosed)
		case magicLock:
			g.printf("magically locked!\n")
			o.SetFrame(chestMagic)
		default:
			g.printf("%s\n", noEffectMsg)
		}
	default:
		if magicUnlock {
			g.printf("unlocked!\n")
			o.SetFrame(chestClosed)
		} else {
			g.printf("%s\n", noEffectMsg)
		}
	}
	if o.CoordUse() != OnMap {
		g.drawInventory()
	}
}

// skillFails is the dexterity or intelligence test of a lockpick or crystal ball.
// C: seg_27A1:2D34 (C_27A1_2D34), seg_27A1:2D61 (C_27A1_2D61)
func (g *Game) skillFails(v int) bool {
	const odds, base = 30, 45
	return g.rand(1, odds) < (base-v)/2
}

// useKey locks or unlocks a door or chest with key or lockpick n.
// C: seg_27A1:2D8E (C_27A1_2D8E)
func (g *Game) useKey(n int) {
	actor := int(g.Party[g.Active])
	kt := g.object(n).Type()
	g.printf("On ")
	g.aim()
	if g.getch() != cmdSelected || g.Selection.obj == noObject {
		g.printf("%s\n", nothingMsg)
		return
	}
	sel := g.Selection.obj
	g.printf("%s", g.objectString(sel))
	g.putch('\n')
	t := g.object(sel).Type()
	p := g.object(actor).Pos
	switch {
	case g.object(sel).CoordUse() == OnMap && !closeEnough(1, g.Selection.x, g.Selection.y, p.X, p.Y):
		g.printf("%s!\n", outOfRangeMsg)
	case isDoor(t) || t == ObjOpenChest:
		k := g.object(n)
		switch {
		case kt == ObjLockpick && g.skillFails(g.dexterityOf(actor)):
			g.sfx(sound.EffectFailure, 0)
			g.printf("\nKey broke.\n")
			if k.Quantity() == 1 {
				g.DeleteObj(n)
			} else {
				k.setQuantity(k.Quantity() - 1)
			}
			if g.StatusDisplay == cmdStatusEquipment {
				g.drawInventory()
			}
		case kt == ObjKey2 && g.object(sel).Quality() != 0 && k.Quality() == g.object(sel).Quality() ||
			kt == ObjLockpick && g.object(sel).Quality() == 0:
			if t == ObjOpenChest {
				g.useChest(sel, true, false, false)
			} else {
				g.useDoor(sel, true, false, false)
			}
		default:
			g.printf("\n%s\n", noEffectMsg)
		}
	default:
		g.printf("%s\n", noEffectMsg)
	}
}

// lightTorch lights torch n (taking one from a stack) or puts out a lit one.
// C: seg_27A1:2FD1 (C_27A1_2FD1)
func (g *Game) lightTorch(n int) {
	o := g.object(n)
	count := o.Quantity()
	if o.Has(StatusLit) {
		g.DeleteObj(n)
		g.putch('\n')
		g.printf(burnOutMsg)
		return
	}
	if g.equipment[slotRightHand] != emptySlot && g.equipment[slotLeftHand] != emptySlot {
		g.printf("\nNo free hand to hold the torch.\n")
	} else {
		if count > 1 {
			stack := n
			if o.CoordUse() == OnMap {
				n = g.AddMapObj(shapeOf(ObjTorch, o.Frame()), 0, uint16(qualQuan(o.Quality(), 1)), o.Pos.X, o.Pos.Y, int(g.MapZ))
			} else {
				n = g.AddInvObj(shapeOf(ObjTorch, 0), Equipped, 1, int(g.Party[g.statusMember]))
			}
			g.object(stack).setQuantity(count - 1)
		}
		t := g.object(n)
		if u := t.CoordUse(); u == InInventory || u == Contained {
			if g.equipment[slotRightHand] == emptySlot {
				g.equipment[slotRightHand] = n
			} else {
				g.equipment[slotLeftHand] = n
			}
			t.SetCoordUse(Equipped)
			g.weighInventory(int(g.Party[g.statusMember]), false)
		}
		const torchLife = 200
		t.Status |= StatusLit
		t.SetFrame(litTorchFrame)
		t.setQuantity(torchLife)
		g.passTime(0)
		g.printf("\nTorch is lit.\n")
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
}

// useGem shows the map around the party and uses up the gem.
// C: seg_27A1:319F (C_27A1_319F)
func (g *Game) useGem(n int) {
	// TODO(port): C_2FC1_1FD6 (peer at the map)
	if n != 0 {
		if o := g.object(n); o.Quantity() > 1 {
			o.setQuantity(o.Quantity() - 1)
		} else {
			g.DeleteObj(n)
		}
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
}

// light lights or douses a fire.
// C: seg_27A1:31F6 (C_27A1_31F6)
func (g *Game) light(n int) {
	o := g.object(n)
	g.putch('\n')
	switch o.Type() {
	case ObjFireplace:
		g.printf("fireplace")
	case ObjBrazier:
		g.printf("brazier")
	case ObjCandelabra:
		g.printf("candelabra")
	case ObjCampfire:
		g.printf("campfire")
	case ObjCandle:
		g.printf("candle")
	}
	lit := o.Frame()&1 != 0
	o.SetFrame(toggleLit(o.Frame()))
	if lit {
		g.printf(" is doused.\n")
	} else {
		g.printf(" is lit.\n")
	}
}

// scaleHz are the notes of the instruments' keys 0..9 (D_1CEE).
var scaleHz = [10]int{7851, 3116, 3497, 3926, 4159, 4668, 5240, 5882, 6231, 6995}

// playInstrument plays a note for each digit typed.
// C: seg_27A1:335A (C_27A1_335A)
func (g *Game) playInstrument(kind int) {
	g.SelectMode = 0
	for {
		ch := g.getch()
		if ch < '0' || ch > '9' {
			return
		}
		g.sfx(kind, scaleHz[ch-'0'])
	}
}

// ringBell swings the bells and rings.
// C: seg_27A1:338D (C_27A1_338D)
func (g *Game) ringBell() {
	bell := int(g.BaseTile[ObjBell])
	for i := range g.animCount {
		if int(g.animTile[i]) != bell {
			continue
		}
		const swings, framesPerSwing = 16, 4
		for d := range swings {
			g.gr.SetTileFrame(bell, int(g.animFrame[i])+d/framesPerSwing)
			g.drawMap(true)
			g.paletteAnimation()
		}
		g.gr.SetTileFrame(bell, int(g.animFrame[i]))
		break
	}
	g.OtherAnimations()
	const step, units, steps, width, delta = 2000, 1, 16000, 20000, -1
	speaker(sound.PlayWavedNote(step, units, steps, width, delta))
}

// buryMoonstone buries moonstone n where the party stands, when that is earth.
// C: seg_27A1:3425 (C_27A1_3425)
func (g *Game) buryMoonstone(n int) {
	p := g.object(int(g.Party[g.Active])).Pos
	tile := g.GetTileAtXYZ(p.X, p.Y, int(g.MapZ))
	const groundFirst, groundLast, earthFirst, earthLast = 1, 7, 16, 111 // TIL_001..007, TIL_010..06F
	if tile >= groundFirst && tile <= groundLast || tile >= earthFirst && tile <= earthLast {
		g.printf("buried.\n")
		g.Moonstones[g.object(n).Frame()] = [3]int16{int16(p.X), int16(p.Y), g.MapZ}
		g.weighInventory(int(g.Party[g.statusMember]), false)
		g.MoveObj(n, p.X, p.Y, int(g.MapZ))
		g.MoveObj(int(g.Party[g.Active]), p.X, p.Y, int(g.MapZ))
		if g.StatusDisplay == cmdStatusEquipment {
			g.drawInventory()
		}
	} else {
		g.printf("Cannot be buried here!\n")
	}
}

// carrierStrong says whether n can carry extra tenths of stones more.
func (g *Game) carrierStrong(n, extra int) bool {
	g.weighInventory(n, false)
	return int(g.NPCs[n].Strength)*carryPerStrength >= g.weightInven+g.weightEquip+extra
}

// user is the character that uses things: the ship in a vehicle (C_27A1_0008).
func (g *Game) user() int { return g.memberUser(int(g.Active)) }

// memberUser is party member i, or the ship in a vehicle.
// C: seg_27A1:0008 (C_27A1_0008)
func (g *Game) memberUser(i int) int {
	if g.inVehicle() {
		return int(g.Party[0])
	}
	return int(g.Party[i])
}

// userX, userY are where object n is, or where the vehicle is.
// C: seg_27A1:0028 (C_27A1_0028), seg_27A1:0051 (C_27A1_0051)
func (g *Game) userX(n int) int {
	if g.inVehicle() {
		return g.object(0).Pos.X
	}
	return g.object(n).Pos.X
}

func (g *Game) userY(n int) int {
	if g.inVehicle() {
		return g.object(0).Pos.Y
	}
	return g.object(n).Pos.Y
}

// fillBucket fills a bucket at a well or a cow.
// C: seg_27A1:3661 (C_27A1_3661), seg_27A1:36E7 (C_27A1_36E7)
func (g *Game) fillBucket(full ObjectType, tooHeavy string) {
	n := g.user()
	bucket := g.FindInvType(n, ObjBucket, 0)
	if bucket == noObject {
		g.printf("\nYou need a bucket.\n")
		return
	}
	const bucketWeight = 32
	if !g.carrierStrong(n, bucketWeight) {
		g.printf(tooHeavy)
		return
	}
	g.DeleteObj(bucket)
	g.giveObj(n, full, 1)
	g.printf("\n%s\n", doneMsg)
}

// churn makes butter of milk.
// C: seg_27A1:376D (C_27A1_376D)
func (g *Game) churn() {
	n := g.user()
	milk := g.FindInvType(n, ObjBucketOfMilk, 0)
	if milk == noObject {
		g.printf("\nYou need some milk.\n")
		return
	}
	g.DeleteObj(milk)
	g.giveObj(n, ObjButter, 1)
	g.giveObj(n, ObjBucket, 1)
	g.printf("\n%s\n", doneMsg)
}

// fill fills an empty jar at a beehive.
// C: seg_27A1:37D3 (C_27A1_37D3)
func (g *Game) fill(empty, full ObjectType, missing string) {
	jar := g.FindInvType(g.user(), empty, 0)
	if jar == noObject {
		g.printf(missing)
		return
	}
	g.setType(jar, full)
	g.printf("\n%s\n", doneMsg)
	g.drawInventory()
}

// Potion colours (frames of ObjBluePotion).
const (
	potionAwaken = iota
	potionCure
	potionHeal
	potionPoison
	potionSleep
	potionProtection
	potionInvisibility
	potionXray
)

// drinkPotion gives potion n to a party member.
// C: seg_27A1:3832 (C_27A1_3832)
func (g *Game) drinkPotion(n int) {
	g.printf("%s", "On whom: ")
	g.MouseMode = 1
	if g.SelectMode != 0 {
		g.drawStatusCursor(0)
		g.SelectMode = selectOnMap
		g.SelectRange = aimRange
	}
	g.getch()
	sel := g.Selection
	if sel.obj == noObject || sel.x == -1 || sel.y == -1 {
		g.printf("%s\n", nothingMsg)
		return
	}
	g.printf("%s", g.objectString(sel.obj))
	g.putch('\n')
	if !g.isPlayerControlled(sel.obj) {
		g.printf("%s\n", noEffectMsg)
		return
	}
	npc := &g.NPCs[sel.obj]
	noEffect := func() { g.printf("\n%s\n", noEffectMsg) }
	switch g.object(n).Frame() {
	case potionAwaken:
		if npc.Has(Asleep) {
			npc.Status &^= Asleep
			g.object(sel.obj).Shape = npc.OrigShape
		} else {
			noEffect()
		}
	case potionCure:
		if npc.Has(Poisoned) {
			npc.Status &^= Poisoned
		} else {
			noEffect()
		}
	case potionHeal:
		// TODO(port): C_1944_114D (SPELL_06, heal)
	case potionPoison:
		if !classHas(monsterClass(g.object(sel.obj).Type()), immunePoison) {
			npc.Status |= Poisoned
			g.showHit(sel.obj)
		} else {
			noEffect()
		}
	case potionSleep:
		// the original tests the undead bit, not immuneSleep
		if !classHas(monsterClass(g.object(sel.obj).Type()), undead) {
			npc.Status |= Asleep
			g.setUnconscious(sel.obj)
		} else {
			noEffect()
		}
	case potionProtection:
		if !npc.Has(Protected) {
			npc.Status |= Protected
			g.magicalHalo(sel.obj, 1)
		} else {
			noEffect()
		}
	case potionInvisibility:
		if !g.object(sel.obj).Has(StatusInvisible) {
			g.object(sel.obj).Status |= StatusInvisible
			g.magicalHalo(sel.obj, 1)
		} else {
			noEffect()
		}
	case potionXray:
		// TODO(port): C_1944_310C (SPELL_49, x-ray)
	}
	g.DeleteObj(n)
	if g.StatusDisplay == cmdStatusParty || g.StatusDisplay == cmdStatusPortrait {
		g.StatusDirty = true
	}
}

// cannon shots by frame (D_1D02, D_1D06)
var cannonStepX, cannonStepY = [4]int{0, 1, 0, -1}, [4]int{-1, 0, 1, 0}

// fireCannon shoots cannon n.
// C: seg_27A1:3A35 (C_27A1_3A35)
func (g *Game) fireCannon(n int) {
	g.printf("\nFire!\n")
	o := g.object(n)
	f := o.Frame()
	x, y := o.Pos.X, o.Pos.Y
	const reach, cannonBall = 6, 399 // TIL_18F
	d := 1
	for ; d < reach; d++ {
		if hit := g.FindLoc(cannonStepX[f]*d+x, cannonStepY[f]*d+y, int(g.MapZ)); hit >= 0 && hit < npcCount {
			break
		}
	}
	x += cannonStepX[f] * d
	y += cannonStepY[f] * d
	if !g.throw(n, &x, &y, reach-1, cannonBall) {
		g.missileX += cannonStepX[f]
		g.missileY += cannonStepY[f]
		g.explosion(g.missileX, g.missileY, tileFireBolt)
	}
	const blastDamage = 60
	if hit := g.FindLoc(g.missileX, g.missileY, int(g.MapZ)); hit >= 0 && quantityType(g.object(hit).Type()) == countSingle {
		g.loseHitPoints(hit, g.rand(1, blastDamage))
	}
}

// lightKeg lights powder keg n.
// C: seg_27A1:3B59 (C_27A1_3B59)
func (g *Game) lightKeg(n int) {
	o := g.object(n)
	if o.Frame() != 0 {
		g.printf("\n%s\n", noEffectMsg)
		return
	}
	if g.KegMinutes != 0 {
		g.printf("\nNot now\n")
		return
	}
	const fuse = 3
	o.SetFrame(1)
	g.KegMinutes, g.KegObject = fuse, int16(n)
	g.printf("\nPowder lit!\n")
	if o.CoordUse() == OnMap {
		g.computeView()
		g.OtherAnimations()
	} else {
		g.drawInventory()
	}
}

// typeAt returns the first object of type t at (x, y, z).
// C: seg_27A1:3E21 (__SearchTypeAt)
func (g *Game) typeAt(x, y, z int, t ObjectType) int {
	i := g.FindLoc(x, y, z)
	for i >= 0 && g.object(i).Type() != t {
		i = g.NextLoc()
	}
	return i
}

// The drawbridge (C_27A1_3E59 ... C_27A1_433D).
const (
	bridgeOpenFirst = 3 // the frame of the first piece of an open bridge
	bridgeShut      = 6 // the frame of the first piece of a shut bridge
	crankTile1      = 1009
	crankTile2      = 1020 // TIL_3F1, TIL_3FC
)

// banks end an opened drawbridge (D_1D0A: TIL_010, 01C, 020, 02C, 0D6).
var banks = []int{16, 28, 32, 44, 214}

// thingOnBridge returns what is not the bridge at (x, y, z), 0 for nothing.
// C: seg_27A1:3E59 (C_27A1_3E59)
func (g *Game) thingOnBridge(x, y, z int) int {
	i := g.FindLoc(x, y, z)
	for i >= 0 && g.object(i).Type() == ObjDrawbridge {
		i = g.NextLoc()
	}
	return max(i, 0)
}

// characterAt returns the character at (x, y, z), 0 for none.
// C: seg_27A1:3E9A (C_27A1_3E9A)
func (g *Game) characterAt(x, y, z int) int {
	i := g.FindLoc(x, y, z)
	for i >= 0 && i >= npcCount {
		i = g.NextLoc()
	}
	return max(i, 0)
}

func isBank(tile int) bool {
	for _, b := range banks {
		if b == tile {
			return true
		}
	}
	return false
}

// addFixed puts a part of the bridge (or a portcullis) on the map for good.
// C: seg_27A1:3EF6 (C_27A1_3EF6)
func (g *Game) addFixed(x, y int, t ObjectType, frame int) int {
	n := g.AddMapObj(shapeOf(t, frame), 0, 0, x, y, int(g.MapZ))
	if n >= 0 {
		g.object(n).Status &^= StatusLocal
	}
	return n
}

// Results of moveBridge.
const (
	bridgeNothing = iota
	bridgeClosed
	bridgeOpened
	bridgeCannotClose
	bridgeCannotOpen
)

// moveBridge opens or closes the drawbridge whose first piece is n.
// C: seg_27A1:3F47 (C_27A1_3F47)
func (g *Game) moveBridge(n int) int {
	o := g.object(n)
	quality, frame := o.Quality(), o.Frame()
	ox, oy := o.Pos.X, o.Pos.Y
	z := int(g.MapZ)
	switch frame {
	case bridgeOpenFirst:
		x, y := ox, oy
		for piece := n; piece >= 0; {
			for piece >= 0 {
				if g.thingOnBridge(x, y, z) != 0 {
					return bridgeCannotClose
				}
				x++
				piece = g.typeAt(x, y, z, ObjDrawbridge)
			}
			x = ox
			y++
			piece = g.typeAt(x, y, z, ObjDrawbridge)
		}
		const shutLeft, shutMiddle, shutRight = 6, 7, 8
		piece, x, y := n, ox, oy
		first := g.addFixed(x-1, y-1, ObjDrawbridge, shutLeft)
		g.object(first).setQuality(quality)
		last := noObject
		for piece >= 0 {
			g.DeleteObj(piece)
			last = g.addFixed(x, y-1, ObjDrawbridge, shutMiddle)
			x++
			piece = g.typeAt(x, y, z, ObjDrawbridge)
		}
		g.DeleteObj(last)
		g.addFixed(x-1, y-1, ObjDrawbridge, shutRight)
		x, y = ox, oy+1
		piece = g.typeAt(x, y, z, ObjDrawbridge)
		for piece >= 0 {
			for piece >= 0 {
				g.DeleteObj(piece)
				x++
				piece = g.typeAt(x, y, z, ObjDrawbridge)
			}
			x = ox
			y++
			piece = g.typeAt(x, y, z, ObjDrawbridge)
		}
		g.computeView()
		return bridgeClosed
	case bridgeShut:
		width := 1
		x, y := ox, oy
		for piece := n; piece >= 0; {
			width++
			x++
			piece = g.typeAt(x, y, z, ObjDrawbridge)
		}
		const unused = 4
		x, y = ox+1, oy+1
		for !isBank(g.GetTileAtXYZ(x, y, z)) {
			for range width - unused {
				if g.thingOnBridge(x, y, z) != 0 {
					return bridgeCannotOpen
				}
				x++
			}
			x = ox + 1
			y++
		}
		x, y = ox, oy
		for piece := n; piece >= 0; {
			g.DeleteObj(piece)
			x++
			piece = g.typeAt(x, y, z, ObjDrawbridge)
		}
		const openLeft, openMiddle, openRight, endLeft, endMiddle, endRight = 3, 4, 5, 0, 1, 2
		x, y = ox+1, oy+1
		for !isBank(g.GetTileAtXYZ(x, y, z)) {
			first := g.addFixed(x, y, ObjDrawbridge, openLeft)
			x++
			g.object(first).setQuality(quality)
			for range width - unused {
				g.addFixed(x, y, ObjDrawbridge, openMiddle)
				x++
			}
			g.addFixed(x, y, ObjDrawbridge, openRight)
			x = ox + 1
			y++
		}
		resettle := func(x, y int) {
			if t := g.thingOnBridge(x, y, z); t != 0 {
				p := g.object(t).Pos
				g.MoveObj(t, p.X, p.Y, z)
			}
		}
		g.addFixed(x, y, ObjDrawbridge, endLeft)
		x++
		resettle(x-1, y)
		for range width - unused {
			g.addFixed(x, y, ObjDrawbridge, endMiddle)
			x++
			resettle(x-1, y)
		}
		g.addFixed(x, y, ObjDrawbridge, endRight)
		resettle(x, y)
		g.computeView()
		return bridgeOpened
	}
	return bridgeNothing
}

// turnCrank works the drawbridge of the crank's quality.
// C: seg_27A1:433D (C_27A1_433D)
func (g *Game) turnCrank(n int) {
	q := g.object(n).Quality()
	bridge := g.SearchArea(0, 0, surfaceMask, surfaceMask)
	for bridge >= 0 && (g.object(bridge).Type() != ObjDrawbridge || g.object(bridge).Quality() != q) {
		bridge = g.NextArea()
	}
	if bridge == noObject {
		g.printf("\nNo correspondent drawbridge\n")
		return
	}
	state := byte(animForward)
	if g.object(bridge).Frame() == bridgeOpenFirst {
		state = animBackward
	}
	g.setTileAnimation(crankTile1, state)
	g.setTileAnimation(crankTile2, state)
	const turns = 20
	for range turns {
		g.OtherAnimations()
	}
	switch g.moveBridge(bridge) {
	case bridgeNothing:
		g.printf("\nStrange, nothing happened.\n")
	case bridgeClosed:
		g.printf("\nClose the drawbridge.\n")
	case bridgeOpened:
		g.printf("\nOpen the drawbridge.\n")
	case bridgeCannotClose:
		g.printf("\nCan't close the drawbridge.\n")
	case bridgeCannotOpen:
		g.printf("\nCan't open the drawbridge.\n")
	}
	g.setTileAnimation(crankTile1, animStopped)
	g.setTileAnimation(crankTile2, animStopped)
}

// switchLever toggles lever or switch n, which raises or lowers the portcullises (or
// electric fields, gate) in the doorways of its quality.
// C: seg_27A1:4479 (C_27A1_4479), seg_27A1:4672 (C_27A1_4672)
func (g *Game) switchLever(n int, gate ObjectType, msg string) {
	const (
		nothing = iota
		noise
		cannotClose
	)
	result := nothing
	g.printf(msg)
	o := g.object(n)
	o.SetFrame(o.Frame() ^ 1)
	if q := o.Quality(); q != 0 {
		for d := g.SearchArea(0, 0, surfaceMask, surfaceMask); d >= 0; {
			for d >= 0 && (g.object(d).Type() != ObjDoorway || g.object(d).Quality() != q) {
				d = g.NextArea()
			}
			if d < 0 {
				continue
			}
			p := g.object(d).Pos
			if g := g; gate == ObjPortcullis {
				if bar := g.typeAt(p.X, p.Y, int(g.MapZ), gate); bar == noObject {
					if g.characterAt(p.X, p.Y, int(g.MapZ)) != 0 {
						result = cannotClose
						break
					}
					const sideways, facing, verticalDoorway = 3, 1, 2
					if g.object(d).Frame()&verticalDoorway != 0 {
						g.addFixed(p.X, p.Y, gate, sideways)
					} else {
						g.addFixed(p.X, p.Y, gate, facing)
					}
				} else {
					g.DeleteObj(bar)
				}
			} else if field := g.typeAt(p.X, p.Y, int(g.MapZ), gate); field == noObject {
				g.addFixed(p.X, p.Y, gate, 0)
			} else {
				g.DeleteObj(field)
			}
			result = noise
			d = g.NextArea()
		}
	}
	switch result {
	case nothing:
		g.printf("strange, nothing happened.\n")
	case noise:
		g.printf("you hear a noise.\n")
	case cannotClose:
		g.printf("can't close the portcullis.\n")
	}
}

// buildBalloon makes a balloon of its parts, which the party must carry.
// C: seg_27A1:47F3 (C_27A1_47F3)
func (g *Game) buildBalloon() {
	g.putch('\n')
	parts := []struct {
		t       ObjectType
		missing string
	}{
		{ObjMammothSilkBag, "Missing a mammoth silk bag.\n"},
		{ObjBalloonBasket, "Missing a balloon basket.\n"},
		{ObjCauldron, "Missing a cauldron.\n"},
		{ObjRope, "Missing a rope.\n"},
	}
	var holders [4]int
	missing := false
	for i, p := range parts {
		if holders[i] = g.partyMemberHolding(p.t, 0); holders[i] == noObject {
			g.printf(p.missing)
			missing = true
		}
	}
	if missing {
		return
	}
	for i, p := range parts {
		g.DeleteObj(g.heldOfQuality(holders[i], p.t, 0))
	}
	actor := int(g.Party[g.Active])
	p := g.object(actor).Pos
	b := g.AddMapObj(shapeOf(ObjBalloon, 0), 0, 0, p.X, p.Y, int(g.MapZ))
	g.setOkToGet(b)
	g.object(b).Status &^= StatusLocal
	g.MoveObj(actor, p.X, p.Y, int(g.MapZ))
	g.printf("Done!\n")
}

// placeNear moves n to a free tile around (x, y); it says whether there was one.
func (g *Game) placeNear(n, x, y int) bool {
	for i := range 3 {
		for j := range 3 {
			if i == 1 && j == 1 {
				continue
			}
			if g.passable(n, x+i-1, y+j-1) {
				g.MoveObj(n, x+i-1, y+j-1, int(g.MapZ))
				return true
			}
		}
	}
	return false
}

// inflateBalloon sets up deflated balloon n next to the party.
// C: seg_27A1:49C3 (C_27A1_49C3)
func (g *Game) inflateBalloon(n int) {
	g.DeleteObj(n)
	p := g.object(int(g.Party[g.Active])).Pos
	const inflated = 3
	b := g.AddMapObj(shapeOf(ObjBalloon2, inflated), 0, 0, p.X, p.Y, int(g.MapZ))
	g.object(b).Status &^= StatusLocal
	// the original skips only (0, 0), the tile up-left, not the party's own
	placed := false
	for i := 0; i < 3 && !placed; i++ {
		for j := range 3 {
			if i == 0 && j == 0 {
				continue
			}
			if g.passable(b, p.X+i-1, p.Y+j-1) {
				g.MoveObj(b, p.X+i-1, p.Y+j-1, int(g.MapZ))
				g.printf("\nDone!\n")
				placed = true
				break
			}
		}
	}
	if !placed {
		g.printf("\nNo room!\n")
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.refreshInventory(g.container)
	}
}

// mantras of the runes (D_1D0F).
var mantras = [8]string{"Ahm", "Mu", "Ra", "Beh", "Cah", "Summ", "Om", "Lum"}

// useRune opens the shrine of the rune's virtue when its mantra is said next to the
// moonstone.
// C: seg_27A1:4B98 (C_27A1_4B98)
func (g *Game) useRune(n int) {
	virtue := int(g.object(n).Type() - ObjRuneOfHonesty)
	g.printf("Mantra: ")
	said := g.gets(4)
	g.putch('\n')
	i := 0
	for i < len(mantras) && !strings.EqualFold(said, mantras[i]) {
		i++
	}
	if i != virtue {
		g.printf("\nWrong mantra!\n")
		return
	}
	p := g.object(int(g.Party[g.Active])).Pos
	freed := false
	for i := range 3 {
		for j := range 3 {
			if i == 1 && j == 1 {
				continue
			}
			x, y := p.X+i-1, p.Y+j-1
			stone := g.typeAt(x, y, int(g.MapZ), ObjMoonstone)
			if stone >= 0 && g.object(stone).Frame() == virtue {
				if field := g.typeAt(x, y, int(g.MapZ), ObjForceField); field >= 0 {
					g.DeleteObj(field)
				}
				sp := g.object(stone).Pos
				g.destroyGargoyleEggs(sp.X, sp.Y)
				freed = true
				break
			}
		}
	}
	if freed {
		// TODO(port): C_1944_0085 (the success sound and flash of a spell)
		g.printf("\n%s\n", doneMsg)
	} else {
		g.printf("\n%s\n", noEffectMsg)
	}
}

// destroyGargoyleEggs removes the gargoyle eggs around a shrine.
// C: seg_27A1:4B0B (C_27A1_4B0B)
func (g *Game) destroyGargoyleEggs(x, y int) {
	const reach = 20
	for e := g.SearchArea(x-reach, y-reach, x+reach, y+reach); e >= 0; e = g.NextArea() {
		if g.object(e).Type() != ObjEgg {
			continue
		}
		i := g.FindInv(e)
		if i < 0 {
			continue
		}
		if t := g.object(i).Type(); t == ObjWingedGargoyle || t == ObjGargoyle {
			for ; i >= 0; i = g.FindInv(e) {
				g.DeleteObj(i)
			}
			g.DeleteObj(e)
		}
	}
}

// fish catches a fish now and then next to water.
// C: seg_27A1:4D46 (C_27A1_4D46)
func (g *Game) fish() {
	n := g.user()
	p := g.object(n).Pos
	near := false
	for i := 0; i < 3 && !near; i++ {
		for j := range 3 {
			if i == 1 && j == 1 {
				continue
			}
			// the original looks at (x+i-1, y+i-1), the diagonal only
			if g.TerrainType[g.GetTileAtXYZ(p.X+i-1, p.Y+i-1, int(g.MapZ))]&terrainWet == 0 {
				continue
			}
			const fishOdds, fishWeight = 4, 5
			if g.CaughtFish == 0 && g.rand(0, fishOdds) == 0 {
				if !g.carrierStrong(n, fishWeight) {
					g.printf("\nGot it, but can't carry it.\n")
				} else {
					g.printf("\nGot it!\n")
					g.AddInvObj(shapeOf(ObjFish, 0), InInventory, 1, n)
					g.CaughtFish = 1
				}
			} else {
				g.printf("\nDidn't get a fish.\n")
			}
			near = true
			break
		}
	}
	if !near {
		g.printf("\nYou need to stand next to water.\n")
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
}

// drink drinks wine, mead or ale.
// C: seg_27A1:4E6F (C_27A1_4E6F)
func (g *Game) drink(n int) {
	g.DeleteObj(n)
	g.DrunkCounter++
	g.printf("\nYou drink it.\n")
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
}

// wishes a fountain grants (D_1D3F).
var wishes = []string{"Food", "Mutton", "Wine", "Fruit", "Meat"}

// wish asks a fountain for food.
// C: seg_27A1:4E9B (C_27A1_4E9B)
func (g *Game) wish() {
	n := g.user()
	g.printf("Make a wish?")
	if g.getch() == 'Y' {
		g.printf("\nWish for: ")
		said := g.gets(9)
		// fix: the original compared with 10 wishes of a list of 5
		granted := false
		for _, w := range wishes {
			if !strings.EqualFold(said, w) {
				continue
			}
			const odds, foodWeight = 4, 2
			if g.rand(0, odds) == 0 {
				g.printf("\n\nYou got food")
				if !g.carrierStrong(n, foodWeight) {
					g.printf(", but you can't carry it.\n")
				} else {
					g.giveObj(n, ObjPortionOfMeat, 1)
					g.printf(".\n")
				}
			} else {
				g.printf("\n\n%s\n", failedMsg)
			}
			granted = true
			break
		}
		if !granted {
			g.printf("\n\n%s\n", noEffectMsg)
		}
	} else {
		g.printf("\n")
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
}

// dig digs with a pick or shovel.
// C: seg_27A1:4FC6 (C_27A1_4FC6)
func (g *Game) dig() {
	g.printf("Direction:")
	g.MouseMode = 0
	g.aim()
	if g.getch() != cmdAdvance {
		g.printf("nowhere.\n")
		return
	}
	g.printf("%s.\n", dirNames[g.AdvanceDir])
	x, y := dirStepX[g.AdvanceDir]+int(g.MapX), dirStepY[g.AdvanceDir]+int(g.MapY)
	z := int(g.MapZ)
	tile := g.GetTileAtXYZ(x, y, z)
	const (
		pirateX, pirateY         = 707, 835 // the buried entrance of the pirate cave
		pirateCave               = 19
		caveFloorFirst           = 108
		caveFloorLast            = 111 // TIL_06C..06F
		fountainOdds, nuggetOdds = 31, 9
	)
	switch {
	case z == 0 && x == pirateX && y == pirateY && g.typeAt(x, y, z, ObjHole) == noObject:
		h := g.AddMapObj(shapeOf(ObjHole, 0), 0, 0, x, y, z)
		g.object(h).setQuality(pirateCave)
		g.printf("\nYou dig a hole.\n")
	case z >= 1 && z <= 4 && tile >= caveFloorFirst && tile <= caveFloorLast:
		switch {
		case g.rand(0, fountainOdds) == 0:
			g.AddMapObj(shapeOf(ObjFountain, 0), 0, 0, x, y, z)
			g.printf("\nYou find a water fountain.\n")
		case g.rand(0, nuggetOdds) == 0:
			g.AddMapObj(shapeOf(ObjGoldNugget, 0), 0, 1, x, y, z)
			g.printf("\nYou find a gold nugget.\n")
		default:
			g.printf("\n%s\n", failedMsg)
		}
	default:
		g.printf("\n%s\n", noEffectMsg)
	}
}

// mountHorse puts the active character on horse n.
// C: seg_27A1:5503 (C_27A1_5503)
func (g *Game) mountHorse(n int) {
	n = g.combatHead(n)
	if n < firstMonster {
		g.printf("\nNot boardable!\n")
		return
	}
	const sherry, behLem = 9, 164
	actor := int(g.Party[g.Active])
	switch {
	case actor == sherry:
		g.printf("\nSherry says: \"Eeek!!! I'm afraid of horses!\"\n")
	case actor == behLem:
		g.printf("\nBehLem says: \"Horses are for food!\"\n")
	case g.object(actor).Type() == ObjMountedHorse:
		g.printf("\nYou're already on a horse!\n")
	default:
		o := g.object(n)
		x, y, frame := o.Pos.X, o.Pos.Y, o.Frame()
		g.setType(n, ObjGold)
		g.DeleteObj(n)
		g.mount(actor, x, y, frame, true)
	}
}

// dismount takes the rider of horse n off it.
// C: seg_27A1:55F0 (C_27A1_55F0)
func (g *Game) dismount(n int) {
	n = g.combatHead(n)
	o := g.object(n)
	x, y := o.Pos.X, o.Pos.Y
	dir := o.Frame() & 6
	t := ObjectType(g.NPCs[n].OrigShape & typeMask)
	o.Shape = shapeOf(t, 0)
	g.turnTo(n, dir)
	rear := g.FindLoc(x-dirStepX[dir], y-dirStepY[dir], int(g.MapZ))
	for rear >= 0 && g.object(rear).Type() != ObjMountedHorse {
		rear = g.NextLoc()
	}
	if rear >= 0 {
		g.DeleteObj(rear)
	} else {
		g.printf("\nRear horse???\n")
	}
	horse := g.generate(int(ObjHorse), 0, 0, int(g.MapZ))
	if !g.placeNear(horse, x, y) {
		g.printf("\nNo room!\n")
		g.DeleteObj(horse)
	}
	g.OtherAnimations()
}

// useOrb opens a red moongate where the party points.
// C: seg_27A1:5789 (C_27A1_5789)
func (g *Game) useOrb() {
	const orbFlag = 5
	if g.NPCs[orbFlag].TalkFlags>>5&1 == 0 {
		g.printf("\nYou can't figure out how to use it.\n")
		return
	}
	g.printf("Where: ")
	g.MouseMode = 1
	if g.SelectMode != 0 {
		g.drawStatusCursor(0)
		g.SelectMode = selectOnMap
		g.SelectRange = aimRange
	}
	g.getch()
	sel := g.Selection
	mx, my := int(g.MapX), int(g.MapY)
	const reach = 2
	if !closeEnough(reach, sel.x, sel.y, mx, my) {
		g.printf("%s\n", failedMsg)
		return
	}
	dest := 0
	if !(sel.y == my && closeEnough(1, sel.x, sel.y, mx, my)) {
		const width = 5
		dest = (sel.y-my+reach)*width + sel.x - mx + reach + 1
	}
	gate := g.AddMapObj(shapeOf(ObjRedGate, 1), 0, uint16(qualQuan(dest, 0)), 0, 0, int(g.MapZ))
	z := int(g.MapZ)
	if g.passable(gate, sel.x, sel.y) && g.typeAt(sel.x, sel.y, z, ObjDoorway) == noObject &&
		g.typeAt(sel.x, sel.y, z, ObjArchway) == noObject {
		g.printf("a red moon gate appears.\n")
		g.MoveObj(gate, sel.x, sel.y, z)
		g.screenFade = true
		g.computeView()
		g.OtherAnimations()
	} else {
		g.printf("%s\n", failedMsg)
		g.DeleteObj(gate)
	}
}

// gazeCrystalBall shows a far place.
// C: seg_27A1:5935 (C_27A1_5935)
func (g *Game) gazeCrystalBall() {
	n := g.user()
	if g.skillFails(int(g.NPCs[n].Intelligence)) {
		const hurt = 10
		g.loseHitPoints(n, g.rand(1, hurt))
		g.putch('\n')
		g.displaySeverity(n)
		return
	}
	// TODO(port): C_27A1_5935 the latitude and longitude prompt and C_1944_39A1 (the view)
	g.printf("\n%s\n", noEffectMsg)
}

// lookThrough looks far in a direction through telescope n.
// C: seg_27A1:5A97 (C_27A1_5A97)
func (g *Game) lookThrough(n int) {
	savedX, savedY := g.MapX, g.MapY
	p := g.object(n).Pos
	g.MapX, g.MapY = int16(p.X), int16(p.Y)
	g.ShowObjects()
	g.enterArea()
	g.OtherAnimations()
	g.printf("Direction-")
	g.MouseMode = 0
	if g.getch() != cmdAdvance {
		g.printf("nowhere.\n")
		// fix: the original left the view at the telescope here
		g.MapX, g.MapY = savedX, savedY
		g.enterArea()
		return
	}
	g.printf("%s.\n", dirNames[g.AdvanceDir])
	g.xray = true
	g.screenFade = true
	g.ShowObjects()
	g.enterArea()
	g.OtherAnimations()
	const farthest, blocking = 40, 252 // TIL_0FC
	for steps, tile := farthest, 0; steps >= 0 && tile < blocking; steps-- {
		g.MapX += int16(dirStepX[g.AdvanceDir])
		g.MapY += int16(dirStepY[g.AdvanceDir])
		g.ShowObjects()
		g.enterArea()
		g.OtherAnimations()
		tile = g.GetTileAtXYZ(int(g.MapX), int(g.MapY), int(g.MapZ))
	}
	g.xray = false
	g.MapX, g.MapY = savedX, savedY
	g.ShowObjects()
	g.enterArea()
	g.screenFade = true
	g.OtherAnimations()
	g.printf("\n%s\n", doneMsg)
}

// blowHorn calls silver serpents.
// C: seg_27A1:5BCF (C_27A1_5BCF)
func (g *Game) blowHorn() {
	for i := firstMonster; i < npcCount; i++ {
		if g.object(i).Shape != 0 && g.object(i).Type() == ObjSilverSerpent {
			g.printf("\nNot now!\n")
			return
		}
	}
	z := int(g.MapZ)
	probe := g.AddMapObj(shapeOf(ObjSilverSerpent, 0), 0, 1, 0, 0, z)
	made := 0
	const serpents, tries = 3, 8
	for range serpents {
		x, y, ok := 0, 0, false
		for range tries {
			x, y = g.edgeSpot()
			if g.passable(probe, x, y) && g.passable(probe, x-1, y) && g.passable(probe, x, y+1) &&
				g.passable(probe, x-1, y+1) && g.passable(probe, x+1, y) && g.passable(probe, x, y-1) {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		const certain, eggLength, minLength, maxLength = 100, 8, 4, 8
		egg := g.AddMapObj(shapeOf(ObjEgg, 0), 0, uint16(qualQuan(4, certain)), x, y, z)
		g.AddInvObj(shapeOf(ObjSilverSerpent, 0), Contained, uint16(qualQuan(eggLength, g.rand(minLength, maxLength))), egg)
		g.hatch(egg)
		made++
	}
	g.DeleteObj(probe)
	if made == 0 {
		g.printf("\nNone generated!\n")
		return
	}
	g.printf("\nSilver snakes are generated!\n")
}

// edgeSpot picks a random tile just outside the view.
func (g *Game) edgeSpot() (int, int) {
	mx, my := int(g.MapX), int(g.MapY)
	side, along := g.rand(0, 3), g.rand(0, 10)-6
	switch side {
	case 0:
		return mx - 6, my + along
	case 1:
		return mx + along, my - 7
	case 2:
		return mx + 7, my + along
	default:
		return mx + along, my + 6
	}
}

// usePassthrough opens or closes a passthrough.
// C: seg_27A1:5DF2 (C_27A1_5DF2)
func (g *Game) usePassthrough(n int) {
	o := g.object(n)
	t, f := o.Type(), o.Frame()
	x, y := o.Pos.X, o.Pos.Y
	z := int(g.MapZ)
	const openBit = 2
	if f == 0 {
		if t == ObjPassthrough {
			g.MoveObj(n, x, y-1, z)
		} else {
			g.MoveObj(n, x-1, y, z)
		}
		o.SetFrame(f | openBit)
		g.printf("\nOpen the passthrough.\n")
		return
	}
	tx, ty := x+1, y
	if t == ObjPassthrough {
		tx, ty = x, y+1
	}
	if c := g.FindLoc(tx, ty, z); c != noObject && c < npcCount {
		g.printf("\nNot now!\n")
		return
	}
	g.MoveObj(n, tx, ty, z)
	o.SetFrame(f & 1)
	g.printf("\nClose the passthrough.\n")
}

// eat eats food n.
// C: seg_27A1:5F43 (C_27A1_5F43)
func (g *Game) eat(n int) {
	g.printf("\nYou eat the food.\n")
	carried := g.object(n).CoordUse() != OnMap
	if o := g.object(n); quantityType(o.Type()) != countSingle && o.Quantity() > 1 {
		o.setQuantity(o.Quantity() - 1)
	} else {
		g.DeleteObj(n)
	}
	if carried {
		g.drawInventory()
	}
}

// deadShapes are the shapes characters take when they fall (DeadShape).
var deadShapes = map[ObjectType]uint16{
	ObjHorse: shapeOf(ObjHorseSCarcass, 1), ObjDaemon: shapeOf(ObjDeadBody, 0), ObjGargoyle: shapeOf(ObjDeadBody, 0),
	ObjTroll: shapeOf(ObjDeadBody, 0), ObjHeadless: shapeOf(ObjDeadBody, 1), ObjCyclops: shapeOf(ObjDeadCyclops, 3),
	ObjMerchant: shapeOf(ObjDeadBody, 2), ObjVillager: shapeOf(ObjDeadBody, 2), ObjWoman: shapeOf(ObjDeadBody, 3),
	ObjMage: shapeOf(ObjDeadBody, 3), ObjGuard: shapeOf(ObjDeadBody, 4), ObjPeasant: shapeOf(ObjDeadBody, 5),
	ObjSwashbuckler: shapeOf(ObjDeadBody, 5), ObjFighter: shapeOf(ObjDeadBody, 6), ObjAvatar: shapeOf(ObjDeadBody, 7),
	ObjMusician: shapeOf(ObjDeadBody, 7), ObjMusician2: shapeOf(ObjDeadBody, 7), ObjChild: shapeOf(ObjDeadBody, 8),
	ObjFarmer: shapeOf(ObjDeadBody, 8), ObjFarmer2: shapeOf(ObjDeadBody, 8), ObjJester: shapeOf(ObjDeadBody, 8),
	ObjSkeleton: shapeOf(ObjDeadBody, 9), ObjWingedGargoyle: shapeOf(ObjDeadGargoyle, 3),
}

// vanishes marks a class that leaves no body (MONSTER_1000).
const vanishes = 4096

// setUnconscious gives a sleeping or dead character its lying shape.
// C: seg_2337:05B4 (SetTypeUnconscious)
func (g *Game) setUnconscious(n int) {
	o := g.object(n)
	t := o.Type()
	dead := g.NPCs[n].Has(Dead)
	if t == ObjMountedHorse {
		g.dismount(n)
		t = o.Type()
	} else if t == ObjDragon || t >= ObjGiantScorpion || t == ObjSilverSerpent || t == ObjShip {
		g.NPCs[n].OrigShape = o.Shape
	}
	cls := monsterClass(t)
	if shape, ok := deadShapes[t]; ok && !classHas(cls, vanishes) {
		if t != ObjHorse {
			o.Shape = shapeOf(ObjectType(shape&typeMask), 0)
		} else if dead {
			g.DeleteObj(g.objectHead(n))
			o.Shape = shapeOf(ObjectType(shape&typeMask), 0)
		}
	} else if dead {
		if cls >= 0 {
			if cls == classGazer {
				g.generate(int(ObjInsects), o.Pos.X, o.Pos.Y, int(g.MapZ))
			}
			if classHas(cls, disappears) || classHas(cls, vanishes) {
				g.DeleteObj(n)
				if classHas(cls, vanishes) && closeEnough(viewCenter, o.Pos.X, o.Pos.Y, int(g.MapX), int(g.MapY)) {
					g.drawMap(true)
					g.ShowObjects()
					g.screenFade = true
					g.drawMap(true)
				}
			}
		}
		if t == ObjSlime {
			g.updateSlimeFrames()
		}
		if t == ObjDragon {
			g.DeleteObj(n)
		}
	}
	if dead && (cls >= 0 && classHas(cls, bleeds) || t == ObjFarmer2 || t == ObjMusician2) {
		if b := g.AddMapObj(shapeOf(ObjBlood, 0), StatusLocal, 0, o.Pos.X, o.Pos.Y, int(g.MapZ)); b >= 0 {
			const bloodFrames = 2
			g.object(b).SetFrame(g.rand(0, bloodFrames))
		}
	}
	g.viewChanged = true
}

// ladderDown is the frame of a ladder that goes down.
const ladderDown = 1

// climb takes the party up or down ladder (or hole) n.
// C: seg_101C:089E (C_101C_089E)
func (g *Game) climb(n int) {
	if g.InCombat != 0 {
		g.breakOff()
	}
	o := g.object(n)
	step := 1
	if g.MapZ != 0 && (g.MapZ == gargoyleLevel || o.Shape == shapeOf(ObjLadder, ladderDown)) {
		step = -1
	}
	g.MapX, g.MapY = int16(o.Pos.X), int16(o.Pos.Y)
	g.partyEnter(2)
	g.MapZ += int16(step)
	mx, my := int(g.MapX), int(g.MapY)
	const (
		chunkMask    = 7
		dungeonAreas = 248 // 0xf8
		surfaceAreas = 992 // 0x3e0
		scale        = 2   // a dungeon tile is 4 surface tiles
	)
	switch {
	case g.MapZ == 1 && step == 1:
		mx = (mx>>scale)&dungeonAreas + mx&chunkMask
		my = (my>>scale)&dungeonAreas + my&chunkMask
	case g.MapZ == 0 && step == -1:
		mx = (mx<<scale)&surfaceAreas + mx&chunkMask
		my = (my<<scale)&surfaceAreas + my&chunkMask
		q := o.Quality()
		const half, whole = 8, 16
		if q&1 != 0 {
			mx += half
		}
		if q&2 != 0 {
			mx += whole
		}
		if q&4 != 0 {
			my += half
		}
		if q&8 != 0 {
			my += whole
		}
	}
	g.MapX, g.MapY = int16(mx), int16(my)
	g.enterArea()
	g.computeView()
	if g.MapZ == 0 || g.MapZ == gargoyleLevel || g.MapZ == 1 && step == 1 || g.MapZ == 4 && step == -1 {
		g.drawSky()
		g.passTime(0)
	}
	g.partyExit(mx, my, int(g.MapZ), 2)
	g.playAreaSong()
}

// partyEnter gathers the party on the leader and takes it off the map.
// C: seg_101C:0572 (PartyEnter)
func (g *Game) partyEnter(animation int) {
	for i := range int(g.PartySize) {
		if g.object(int(g.Party[i])).Type() == ObjMountedHorse {
			g.dismount(int(g.Party[i]))
		}
	}
	g.enterArea()
	mx, my, mz := int(g.MapX), int(g.MapY), int(g.MapZ)
	const steps, held = 30, 15
	if animation != 0 {
		for range steps {
			done := true
			for i := range int(g.PartySize) {
				n := int(g.Party[i])
				npc := &g.NPCs[n]
				if npc.Has(Asleep) {
					g.object(n).Shape = npc.OrigShape
				}
				npc.Status &^= Asleep | Paralyzed
				p := g.object(n).Pos
				if p.Z == mz && !g.incapacitated(n) && !g.draggedUnder(n) {
					switch {
					case p.X == mx && p.Y == my:
						g.ShowObjects()
						g.OtherAnimations()
						g.gameLoaded = false
						g.MoveObj(n, mx, my, held)
						g.gameLoaded = true
						if animation == 1 {
							g.screenFade = true
							g.ShowObjects()
							g.OtherAnimations()
						}
					case !g.TryMoveTo(n, mx, my):
						const nextTo = 2
						if g.distance(n, mx, my) < nextTo {
							g.MoveObj(n, mx, my, mz)
						} else {
							g.TryStraightMove(n, g.rand(0, 3)<<1, true)
						}
					default:
						done = false
					}
				}
				npc.MovePoints = int8(npc.Dexterity)
			}
			g.ShowObjects()
			g.OtherAnimations()
			if done {
				break
			}
		}
	}
	g.gameLoaded = false
	for i := range int(g.PartySize) {
		if g.object(int(g.Party[i])).Pos.Z != held {
			g.MoveObj(int(g.Party[i]), mx, my, held)
		}
	}
	g.gameLoaded = true
}

// partyExit puts the party back on the map at (x, y, z) and lets it fan out.
// C: seg_101C:077E (PartyExit)
func (g *Game) partyExit(x, y, z, animation int) {
	g.gameLoaded = false
	for i := range int(g.PartySize) {
		g.MoveObj(int(g.Party[i]), x, y, z)
	}
	g.gameLoaded = true
	g.SetPartyMode()
	if animation == 1 {
		g.screenFade = true
		g.ShowObjects()
		g.OtherAnimations()
	}
	const fanOut = 5
	for range fanOut {
		g.MoveFollowers(int(g.Party[g.Active]), true)
		if animation != 0 {
			g.ShowObjects()
			g.OtherAnimations()
		}
	}
	for i := range int(g.PartySize) {
		npc := &g.NPCs[g.Party[i]]
		npc.MovePoints = int8(npc.Dexterity)
	}
}

// partyTeleport moves the party to (x, y, z).
// C: seg_101C:0828 (PartyTeleport)
func (g *Game) partyTeleport(x, y, z, animation int) {
	g.partyEnter(animation)
	if gate := g.findTypeAt(ObjRedGate, int(g.MapX), int(g.MapY)); gate >= 0 {
		g.DeleteObj(gate)
	}
	g.MapX, g.MapY, g.MapZ = int16(x), int16(y), int16(z)
	g.enterArea()
	g.forceHatching = true
	g.hatchEggs()
	g.forceHatching = false
	g.computeView()
	g.drawSky()
	g.passTime(0)
	g.partyExit(x, y, z, animation)
}

// gateTravel takes the party to moonstone dest, or back where it stands when the stone
// is not buried.
// C: seg_101C:0A3A (GateTravel)
func (g *Game) gateTravel(dest int) {
	if g.InCombat != 0 {
		g.breakOff()
	}
	if s := g.Moonstones[dest]; s != [3]int16{} {
		g.partyTeleport(int(s[0]), int(s[1]), int(s[2]), 1)
	} else {
		g.partyTeleport(int(g.MapX), int(g.MapY), int(g.MapZ), 1)
	}
	g.playAreaSong()
}

// Red gate destinations by quality - 1 (D_171C, D_174E, D_1780).
var redGates = [25][3]int{
	{899, 499, 0}, {935, 262, 0}, {435, 395, 0}, {503, 358, 0}, {147, 883, 0},
	{919, 934, 0}, {68, 45, 5}, {307, 352, 0}, {188, 45, 5}, {159, 942, 0},
	{739, 699, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {227, 131, 0},
	{23, 22, 1}, {128, 86, 5}, {108, 221, 5}, {923, 876, 0}, {295, 38, 0},
	{75, 507, 0}, {327, 822, 0}, {387, 787, 0}, {831, 166, 0}, {667, 67, 0},
}

// placeNames are the dungeons and shrines Shamino names (D_17E2).
var placeNames = [...]string{"Deceit", "Despise", "Destard", "Wrong", "Covetous", "Shame", "Hythloth", "GSA",
	"Control", "Passion", "Diligence", "Tomb of Kings", "Ant Mound", "Swamp Cave", "Spider Cave",
	"Cyclops Cave", "Heftimus Cave", "Heroes' Hole", "Pirate Cave", "Buccaneer's Cave"}

// travelThrough takes the party through the moongate, red gate or entrance n.
// It returns false when solo mode refused it.
// C: seg_1E0F:184D (C_1E0F_184D), its body
func (g *Game) travelThrough(n int) bool {
	o := g.object(n)
	switch t := o.Type(); t {
	case ObjMoongate:
		if g.SoloMember != -1 {
			return false
		}
		const shrineMinutes = 10
		if g.Hour == 0 && g.Minute < shrineMinutes {
			const spiritX, spiritY, spiritZ = 24, 29, 1 // the shrine of spirituality
			g.partyTeleport(spiritX, spiritY, spiritZ, 1)
			g.playSong(sound.SongStones)
			g.passTime(shrineMinutes)
			return true
		}
		const full = 7
		ph := g.MoonPhases
		closer := abs(full-int(ph[1])) - abs(full-int(ph[3]))
		const halfHour = 30
		if closer < 0 || closer <= 0 && (g.Minute < halfHour) == (ph[1] < ph[3]) {
			g.gateTravel(int(ph[0]))
		} else {
			g.gateTravel(int(ph[2]))
		}
	case ObjRedGate:
		if g.SoloMember != -1 {
			return false
		}
		if q := o.Quality(); q != 0 {
			d := redGates[q-1]
			g.partyTeleport(d[0], d[1], d[2], 1)
			g.playAreaSong()
		} else {
			g.partyTeleport(int(g.MapX), int(g.MapY), int(g.MapZ), 1)
		}
	default:
		if q := o.Quality(); g.shaminoNear() && q != 0 && q <= len(placeNames) {
			g.printf("Shamino says, \"This is the ")
			const dungeons, shrinesFirst, shrinesLast = 8, 9, 11
			if q < dungeons {
				g.printf("dungeon ")
			} else if q >= shrinesFirst && q <= shrinesLast {
				g.printf("shrine of ")
			}
			g.printf(placeNames[q-1])
			g.printf(".\"\n")
			g.promptFlag = true
		}
		if g.SoloMember != -1 {
			return false
		}
		g.climb(n)
	}
	return true
}

// The places that rest needs.
const (
	restRoom       = 1
	restCampfire   = 1
	restTries      = 20
	minutesPerRest = 10
	healAfter      = 4 // hours of sleep that heal
	sleepBetween   = 12
	ambushOdds     = 64
	ambushMax      = 8
	nearOthers     = 5
	repairStep     = 5
	hullMax        = 100
)

// edibleTypes are what the party eats at rest (D_2CCE).
var edibleTypes = []ObjectType{ObjLoafOfBread, ObjPortionOfMeat, ObjRolls, ObjCake, ObjCheese, ObjHam,
	ObjHorseChops, ObjBunchOfGrapes, ObjMeat, ObjRibs, ObjBucketOfMilk, ObjJarOfHoney, ObjFish, ObjButter}

// eatFrom has n eat one of its edible objects.
// C: seg_3200:0035 (TryEatExclusive)
func (g *Game) eatFrom(n int) bool {
	for _, i := range g.heldBy(n) {
		if t := g.object(i).Type(); hasType(edibleTypes, t) {
			g.takeObj(n, t, 1)
			return true
		}
	}
	return false
}

// feed has member i eat, or another member share.
// C: seg_3200:0094 (TryEat)
func (g *Game) feed(i int) bool {
	if g.eatFrom(int(g.Party[i])) {
		return true
	}
	for j := range int(g.PartySize) {
		if g.eatFrom(int(g.Party[j])) {
			return true
		}
	}
	return false
}

// notWilderness says whether walls with sides or water stop the party resting.
// C: seg_3200:00D3 (C_3200_00D3)
func (g *Game) notWilderness() bool {
	mx, my, mz := int(g.MapX), int(g.MapY), int(g.MapZ)
	for x := mx - viewCenter; x <= mx+viewCenter; x++ {
		for y := my - viewCenter; y <= my+viewCenter; y++ {
			tile := g.GetTileAtXYZ(x, y, mz)
			if g.isWall(tile) && g.TerrainType[tile]&terrainSides != 0 {
				return true
			}
		}
	}
	return g.TerrainType[g.GetTileAtXYZ(mx, my, mz)]&terrainWet != 0 && !g.inVehicle()
}

// foesNear says whether an evil or chaotic creature is in the area.
// C: seg_3200:0160 (C_3200_0160)
func (g *Game) foesNear() bool {
	mask := coordMaskFor(int(g.MapZ))
	ax, ay := (int(g.MapX)-areaMargin)&areaAlign, (int(g.MapY)-areaMargin)&areaAlign
	const inArea = 39
	for n := range npcCount {
		o := g.object(n)
		a := g.NPCs[n].Status & alignmentMask
		if o.Shape != 0 && (a == alignEvil || a == alignChaotic) && (o.Pos.X-ax)&mask < inArea &&
			(o.Pos.Y-ay)&mask < inArea && o.Pos.Z == int(g.MapZ) {
			return true
		}
	}
	return false
}

// othersNear says whether someone not of the party is close.
// C: seg_3200:0226 (C_3200_0226)
func (g *Game) othersNear() bool {
	for n := range npcCount {
		o := g.object(n)
		if o.Shape != 0 && !g.isPlayerControlled(n) && closeEnough(nearOthers, o.Pos.X, o.Pos.Y, int(g.MapX), int(g.MapY)) &&
			o.Pos.Z == int(g.MapZ) && o.Type() != ObjHorse {
			return true
		}
	}
	return false
}

// noRoom says whether a tile around the party is blocked.
// C: seg_3200:02C4 (C_3200_02C4)
func (g *Game) noRoom() bool {
	for i := range 3 {
		for j := range 3 {
			if !g.passable(int(g.Party[g.Active]), int(g.MapX)+i-1, int(g.MapY)+j-1) {
				return true
			}
		}
	}
	return false
}

// nightAmbush brings creatures to the camp; it says whether any came.
// C: seg_3200:0309 (HasNightEncounter)
func (g *Game) nightAmbush() bool {
	kinds := [4]ObjectType{ObjGiantRat, ObjMongbat, ObjGiantBat, ObjGremlin}
	t := kinds[g.rand(0, 3)]
	came := false
	for range g.rand(1, ambushMax) {
		n := g.generate(int(t), 0, 0, int(g.MapZ))
		const tries = 8
		placed := false
		for range tries {
			x, y := g.edgeSpot()
			if g.passable(n, x, y) {
				g.MoveObj(n, x, y, int(g.MapZ))
				g.NPCs[n].Status = g.NPCs[n].Status&^alignmentMask | alignEvil
				g.NPCs[n].Mode = AIAssault
				came, placed = true, true
				break
			}
		}
		if !placed {
			g.DeleteObj(n)
		}
	}
	return came
}

// heal restores some hit points and all the magic of member i.
// C: seg_3200:0467 (C_3200_0467)
func (g *Game) heal(i int, say bool) {
	n := int(g.Party[i])
	npc := &g.NPCs[n]
	if lost := g.maxHitPoints(n) - int(npc.HitPoints); lost != 0 {
		if lost == 1 {
			lost = 2
		}
		npc.HitPoints = byte(min(int(npc.HitPoints)+g.rand(0, lost/2)+lost/2, g.maxHitPoints(n)))
		if say {
			g.printf("%s", cString(g.Names[i][:]))
			g.printf(" has healed.\n")
		}
	}
	npc.Magic = byte(g.maxMagic(n))
}

// camp places of the party around the fire, and the way each faces (D_2CEA..D_2CFA).
var (
	campX   = [8]int{0, 0, 1, -1, 1, -1, 1, -1}
	campY   = [8]int{-1, 1, 0, 0, -1, 1, 1, -1}
	campDir = [8]int{4, 0, 6, 2, 4, 0, 6, 2}
)

// rest camps for some hours, or repairs the ship.
// C: seg_3200:055D (C_3200_055D)
func (g *Game) rest() {
	g.printf("Rest")
	byMouse := g.SelectMode != 0
	switch {
	case g.InCombat != 0:
		g.printf("-Not while in Combat!\n")
		return
	case g.inVehicle() && g.object(int(g.Party[g.Active])).Type() != ObjShip:
		g.printf("-Can not be repaired!\n")
		return
	case g.notWilderness():
		g.printf("-Only in the wilderness!\n")
		return
	case g.foesNear():
		g.printf("-Not while foes are near!\n")
		return
	case g.othersNear():
		g.printf("-Not while others are near!\n")
		return
	case g.SoloMember != -1:
		g.printf("-Not in solo mode!\n")
		return
	case g.noRoom():
		g.printf("-Not enough room!\n")
		return
	case g.inVehicle():
		n := int(g.Party[g.Active])
		g.NPCs[n].HitPoints = byte(min(int(g.NPCs[n].HitPoints)+repairStep, hullMax))
		g.passTime(repairStep)
		g.printf("\nHull Strength: %d%%\n", int(g.NPCs[n].HitPoints))
		return
	}
	for i := range int(g.PartySize) {
		if g.object(int(g.Party[i])).Type() == ObjMountedHorse {
			g.printf("-Dismount first!\n")
			return
		}
	}
	g.printf("\nHow many hours? ")
	ch := g.getch()
	if ch < '1' || ch > '9' {
		g.printf("0\n")
		return
	}
	g.printf("%c\n", ch)
	hours := ch - '0'
	guard := npcCount
	if g.PartySize != 1 {
		g.printf("Who will guard? ")
		g.MouseMode = 1
		if byMouse {
			g.SelectRange = aimRange
			g.drawStatusCursor(0)
			g.SelectMode = selectOnMap
		}
		ch = g.getch()
		if !byMouse && ch >= '1' && ch < int(g.PartySize)+'0' {
			g.Selection.obj = int(g.Party[ch-'1'])
			ch = cmdSelected
			g.Selection.x = g.object(g.Selection.obj).Pos.X
		}
		g.SelectMode = 0
		g.OtherAnimations()
		sel := g.Selection
		if ch != cmdSelected || sel.obj == noObject || sel.x == -1 || sel.obj >= npcCount || !g.isPlayerControlled(sel.obj) {
			g.printf("none\n")
		} else {
			g.printf("%s", g.objectString(sel.obj))
			g.printf("\n")
			guard = sel.obj
		}
	}
	fire := g.AddMapObj(shapeOf(ObjCampfire, restCampfire), 0, 1, int(g.MapX), int(g.MapY), int(g.MapZ))
	g.ShowObjects()
	for tries, blocked := restTries, true; blocked && tries > 0; tries-- {
		blocked = false
		for i := range int(g.PartySize) {
			n := int(g.Party[i])
			if g.TryMoveTo(n, int(g.MapX)+campX[i], int(g.MapY)+campY[i]) {
				blocked = true
			} else {
				g.turnTo(n, campDir[i])
			}
		}
		g.ShowObjects()
		g.OtherAnimations()
	}
	g.printf("\nMealtime!\n")
	for i := range int(g.PartySize) {
		if n := int(g.Party[i]); g.object(n).Type() == ObjMusician {
			g.setType(n, ObjMusician2)
			g.turnTo(n, campDir[i])
			g.printf("%s", cString(g.Names[i][:]))
			g.printf(" plays a tune.\n")
			break
		}
	}
	g.ShowObjects()
	hungry := -1
	// ponytail: the original loops while STONES.M plays; the Go songs loop, so the meal
	// lasts one check per member, 4 frames apart.
	const framesPerMember = 4
	for i := 0; i == 0 || sound.CurrentSong() != "" && i < int(g.PartySize)*framesPerMember; i++ {
		if i%framesPerMember == 0 {
			if m := i / framesPerMember; m < int(g.PartySize) {
				g.printf("%s", cString(g.Names[m][:]))
				if g.feed(m) {
					g.printf(" has food.\n")
				} else {
					g.printf(" has no food.\n")
					if hungry == -1 {
						hungry = m
					}
				}
			}
		}
		g.animationTick = true
		g.OtherAnimations()
		g.paletteAnimation()
		g.serveMouse()
	}
	if hungry == -1 {
		hungry = int(g.PartySize)
	}
	if g.StatusDisplay == cmdStatusEquipment {
		g.drawInventory()
	}
	for i := range int(g.PartySize) {
		if n := int(g.Party[i]); g.object(n).Type() == ObjMusician2 {
			g.setType(n, ObjMusician)
			g.turnTo(n, campDir[i])
			break
		}
	}
	for i := range int(g.PartySize) {
		if int(g.Party[i]) != guard {
			g.setUnconscious(int(g.Party[i]))
		} else {
			g.printf("\n")
			g.printf("%s", cString(g.Names[i][:]))
			g.printf(" stands guard while the party rests.\n")
		}
	}
	g.ShowObjects()
	var guardMode AIMode
	if guard < npcCount {
		guardMode = g.NPCs[guard].Mode
		g.NPCs[guard].Mode = AIGraze
	}
	const (
		awake = iota
		heard
		surprised
	)
	woke := awake
	for i := range hours * 60 / minutesPerRest {
		if guard < npcCount {
			g.npcTurn(guard)
		}
		g.passTime(minutesPerRest)
		g.ShowObjects()
		g.animationTick = true
		g.OtherAnimations()
		g.paletteAnimation()
		g.serveMouse()
		const strange = 5 // the original tests (i & 5)
		if i&strange == 0 && g.rand(1, ambushOdds) == 1 && g.nightAmbush() {
			if guard < npcCount {
				g.printf("\nYou hear something!\n")
				woke = heard
			} else {
				woke = surprised
			}
			break
		}
	}
	g.object(fire).SetFrame(0)
	g.ShowObjects()
	for i := range int(g.PartySize) {
		n := int(g.Party[i])
		if woke < surprised {
			if n != guard {
				g.object(n).Shape = g.NPCs[n].OrigShape
			}
		} else {
			g.NPCs[n].Status |= Asleep
		}
		g.NPCs[n].MovePoints = int8(g.NPCs[n].Dexterity)
	}
	if guard < npcCount {
		g.NPCs[guard].Mode = guardMode
	}
	if hours >= healAfter && g.NextSleep == 0 && woke == awake {
		for i := range hungry {
			if int(g.Party[i]) != guard {
				g.heal(i, true)
			}
		}
		g.NextSleep = sleepBetween
	}
	p := g.object(int(g.Party[g.Active])).Pos
	g.MapX, g.MapY = int16(p.X), int16(p.Y)
	g.ShowObjects()
	g.enterArea()
	g.OtherAnimations()
	if g.StatusDisplay == cmdStatusParty {
		g.StatusDirty = true
	}
}
