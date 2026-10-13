package game

import (
	"fmt"
	"image"
)

// The new style UI's gumps (after Nuvie's DollViewGump, ContainerViewGump and
// PortraitViewGump): Nuvie's background art with parts of the status area's picture placed on
// it. A click on a part goes to the status area's place under it, so the game's own clicks,
// commands and checks apply as in the original look.

// gumpPart is what a gump shows at at: a picture of Nuvie's (art) or the status area's
// rectangle from. A click on the part goes to from, at the same offset (kept inside from).
type gumpPart struct {
	art  string
	at   image.Point
	from image.Rectangle
	// see is which of the status area's pixels show: all, or those the art shows through
	see seeThrough
}

// seeThrough picks the pixels of the status area left transparent on a gump.
type seeThrough int

const (
	opaque           seeThrough = iota
	seeBackground               // the background colour
	seeSlotBackdrops            // the backdrops of the doll's slots and the grid's cells
)

// gumpText is a line of the gump font on a gump.
type gumpText struct {
	text string
	at   image.Point
}

// gump is a panel of the new style UI.
type gump struct {
	size  image.Point
	parts []gumpPart
	texts []gumpText
	// rest is the status area's place a click elsewhere on the gump goes to: the name row,
	// which moves a window
	rest image.Point
}

// gumpCanvasPoint maps a point of gump gp to the status area.
func (g *Game) gumpCanvasPoint(gp *gump, pt image.Point) image.Point {
	for i := len(gp.parts) - 1; i >= 0; i-- {
		p := gp.parts[i]
		if p.from.Empty() {
			continue
		}
		size := p.from.Size()
		if p.art != "" {
			size = g.art(p.art).size()
		}
		if offset := pt.Sub(p.at); offset.In(image.Rectangle{Max: size}) {
			last := p.from.Size().Sub(image.Pt(1, 1))
			return p.from.Min.Add(image.Pt(min(offset.X, last.X), min(offset.Y, last.Y)))
		}
	}
	return gp.rest
}

// gumpScreenPoint is where the status area's point pt shows on gump gp at at, or false.
func (gp *gump) screenPoint(pt, at image.Point) (image.Point, bool) {
	for _, p := range gp.parts {
		if p.art == "" && pt.In(p.from) {
			return at.Add(p.at).Add(pt.Sub(p.from.Min)), true
		}
	}
	return image.Point{}, false
}

// The doll gump (doll_bg, 108x136) and the backpack gump right of it (backpack_bg and a
// row for the weight under it), as Nuvie lays them out.
var (
	dollWidget          = image.Pt(26, 16) // the doll's slots, as the status area's from slotsOrigin
	slotsOrigin         = image.Pt(176, 16)
	dollFigure          = image.Pt(42, 32)
	previousArrow       = image.Pt(23, 7)
	nextArrow           = image.Pt(86, 7)
	combatButton        = image.Pt(23, 92)
	heartButton         = image.Pt(23, 108)
	partyButton         = image.Pt(47, 108)
	dollNameRow         = image.Pt(29, 7)
	dollNameWidth       = 58
	combatModeAt        = image.Pt(36, 97)
	combatModeTextWidth = 55
	// equipWeightRight and strengthRight are where the doll's weight numbers end
	equipWeightRight, strengthRight, weightRowY = 70, 87, 82
	backpackAt                                  = image.Pt(108, 0)
	backpackGrid                                = image.Pt(21, 29)
	backpackUp                                  = image.Pt(83, 35)
	backpackDown                                = image.Pt(83, 66)
	backpackDoll                                = image.Pt(18, 101)
	backpackName                                = image.Pt(18, 2)
	backpackWeight                              = image.Pt(52, 103)
	// the portrait gump (portrait_bg, 188x91)
	portraitAt                        = image.Pt(25, 17)
	portraitNext                      = image.Pt(166, 6)
	portraitNameRow                   = image.Pt(29, 6)
	portraitNameWidth                 = 138
	statsRight, pointsRight           = 170, 142
	strengthY, dexterityY, intellectY = 18, 27, 36
	magicY, healthY, levelY           = 55, 64, 73
	portraitDollButton                = image.Pt(0, 67)
)

// statusButton is the status area's button at x under the views.
func statusButton(x int) image.Rectangle { return image.Rect(x, buttonsY, x+tileSize, buttonsBottom) }

// inventoryInfo is what an inventory gump writes and which buttons it has.
type inventoryInfo struct {
	name                                 string
	member                               int // the party member, or -1 for someone looked at
	equipWeight, carriedWeight, strength int
	combatMode                           string
	inBag                                bool
	partySize                            int
}

