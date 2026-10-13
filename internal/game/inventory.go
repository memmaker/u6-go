package game

// The portrait, equipment and inventory views of the status area, readying and the status
// cursor (seg_155D, parts of seg_0C9C and seg_2FC1).

import (
	"encoding/binary"
	"image"
	"io/fs"

	"github.com/memmaker/u6-go/internal/data"
	"github.com/memmaker/u6-go/internal/vga"
)

// The status area at the right of the screen.
const (
	statusX0, statusY0 = 168, 6
	statusX1, statusY1 = 312, 103
	statusTextColumns  = 17 // the width of the party clip, for centring names
	partyRowY          = 24
	// the buttons under the portrait and equipment views
	buttonsY       = 88
	buttonsBottom  = 104
	buttonPrevious = 176
	buttonParty    = 192
	buttonSwitch   = 208
	buttonNext     = 224
	buttonCombat   = 240
	buttonsEnd     = 256
	// the combat mode under the inventory
	combatModeX0, combatModeY0 = 256, 96
	combatModeX1               = 311
	combatModeColumn           = 10
	combatModeRow              = 11
	combatModeWidth            = 8
	// the portrait view
	portraitX, portraitY       = 176, 16
	portraitWidth              = 56
	portraitHeight             = 64
	statsColumn                = 10
	upArrowChar, downArrowChar = 24, 25
	lowHitPoints               = 10
	// the equipment view: the paper doll and the inventory grid
	dollX, dollY           = 192, 32
	inventoryX, inventoryY = 248, 32
	inventoryColumns       = 4
	inventorySlots         = 12
	inventoryRight         = 312
	inventoryBottom        = 80
	inventoryClearY0       = 16
	inventoryClearY1       = 79
	inventoryClearX1       = 311
	containerX, containerY = 272, 16
	scrollArrowX           = 240
	scrollArrowRight       = 248
	scrollUpY0, scrollUpY1 = 32, 40
	scrollDnY0, scrollDnY1 = 72, 80
	scrollArrowColumn      = 8
	scrollUpRow            = 3
	scrollDownRow          = 8
	weightRow              = 9
	equipWeightColumn      = 2
	inventoryWeightColumn  = 11
	equipLabelColumn       = 0
	inventoryLabelColumn   = 9
	nameRow                = 10 // the name under a looked-at character
)

// tiles of the status area
const (
	tileDollTopLeft   = 368 // TIL_170, the 2x2 paper doll
	tileDollTopRight  = 369
	tileDollBottomL   = 370
	tileDollBottomR   = 371
	tileSwitchToEquip = 385 // TIL_181
	tileOneHanded     = 389 // TIL_185, the left hand taken by a two-handed weapon
	tileCombatMode    = 391 // TIL_187
	tileEmptySlot     = 410 // TIL_19A
)

// Equipment slots (SLOT_xxx); slotTwoHanded and slotRing are what equipSlot returns for
// objects that take both hands or either finger.
const (
	slotHead = iota
	slotNeck
	slotRightHand
	slotRightFinger
	slotChest
	slotLeftHand
	slotLeftFinger
	slotFeet
	slotTwoHanded
	slotRing
	slotCount = slotTwoHanded
	noSlot    = -1
)

// emptySlot and twoHandedMark are the values of equipment[] for a free slot and for the
// left hand of a two-handed weapon.
const (
	emptySlot     = 0
	twoHandedMark = 1
)

// slotPlaces are where each slot is drawn, from the top left of the paper doll's area
// (C_155D_08F4).
var slotPlaces = [slotCount][2]int{
	slotHead: {200, 16}, slotNeck: {176, 24}, slotRightHand: {176, 40}, slotRightFinger: {176, 56},
	slotChest: {224, 24}, slotLeftHand: {224, 40}, slotLeftFinger: {224, 56}, slotFeet: {200, 64},
}

// inventory is the state of the equipment view.
type inventory struct {
	// equipment are the objects in each slot (Equipment[]).
	equipment [slotCount]int
	// oneHanded says that a party member's left hand holds a two-handed weapon
	// (Party_1Hand[]); npcOneHanded is the same for an NPC shown by look (NPC_1Hand).
	oneHanded    [partyMax]bool
	npcOneHanded bool
	// showNPC is the NPC whose inventory look shows, 0 for a party member (ShowNPCInven).
	showNPC int
	// container is the object whose contents the grid shows (D_E709), inventoryScroll its
	// first row (D_07CE), inventoryShown the objects in the grid (D_E70F).
	container       int
	inventoryScroll int
	inventoryShown  [inventorySlots]int
	// invenSize counts the carried objects, weightInven and weightEquip weigh them
	// (InvenSize, WeightInven, WeightEquip).
	invenSize, weightInven, weightEquip int
	// spellPeek is set while the spellbook is shown in the status area (D_EBB5).
	spellPeek bool
}

// combatModeNames are the combat modes, from AICommand (D_07CF).
var combatModeNames = [...]string{"COMMAND", "FRONT", "REAR", "FLANK", "BERSERK", "RETREAT", "ASSAULT"}

// clearStatus fills the status area with the background.
func (g *Game) clearStatus() {
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(statusX0, statusY0, statusX1, statusY1)
}

// centreName prints a name centred on row of the party clip.
func (g *Game) centreName(name string, row int) {
	g.gotoxy((statusTextColumns-len(name))/2, row)
	g.printf("%s\n", name)
}

// refreshCombatMode shows the combat mode of the member in the equipment view.
// C: seg_155D:01BC (STAT_refreshComMode)
func (g *Game) refreshCombatMode() {
	saved := g.current
	g.setClip(clipParty)
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(combatModeX0, combatModeY0, combatModeX1, statusY1)
	name := combatModeNames[g.NPCs[g.Party[g.statusMember]].CombatMode-AICommand]
	g.gotoxy((combatModeWidth-len(name))/2+combatModeColumn, combatModeRow)
	g.printf("%s", name)
	g.current = saved
}

// portraitView shows party member i's portrait and statistics.
// C: seg_155D:028A (C_155D_028A)
// C: seg_2337:04E1 (GetLevel), inlined
func (g *Game) portraitView(i int) {
	g.clearStatus()
	g.setClip(clipParty)
	g.centreName(cString(g.Names[i][:]), 0)
	n := int(g.Party[i])
	g.drawPortrait(n, portraitX, portraitY)
	npc := &g.NPCs[n]
	g.gotoxy(statsColumn, 2)
	g.printf("STR:%d\n", int(npc.Strength))
	g.gotoxy(statsColumn, 3)
	g.printf("DEX:%d\n", int(npc.Dexterity))
	g.gotoxy(statsColumn, 4)
	g.printf("INT:%d\n", int(npc.Intelligence))
	g.gotoxy(statsColumn, 6)
	g.printf("Magic")
	g.gotoxy(statsColumn, 7)
	g.printf("%2d%/%d", int(npc.Magic), g.maxMagic(n))
	g.gotoxy(statsColumn, 8)
	g.printf("Health")
	g.gotoxy(statsColumn, 9)
	g.hitPointsColour(n)
	g.printf("%3d", int(npc.HitPoints))
	g.current.fg = g.colours.text
	g.printf("%/%d", g.maxHitPoints(n))
	g.gotoxy(statsColumn, 10)
	g.printf("Lev%/Exp")
	g.gotoxy(statsColumn, 11)
	g.printf("%d%/%d", int(npc.Level), int(npc.Experience))
	g.drawStatusButtons(tileStatusEquipButton)
	g.setClip(clipMessage)
}

// hitPointsColour colours the hit points of character n: poisoned or low.
func (g *Game) hitPointsColour(n int) {
	if g.NPCs[n].Has(Poisoned) {
		g.current.fg = g.colours.poisoned
	} else if g.NPCs[n].HitPoints < lowHitPoints {
		g.current.fg = g.colours.highlight
	}
}

// drawStatusButtons draws the buttons under the portrait and equipment views; switchTile
// is the button to the other view.
func (g *Game) drawStatusButtons(switchTile int) {
	if g.statusMember > 0 {
		g.gr.DrawTile(tileStatusPrevious, buttonPrevious, buttonsY)
	}
	g.gr.DrawTile(tileStatusPortraitButton, buttonParty, buttonsY)
	g.gr.DrawTile(switchTile, buttonSwitch, buttonsY)
	if int(g.PartySize)-1 > g.statusMember {
		g.gr.DrawTile(tileStatusNext, buttonNext, buttonsY)
	}
}

// equipTiles are the tiles that go in the right hand (D_07DD).
var equipTiles = []int{
	544, 545, 547, 548, 549, 550, 551, 554, 559, 560, 568, 596, 598, 597, 601, 610,
	611, 612, 624, 625, 626, 627, 628, 629, 633, 637, 638, 639, 640, 641, 674, 675, 697,
}

// slotTiles are the tile ranges worn in each slot other than the right hand, checked in order.
var slotTiles = []struct{ first, last, slot int }{
	{538, 539, slotFeet},      // TIL_21A, TIL_21B
	{600, 600, slotRing},      // TIL_258
	{893, 895, slotRing},      // TIL_37D..37F
	{537, 537, slotNeck},      // TIL_219
	{592, 594, slotNeck},      // TIL_250..252
	{535, 535, slotNeck},      // TIL_217
	{257, 257, slotNeck},      // TIL_101
	{512, 519, slotHead},      // TIL_200..207
	{528, 534, slotChest},     // TIL_210..216
	{536, 536, slotChest},     // TIL_218
	{652, 652, slotChest},     // TIL_28C
	{654, 654, slotChest},     // TIL_28E
	{669, 669, slotChest},     // TIL_29D
	{599, 599, slotChest},     // TIL_257
	{552, 553, slotTwoHanded}, // TIL_228, 229
	{561, 561, slotTwoHanded}, // TIL_231
	{565, 565, slotTwoHanded}, // TIL_235
	{555, 558, slotTwoHanded}, // TIL_22B..22E
	{520, 527, slotLeftHand},  // TIL_208..20F
	{546, 546, slotLeftHand},  // TIL_222
}