// inventoryInfo is the status area's equipment view's info.
func (g *Game) inventoryInfo() inventoryInfo {
	owner := g.container
	for owner >= npcCount {
		owner = g.object(owner).Pos.Owner()
	}
	info := inventoryInfo{
		member: -1, partySize: int(g.PartySize), inBag: g.container >= npcCount,
		equipWeight: roundWeight(g.weightEquip), carriedWeight: roundWeight(g.weightInven + g.weightEquip),
		strength: int(g.NPCs[owner].Strength),
	}
	if g.showNPC == 0 {
		info.member, info.name = g.statusMember, cString(g.Names[g.statusMember][:])
		if g.statusMember != 0 {
			info.combatMode = combatModeNames[g.NPCs[g.Party[g.statusMember]].CombatMode-AICommand]
		}
	}
	return info
}

// centred is at moved right to centre text in width.
func centred(at image.Point, text string, width int) image.Point {
	return at.Add(image.Pt((width-textWidth(gumpFont, text))/2, 0))
}

// rightAligned is text ending at right on row y.
func rightAligned(text string, right, y int) gumpText {
	return gumpText{text, image.Pt(right-textWidth(gumpFont, text), y)}
}

// inventoryGump is the doll and backpack gumps of the equipment view described by info.
func inventoryGump(info inventoryInfo) *gump {
	gp := &gump{size: image.Pt(backpackAt.X+backpackWidth, dollHeight), rest: statusPanel.Min}
	add := func(p gumpPart) { gp.parts = append(gp.parts, p) }
	add(gumpPart{art: "doll_bg"})
	add(gumpPart{art: "backpack_bg", at: backpackAt})
	for _, place := range slotPlaces {
		from := image.Rect(place[0], place[1], place[0]+tileSize, place[1]+tileSize)
		add(gumpPart{at: dollWidget.Add(from.Min.Sub(slotsOrigin)), from: from, see: seeSlotBackdrops})
	}
	figure := image.Rect(dollX, dollY, dollX+2*tileSize, dollY+2*tileSize)
	add(gumpPart{at: dollFigure, from: figure, see: seeBackground})
	grid := image.Rect(inventoryX, inventoryY, inventoryRight, inventoryBottom)
	add(gumpPart{at: backpackAt.Add(backpackGrid), from: grid, see: seeSlotBackdrops})
	add(gumpPart{art: "cont_up_btn_up", at: backpackAt.Add(backpackUp), from: image.Rect(scrollArrowX, scrollUpY0, scrollArrowRight, scrollUpY1)})
	add(gumpPart{art: "cont_down_btn_up", at: backpackAt.Add(backpackDown), from: image.Rect(scrollArrowX, scrollDnY0, scrollArrowRight, scrollDnY1)})
	if info.inBag {
		add(gumpPart{art: "cont_doll_btn_up", at: backpackAt.Add(backpackDoll), from: image.Rect(containerX, containerY, containerX+tileSize, containerY+tileSize)})
	}
	if info.member >= 0 {
		if info.member > 0 {
			add(gumpPart{art: "left_arrow", at: previousArrow, from: statusButton(buttonPrevious)})
			add(gumpPart{art: "combat_btn_up", at: combatButton, from: statusButton(buttonCombat)})
		}
		if info.member < info.partySize-1 {
			add(gumpPart{art: "right_arrow", at: nextArrow, from: statusButton(buttonNext)})
		}
		add(gumpPart{art: "heart_btn_up", at: heartButton, from: statusButton(buttonSwitch)})
		add(gumpPart{art: "party_btn_up", at: partyButton, from: statusButton(buttonParty)})
	}
	gp.texts = []gumpText{
		{info.name, centred(dollNameRow, info.name, dollNameWidth)},
		rightAligned(fmt.Sprint(info.equipWeight), equipWeightRight, weightRowY),
		rightAligned(fmt.Sprint(info.strength), strengthRight, weightRowY),
		{info.name, backpackAt.Add(backpackName)},
		{fmt.Sprintf("I:%d/%ds", info.carriedWeight, info.strength*2), backpackAt.Add(backpackWeight)},
	}
	if info.combatMode != "" {
		gp.texts = append(gp.texts, gumpText{info.combatMode, centred(combatModeAt, info.combatMode, combatModeTextWidth)})
	}
	return gp
}