// equipSlot returns the slot object n is worn in, or noSlot.
// C: seg_155D:0516 (STAT_GetEquipSlot)
func (g *Game) equipSlot(n int) int {
	t := g.tileFrame(n)
	for _, r := range slotTiles {
		if t >= r.first && t <= r.last {
			return r.slot
		}
	}
	for _, e := range equipTiles {
		if e == t {
			return slotRightHand
		}
	}
	return noSlot
}

// lightTypes weigh a tenth of their weight each (D_081F).
var lightTypes = []ObjectType{ObjGold, ObjBlackPearl3, ObjBloodMoss3, ObjGarlic3, ObjGinseng3,
	ObjMandrake3, ObjNightshade3, ObjSpidersilk3, ObjAsh3}

// unliftable is the weight that means an object cannot be carried.
const (
	unliftableWeight = 255
	unliftable       = 10000
	litTorchFrame    = 1
)

// weightOf returns the weight of object n, in tenths of a stone.
// C: seg_155D:0661 (GetWeight)
func (g *Game) weightOf(n int) int {
	o := g.object(n)
	t := o.Type()
	qty := 1
	switch quantityType(t) {
	case countGold:
		qty = int(o.Amount)
	case countQuantity:
		if t != ObjTorch || o.Frame() != litTorchFrame {
			qty = o.Quantity()
		}
	}
	weight := int(g.TypeWeight[t]) * qty
	for _, l := range lightTypes {
		if l == t {
			weight /= 10
			break
		}
	}
	if weight == unliftableWeight {
		weight = unliftable
	}
	return weight
}

// encumbrance returns the weight of what object n holds.
// C: seg_155D:0707 (Encumbrance)
func (g *Game) encumbrance(n int) int {
	weight := 0
	for _, i := range g.heldBy(n) {
		weight += g.weightOf(i)
	}
	return weight
}

// weighInventory counts and weighs what n carries and, with fillSlots, fills the slots of
// equipment[].
// C: seg_155D:0748 (C_155D_0748)
// C: seg_155D:07E0 (C_155D_07E0)
func (g *Game) weighInventory(n int, fillSlots bool) {
	g.invenSize, g.weightInven = 0, 0
	if n < npcCount {
		g.weightEquip = 0
	}
	member := 0
	for member < partyMax/2 && int(g.Party[member]) != n {
		member++
	}
	for i := g.FindInv(n); i >= 0; i = g.NextInv() {
		if g.object(i).CoordUse() != Equipped {
			g.weightInven += g.weightOf(i) + g.encumbrance(i)
			g.invenSize++
			continue
		}
		g.weightEquip += int(g.TypeWeight[g.object(i).Type()])
		if fillSlots {
			g.placeEquipped(i, member, g.showNPC == 0)
		}
	}
}

// placeEquipped puts equipped object i in its slot of equipment[]; member is the party
// member wearing it when inParty.
func (g *Game) placeEquipped(i, member int, inParty bool) {
	slot := g.equipSlot(i)
	switch {
	case slot == slotTwoHanded:
		slot = slotRightHand
		g.equipment[slotLeftHand] = twoHandedMark
		if inParty {
			g.oneHanded[member] = true
		} else {
			g.npcOneHanded = true
		}
	case slot == slotRightHand && g.equipment[slotRightHand] != emptySlot:
		slot = slotLeftHand
	case slot == slotLeftHand && g.equipment[slotLeftHand] != emptySlot:
		slot = slotRightHand
	case slot == slotRing:
		slot = slotRightFinger
		if g.equipment[slotRightFinger] != emptySlot {
			slot = slotLeftFinger
		}
	}
	// fix: an object that fits no slot was stored at Equipment[-1]
	if slot >= 0 {
		g.equipment[slot] = i
	}
}

// leftHandTaken says that the left hand shown holds a two-handed weapon.
func (g *Game) leftHandTaken() bool {
	if g.showNPC == 0 {
		return g.oneHanded[g.statusMember]
	}
	return g.npcOneHanded
}

// drawEquipment draws the slots, offset by (x, y).
// C: seg_155D:08F4 (C_155D_08F4)
func (g *Game) drawEquipment(x, y int) {
	for slot, place := range slotPlaces {
		px, py := place[0]+x, place[1]+y
		n := g.equipment[slot]
		switch {
		case slot == slotLeftHand && g.leftHandTaken():
			g.gr.DrawTile(tileOneHanded, px, py)
		case n != emptySlot:
			g.drawTileOver(g.tileFrame(n), tileStatusSlot, px, py)
		default:
			g.gr.DrawTile(tileEmptySlot, px, py)
		}
	}
}

// roundWeight turns tenths of stones into stones, rounding half up (and anything above 0
// to at least 1).
// C: seg_155D:0CB6 (C_155D_0CB6)
func roundWeight(w int) int {
	stones, tenths := w/10, w%10
	if stones != 0 {
		if tenths >= 5 {
			stones++
		}
		return stones
	}
	if tenths != 0 {
		return 1
	}
	return 0
}

// isCarried says that an object of a container is shown in the grid.
func isCarried(o *Object) bool {
	return o.CoordUse() == InInventory || o.CoordUse() == Contained
}

// drawInventory draws the equipment, the inventory grid and the weights.
// C: seg_155D:0CF5 (C_155D_0CF5)
func (g *Game) drawInventory() {
	if g.StatusDisplay == cmdStatusParty && !g.spellPeek {
		return
	}
	saved := g.current
	g.setClip(clipParty)
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(inventoryX, inventoryClearY0, inventoryClearX1, inventoryClearY1)
	g.drawEquipment(0, 0)
	const rowStep = inventoryColumns
	if g.inventoryScroll >= rowStep && g.inventoryScroll+inventorySlots-rowStep >= g.invenSize {
		g.inventoryScroll -= rowStep
	}
	item := g.FindInv(g.container)
	for skipped := 0; g.inventoryScroll > skipped; item = g.NextInv() {
		if isCarried(g.object(item)) {
			skipped++
		}
	}
	if g.container < npcCount {
		t := ObjectType(g.NPCs[g.container].OrigShape & typeMask)
		const south = 4
		g.drawTileOver(directionFrame(t, south)+int(g.BaseTile[t]), tileStatusSlot, containerX, containerY)
	} else {
		g.drawTileOver(g.tileFrame(g.container), tileStatusSlot, containerX, containerY)
	}
	slot := 0
	for ; item >= 0 && slot < inventorySlots; item = g.NextInv() {
		if isCarried(g.object(item)) {
			g.drawObjectCount(item, inventoryX+slot%inventoryColumns*tileSize, inventoryY+slot/inventoryColumns*tileSize)
			g.inventoryShown[slot] = item
			slot++
		}
	}
	for ; slot < inventorySlots; slot++ {
		g.inventoryShown[slot] = emptySlot
		g.gr.DrawTile(tileEmptySlot, inventoryX+slot%inventoryColumns*tileSize, inventoryY+slot/inventoryColumns*tileSize)
	}
	g.gotoxy(scrollArrowColumn, scrollDownRow)
	if g.invenSize > g.inventoryScroll+inventorySlots {
		g.putch(downArrowChar)
	} else {
		g.putch(' ')
	}
	g.gotoxy(scrollArrowColumn, scrollUpRow)
	if g.inventoryScroll == 0 {
		g.putch(' ')
	} else {
		g.putch(upArrowChar)
	}
	owner := g.container
	for owner >= npcCount {
		owner = g.object(owner).Pos.Owner()
	}
	strength := int(g.NPCs[owner].Strength)
	g.gotoxy(equipWeightColumn, weightRow)
	g.printf("      ")
	g.gotoxy(equipWeightColumn, weightRow)
	g.printf("%d%/%ds", roundWeight(g.weightEquip), strength)
	g.gotoxy(inventoryWeightColumn, weightRow)
	g.printf("      ")
	g.gotoxy(inventoryWeightColumn, weightRow)
	g.printf("%d%/%ds", roundWeight(g.weightInven+g.weightEquip), strength<<1)
	g.current = saved
}

// refreshInventory recounts what n holds and draws the grid.
// C: seg_155D:1022 (C_155D_1022)
func (g *Game) refreshInventory(n int) {
	g.inventoryShown = [inventorySlots]int{}
	if n < npcCount {
		g.equipment = [slotCount]int{}
	}
	g.weighInventory(n, true)
	g.drawInventory()
}

// equipmentView shows the equipment and inventory of the status member, or of showNPC.
// C: seg_155D:1065 (C_155D_1065)
func (g *Game) equipmentView() {
	g.inventoryScroll = 0
	g.clearStatus()
	g.setClip(clipParty)
	if g.showNPC == 0 {
		g.centreName(cString(g.Names[g.statusMember][:]), 0)
	}
	g.drawDoll(dollX, dollY)
	g.gotoxy(equipLabelColumn, weightRow)
	g.printf("E:")
	g.gotoxy(inventoryLabelColumn, weightRow)
	g.printf("I:")
	if g.showNPC == 0 {
		g.drawStatusButtons(tileSwitchToEquip)
		if g.statusMember != 0 {
			g.gr.DrawTile(tileCombatMode, buttonCombat, buttonsY)
		}
		g.container = int(g.Party[g.statusMember])
	} else {
		g.container = g.showNPC
	}
	g.oneHanded[g.statusMember] = false
	g.refreshInventory(g.container)
	g.refreshCombatMode()
	g.setClip(clipMessage)
}

// scrollInventory scrolls the grid down (rows > 0) or up a row.
// C: seg_155D:120A (C_155D_120A)
// C: seg_155D:123E (C_155D_123E)
func (g *Game) scrollInventory(down bool) {
	g.setClip(clipParty)
	switch {
	case down && g.inventoryScroll+inventoryColumns < g.invenSize:
		g.inventoryScroll += inventoryColumns
	case !down && g.inventoryScroll >= inventoryColumns:
		g.inventoryScroll -= inventoryColumns
	default:
		return
	}
	g.drawInventory()
	g.setClip(clipMessage)
}