// The gumps' sizes: their backgrounds'.
const (
	dollHeight         = 136
	backpackWidth      = 111
	portraitGumpWidth  = 188
	portraitGumpHeight = 91
)

// portraitGump is the portrait gump of party member i.
func (g *Game) portraitGump(i int) *gump {
	n := int(g.Party[i])
	npc := &g.NPCs[n]
	gp := &gump{size: image.Pt(portraitGumpWidth, portraitGumpHeight), rest: statusPanel.Min}
	gp.parts = []gumpPart{
		{art: "portrait_bg"},
		{at: portraitAt, from: image.Rect(portraitX, portraitY, portraitX+portraitWidth, portraitY+portraitHeight)},
		{art: "cont_doll_btn_up", at: portraitDollButton, from: statusButton(buttonSwitch)},
	}
	if i > 0 {
		gp.parts = append(gp.parts, gumpPart{art: "left_arrow", at: previousArrow, from: statusButton(buttonPrevious)})
	}
	if i < int(g.PartySize)-1 {
		gp.parts = append(gp.parts, gumpPart{art: "right_arrow", at: portraitNext, from: statusButton(buttonNext)})
	}
	name := cString(g.Names[i][:])
	number := func(v, right, y int) gumpText { return rightAligned(fmt.Sprint(v), right, y) }
	gp.texts = []gumpText{
		{name, centred(portraitNameRow, name, portraitNameWidth)},
		number(int(npc.Strength), statsRight, strengthY),
		number(int(npc.Dexterity), statsRight, dexterityY),
		number(int(npc.Intelligence), statsRight, intellectY),
		number(int(npc.Magic), pointsRight, magicY),
		number(g.maxMagic(n), statsRight, magicY),
		number(int(npc.HitPoints), pointsRight, healthY),
		number(g.maxHitPoints(n), statsRight, healthY),
		number(int(npc.Level), pointsRight, levelY),
		number(int(npc.Experience), statsRight, levelY),
	}
	return gp
}

// statusGump is the gump the status area shows as in the new style UI, or nil.
func (g *Game) statusGump() *gump {
	if !g.newStyle() || g.spellPeek {
		return nil
	}
	switch g.StatusDisplay {
	case cmdStatusEquipment, cmdStatusContainer:
		return inventoryGump(g.inventoryInfo())
	case cmdStatusPortrait:
		return g.portraitGump(g.statusMember)
	}
	return nil
}

// drawGump draws gp with its corner at at, the status area's pixels read by pixel.
func (g *Game) drawGump(gp *gump, at image.Point, pixel func(x, y int) byte) {
	screen := image.Rect(0, 0, g.screenWidth, g.screenHeight)
	for _, p := range gp.parts {
		if p.art != "" {
			g.drawWhole(g.art(p.art), at.Add(p.at))
			continue
		}
		for y := p.from.Min.Y; y < p.from.Max.Y; y++ {
			for x := p.from.Min.X; x < p.from.Max.X; x++ {
				c := pixel(x, y)
				if g.seesThrough(p, x, y, c) {
					continue
				}
				if pt := at.Add(p.at).Add(image.Pt(x, y).Sub(p.from.Min)); pt.In(screen) {
					g.shown[pt.Y*g.screenWidth+pt.X] = c
				}
			}
		}
	}
	for _, t := range gp.texts {
		g.drawNuvieText(gumpFont, t.text, at.Add(t.at))
	}
}

// seesThrough says that the status area's pixel c at (x, y) is left out of part p.
func (g *Game) seesThrough(p gumpPart, x, y int, c byte) bool {
	switch p.see {
	case seeBackground:
		return c == g.colours.background
	case seeSlotBackdrops:
		cell := (y-p.from.Min.Y)%tileSize*tileSize + (x-p.from.Min.X)%tileSize
		return c == g.slotBackdrop(tileStatusSlot)[cell] || c == g.slotBackdrop(tileEmptySlot)[cell]
	}
	return false
}

// slotBackdrop is the picture of tile, drawn over the background colour, as the status
// area draws it under a thing or for an empty cell.
func (g *Game) slotBackdrop(tile int) []byte {
	if b := g.backdrops[tile]; b != nil {
		return b
	}
	b := make([]byte, tileSize*tileSize)
	for i := range b {
		b[i] = g.colours.background
	}
	g.gr.DrawTileOn(b, tileSize, tile, 0, 0)
	if g.backdrops == nil {
		g.backdrops = map[int][]byte{}
	}
	g.backdrops[tile] = b
	return b
}