// pointInventory selects the object of the grid or the container under the pointer.
// C: seg_155D:1267 (C_155D_1267)
func (g *Game) pointInventory() bool {
	x, y := g.PointerX, g.PointerY
	if x >= inventoryX && x < inventoryRight && y >= inventoryY && y < inventoryBottom {
		slot := (x-inventoryX)/tileSize + (y-inventoryY)/tileSize*inventoryColumns
		g.Selection.obj = g.inventoryShown[slot]
		if g.Selection.obj == emptySlot {
			g.Selection.obj = noObject
		}
		g.Selection.x = slot + 1
		return true
	}
	if x >= containerX && x < containerX+tileSize && y >= containerY && y < containerY+tileSize {
		if g.container >= npcCount {
			g.Selection.x = 0
		} else {
			g.Selection.x, g.Selection.y = int(g.MapX), int(g.MapY)
		}
		g.Selection.obj = g.container
		return true
	}
	g.Selection.obj = noObject
	return false
}

// pointEquipment selects the equipped object under the pointer.
// C: seg_155D:130E (C_155D_130E)
func (g *Game) pointEquipment() bool {
	x, y := g.PointerX, g.PointerY
	in := func(x0, y0, x1, y1 int) bool { return x >= x0 && x < x1 && y >= y0 && y < y1 }
	const (
		rightColumn, middleColumn, leftColumn = 176, 200, 224
		handsTop, handsBottom                 = 24, 72
		headY, feetY                          = 16, 64
	)
	switch {
	case in(middleColumn, headY, middleColumn+tileSize, headY+tileSize):
		g.Selection.x = slotHead
	case in(rightColumn, handsTop, rightColumn+tileSize, handsBottom):
		g.Selection.x = (y-handsTop)/tileSize + slotNeck
	case in(leftColumn, handsTop, leftColumn+tileSize, handsBottom):
		g.Selection.x = (y-handsTop)/tileSize + slotChest
	case in(middleColumn, feetY, middleColumn+tileSize, feetY+tileSize):
		g.Selection.x = slotFeet
	default:
		g.Selection.obj = noObject
		return false
	}
	g.Selection.obj = g.equipment[g.Selection.x]
	if g.Selection.obj == emptySlot || g.Selection.obj == twoHandedMark {
		g.Selection.obj = noObject
	}
	return true
}

// slotOf returns the slot object n is in, or noSlot.
// C: seg_155D:13F4 (C_155D_13F4)
func (g *Game) slotOf(n int) int {
	for i, e := range g.equipment {
		if e == n {
			return i
		}
	}
	return noSlot
}

// stormCloakSpell is the spell effect of the storm cloak (SpellFx[13]).
const (
	stormCloakSpell    = 13
	stormCloakDuration = 20
)

// ready puts the selected object on the status member.
// C: seg_155D:144B (C_155D_144B)
func (g *Game) ready() {
	n := int(g.Party[g.statusMember])
	sel := g.Selection.obj
	t := g.object(sel).Type()
	g.printf("Ready-")
	if t == ObjTorch {
		g.printf("torch\n")
	} else {
		g.printf("%s\n", g.objectString(sel))
	}
	slot := g.equipSlot(sel)
	switch {
	case slot == noSlot:
		g.printf("\nCan't be readied!\n")
	case int(g.TypeWeight[t])+g.weightEquip > int(g.NPCs[n].Strength)*10:
		g.printf("\nToo heavy!\n")
	default:
		e := &g.equipment
		switch {
		case slot == slotTwoHanded:
			switch {
			case e[slotRightHand] != emptySlot:
				slot = slotRightHand
			case e[slotLeftHand] != emptySlot:
				slot = slotLeftHand
			default:
				slot = slotRightHand
				e[slotLeftHand] = twoHandedMark
				g.oneHanded[g.statusMember] = true
			}
		case slot == slotRightHand && e[slotRightHand] != emptySlot && e[slotLeftHand] == emptySlot:
			slot = slotLeftHand
		case slot == slotLeftHand && e[slotLeftHand] != emptySlot && e[slotRightHand] == emptySlot:
			slot = slotRightHand
		case slot == slotRing:
			slot = slotRightFinger
			if e[slotRightFinger] != emptySlot {
				slot = slotLeftFinger
			}
		}
		if e[slot] != emptySlot {
			g.printf("\nNo place to put!\n")
			break
		}
		owner := sel
		for owner = g.object(owner).Pos.Owner(); owner >= npcCount; owner = g.object(owner).Pos.Owner() {
		}
		if t == ObjTorch {
			o := g.object(sel)
			if o.setQuantity(o.Quantity() - 1); o.Quantity() == 0 {
				g.DeleteObj(sel)
			}
			e[slot] = g.AddInvObj(uint16(ObjTorch), Equipped, 1, owner)
		} else {
			e[slot] = g.InsertObj(sel, owner, Equipped)
			g.invenSize--
		}
		g.refreshInventory(int(g.Party[g.statusMember]))
		switch g.object(sel).Type() {
		case ObjInvisibilityRing:
			g.object(n).Status |= StatusInvisible
			g.magicalHalo(n, 1)
		case ObjStormCloak:
			g.SpellFx[stormCloakSpell] = stormCloakDuration
			g.magicalHalo(n, 1)
		}
	}
	g.printf("\n%s:\n%>", cString(g.Names[g.Active][:]))
}

// What unready found that stops it.
const (
	unreadyFree     = 0
	unreadyRefused  = 1
	unreadyBurntOut = 2
)

// amuletMsg is D_0DDC[12].
const amuletMsg = "Magical energy prevents you from removing the amulet."

// unreadyCheck refuses to take off the amulet of submission and lets a lit torch burn
// out.
// C: seg_155D:1666 (C_155D_1666)
func (g *Game) unreadyCheck(n int) int {
	o := g.object(n)
	if o.CoordUse() != Equipped {
		return unreadyFree
	}
	switch {
	case o.Type() == ObjAmuletOfSubmission:
		g.printf("\n%s\n", amuletMsg)
		return unreadyRefused
	case o.Type() == ObjTorch && o.Frame() == litTorchFrame:
		g.DeleteObj(n)
		g.printf("\n")
		g.printf(burnOutMsg)
		return unreadyBurntOut
	}
	return unreadyFree
}

// stopWornEffect ends what an equipped ring of invisibility or storm cloak does for
// wearer.
// C: seg_155D:16E7 (C_155D_16E7)
func (g *Game) stopWornEffect(n, wearer int) {
	o := g.object(n)
	if o.CoordUse() != Equipped {
		return
	}
	switch o.Type() {
	case ObjInvisibilityRing:
		g.object(wearer).Status &^= StatusInvisible
		g.magicalHalo(wearer, 1)
	case ObjStormCloak:
		g.SpellFx[stormCloakSpell] = 0
		g.magicalHalo(wearer, 1)
	}
}

// unready takes the selected object off the status member.
// C: seg_155D:1738 (C_155D_1738)
func (g *Game) unready() {
	sel := g.Selection.obj
	if sel == twoHandedMark {
		return
	}
	g.printf("Unready-%s\n", g.objectString(sel))
	t := g.object(sel).Type()
	n := int(g.Party[g.statusMember])
	result := g.unreadyCheck(sel)
	if result != unreadyRefused {
		// fix: the original wrote Equipment[-1] for an object in no slot
		if slot := g.slotOf(sel); slot != noSlot {
			g.equipment[slot] = emptySlot
		}
		if result != unreadyBurntOut {
			g.stopWornEffect(sel, n)
			if g.equipSlot(sel) == slotTwoHanded {
				g.oneHanded[g.statusMember] = false
				g.equipment[slotLeftHand] = emptySlot
			}
			if torch := g.FindInvType(n, ObjTorch, 0); t == ObjTorch && torch >= 0 && g.object(torch).CoordUse() == InInventory {
				g.DeleteObj(sel)
				g.giveObj(n, ObjTorch, 1)
			} else {
				g.object(sel).SetCoordUse(InInventory)
				g.invenSize++
			}
		}
	}
	g.refreshInventory(int(g.Party[g.statusMember]))
	g.printf("\n%s:\n%>", cString(g.Names[g.Active][:]))
}

// drawDoll draws the paper doll at (x, y).
// C: seg_2FC1:1EAF (C_2FC1_1EAF)
func (g *Game) drawDoll(x, y int) {
	g.gr.DrawTile(tileDollTopLeft, x, y)
	g.gr.DrawTile(tileDollTopRight, x+tileSize, y)
	g.gr.DrawTile(tileDollBottomL, x, y+tileSize)
	g.gr.DrawTile(tileDollBottomR, x+tileSize, y+tileSize)
}

// Portraits: PORTRAIT.A holds the NPCs up to portraitsInA, PORTRAIT.B the others and the
// generic wisp, guard and gargoyle, PORTRAIT.Z the avatars.
const (
	portraitsInA    = 98  // 0x62
	wispPortrait    = 192 // 0xc0
	guardPortrait   = 193
	gargoylePortait = 194
)

// drawPortrait draws the portrait of character n at (x, y).
// C: seg_2FC1:1C19 (C_2FC1_1C19)
func (g *Game) drawPortrait(n, x, y int) {
	if n >= firstMonster {
		switch g.object(n).Type() {
		case ObjWisp:
			n = wispPortrait
		case ObjGuard:
			n = guardPortrait
		case ObjGargoyle:
			n = gargoylePortait
		}
	}
	file := "PORTRAIT.A"
	if n == avatar {
		file = "PORTRAIT.Z"
		if n = int(g.PortraitPlusOne) - 1; n < 0 {
			return
		}
	} else {
		if n != 0 {
			n--
		}
		if n >= portraitsInA {
			file = "PORTRAIT.B"
			n -= portraitsInA
		}
	}
	raw, err := fs.ReadFile(g.files, file)
	const offsetBytes = 4
	if err != nil || len(raw) < (n+1)*offsetBytes {
		return
	}
	at := int(binary.LittleEndian.Uint32(raw[n*offsetBytes:]))
	if at >= len(raw) {
		return
	}
	pixels, err := data.Decompress(raw[at:])
	if err != nil || len(pixels) < portraitWidth*portraitHeight {
		return
	}
	g.gr.DrawPixmap(&vga.Pixmap{Width: portraitWidth, Height: portraitHeight, Pixels: pixels}, x, y)
	g.talkPortrait = image.Rect(x, y, x+portraitWidth, y+portraitHeight)
}

// smallFont is the 3x5 font of the counts on inventory tiles, '0'-'9' then 'A'-'P' (D_04E3).
var smallFont = [...]uint16{
	0b111_101_101_101_111_0, // 0
	0b010_010_010_010_010_0, // 1
	0b111_001_111_100_111_0, // 2
	0b111_001_011_001_111_0, // 3
	0b101_101_111_001_001_0, // 4
	0b111_100_111_001_111_0, // 5
	0b111_100_111_101_111_0, // 6
	0b111_001_001_010_010_0, // 7
	0b111_101_111_101_111_0, // 8
	0b111_101_111_001_111_0, // 9
	0b111_101_111_101_101_0, // A
	0b110_101_110_101_110_0, // B
	0b111_100_100_100_111_0, // C
	0b110_101_101_101_110_0, // D
	0b111_100_110_100_111_0, // E
	0b111_100_110_100_100_0, // F
	0b111_100_101_101_111_0, // G
	0b101_101_111_101_101_0, // H
	0b111_010_010_010_111_0, // I
	0b001_001_001_101_111_0, // J
	0b101_101_110_101_101_0, // K
	0b100_100_100_100_111_0, // L
	0b101_111_101_101_101_0, // M
	0b000_000_111_101_101_0, // N
	0b000_000_111_101_111_0, // O
	0b111_101_111_100_100_0, // P
}

// digitColumns are the x of each digit of a count, by its number of digits (D_0517).
var digitColumns = [4][4]int{{7}, {9, 4}, {11, 7, 3}, {13, 9, 5, 1}}

// small font geometry
const (
	smallTop       = 11
	smallBottom    = 16
	smallWidth     = 3
	smallTopBit    = 1 << 15
	smallLetterA   = 10 // the index of 'A' in smallFont
	maxCountDigits = 4
)

// drawSmallChar draws character c of smallFont at column x of the back buffer's tile.
// C: seg_0C9C:0C10 (C_0C9C_0C10)
func (g *Game) drawSmallChar(c, x int) {
	g.gr.Offscreen = true
	bits := smallFont[c]
	g.gr.SetColor(g.colours.text)
	for y := smallTop; y < smallBottom; y++ {
		for i := x; i < x+smallWidth; i++ {
			if bits&smallTopBit != 0 {
				g.gr.PutPixel(i, y)
			}
			bits <<= 1
		}
	}
	g.gr.Offscreen = false
}

// drawCount draws tile with the count v at (x, y).
// C: seg_0C9C:0C7D (C_0C9C_0C7D)
func (g *Game) drawCount(tile, v, x, y int) {
	g.gr.DrawTileOffscreen(tileStatusSlot, 0, 0)
	g.gr.DrawTileOffscreen(tile, 0, 0)
	digits := 1
	for limit := 10; v >= limit; limit *= 10 {
		digits++
	}
	if digits > maxCountDigits {
		digits = 0
	}
	for i := range digits {
		g.drawSmallChar(v%10, digitColumns[digits-1][i])
		v /= 10
	}
	g.gr.CopyBackBufferToScreen(0, 0, tileSize-1, tileSize-1, x, y)
}

// drawObjectCount draws an object of the inventory with its count, or a spell's letter.
// C: seg_0C9C:0D31 (C_0C9C_0D31)
func (g *Game) drawObjectCount(n, x, y int) {
	o := g.object(n)
	tile := g.tileFrame(n)
	switch {
	case o.Type() == ObjEffect:
		const letterColumn = 7
		g.gr.DrawTileOffscreen(tileStatusSlot, 0, 0)
		g.gr.DrawTileOffscreen(tile, 0, 0)
		g.drawSmallChar(o.Quality()+smallLetterA-1, letterColumn)
		g.gr.CopyBackBufferToScreen(0, 0, tileSize-1, tileSize-1, x, y)
	case quantityType(o.Type()) == countGold:
		g.gr.DrawTile(tileStatusSlot, x, y)
		g.drawCount(tile, int(o.Amount), x, y)
	case quantityType(o.Type()) == countQuantity:
		g.gr.DrawTile(tileStatusSlot, x, y)
		g.drawCount(tile, o.Quantity(), x, y)
	default:
		g.drawTileOver(tile, tileStatusSlot, x, y)
	}
}

// the status cursor's places: columns (D_054B) and, by row, y (D_0559)
var (
	cursorColumns = [7]int{176, 200, 224, 248, 264, 280, 296}
	cursorRows    = [3][7]int{
		{24, 16, 24, 32, 32, 32, 32},
		{40, 40, 40, 48, 48, 48, 48},
		{56, 64, 56, 64, 64, 64, 64},
	}
)

// status cursor geometry
const (
	cursorColumnsCount = 7
	cursorRowsCount    = 3
	cursorButtonsRow   = 3
	cursorOnContainer  = -1 // the row of the container icon above the grid
	cursorLastButton   = 4
	cursorDollColumns  = 3
	cursorDollStepX    = 24
	cursorDollStepY    = 20
	cursorGridX        = 200
	cursorGridY        = 32
	tileStatusCursor   = tileAimTarget
)

// drawStatusCursor draws (mode 1), erases (mode 0) or points at (mode 2) the cursor in
// the equipment and portrait views.
// C: seg_0C9C:1AE5 (C_0C9C_1AE5)
func (g *Game) drawStatusCursor(mode int) {
	if g.SelectMode != 2 {
		return
	}
	if g.StatusDisplay != cmdStatusEquipment && g.StatusDisplay != cmdStatusPortrait {
		return
	}
	icons := [5]int{tileStatusPrevious, tileStatusPortraitButton, tileSwitchToEquip, tileStatusNext, tileCombatMode}
	if g.statusMember == 0 {
		icons[0], icons[4] = tileStatusSlot, tileStatusSlot
	}
	if int(g.PartySize)-1 == g.statusMember {
		icons[3] = tileStatusSlot
	}
	if g.StatusDisplay == cmdStatusPortrait {
		icons[2] = tileStatusEquipButton
		g.statusCursorY = cursorButtonsRow
		g.statusCursorX = min(g.statusCursorX, cursorButtonsRow)
	}
	var x, y int
	cursorX, cursorY := g.statusCursorX, g.statusCursorY
	g.pointerOverUI = true
	switch {
	case cursorY < 0:
		x, y = containerX, containerY
		g.PointerX, g.PointerY = x, y
	case cursorY > cursorRowsCount-1:
		g.statusCursorX = min(g.statusCursorX, cursorLastButton)
		cursorX = g.statusCursorX
		x, y = cursorX*tileSize+buttonPrevious, buttonsY
		g.PointerX, g.PointerY = x, y
	case cursorX < cursorDollColumns:
		g.PointerX, g.PointerY = cursorX*cursorDollStepX+buttonPrevious, cursorY*cursorDollStepY+partyRowY
	default:
		g.PointerX, g.PointerY = cursorX*tileSize+cursorGridX, cursorY*tileSize+cursorGridY
	}
	if cursorY >= 0 && cursorY < cursorRowsCount {
		x, y = cursorColumns[cursorX], cursorRows[cursorY][cursorX]
	}
	switch mode {
	case 1:
		g.gr.DrawTile(tileStatusCursor, x, y)
	case 0:
		selected := g.mouseSelection()
		switch {
		case cursorY > cursorRowsCount-1:
			g.gr.DrawTile(icons[cursorX], x, y)
		case (selected == cmdSelectEquipment || selected == cmdSelectInventory) && g.Selection.obj >= 0:
			if cursorY < 0 {
				t := g.object(g.Selection.obj).Type()
				const south = 4
				g.drawTileOver(directionFrame(t, south)+int(g.BaseTile[t]), tileStatusSlot, x, y)
			} else {
				g.drawObjectCount(g.Selection.obj, x, y)
			}
		case cursorY == 1 && cursorX == 2 && g.leftHandTaken():
			g.gr.DrawTile(tileOneHanded, x, y)
		default:
			g.gr.DrawTile(tileEmptySlot, x, y)
		}
	case 2:
		g.PointerX, g.PointerY = x, y
	}
}

// halo silhouettes: the colours MagicalHalo outlines a character in
const (
	haloOn  = 6 << 10
	haloOff = 4 << 10
)

// magicalHalo flashes an outline around character n when it is in view (on: blue).
// C: seg_2FC1:00B0 (MagicalHalo)
func (g *Game) magicalHalo(n, on int) {
	o := g.object(n)
	if o.CoordUse() != OnMap {
		return
	}
	deltaX := (o.Pos.X - int(g.MapX) + viewCenterX) & coordMask
	deltaY := (o.Pos.Y - int(g.MapY) + viewCenterY) & coordMask
	tile := g.tileFrame(n) + haloOff
	if on != 0 {
		tile = g.tileFrame(n) + haloOn
	}
	if deltaX < viewTilesX && deltaY < viewTilesY {
		const overlay = 2
		g.showObject(tile, deltaX, deltaY, overlay)
		g.screenFade = true
		g.drawMap(true)
		g.ShowObjects()
		g.screenFade = true
		g.drawMap(true)
	}
}
