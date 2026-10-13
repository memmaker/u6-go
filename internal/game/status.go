package game

// The status area at the right of the screen: the party list, a member's portrait or
// equipment (seg_155D).

// The party column: each member's picture with a health bar under it, on the canvas below
// the shown screen; layout.go shows it along the screen's right border.
const (
	partyListRows   = 9 // how many members the column shows at once
	partyColumnX    = 8
	partyColumnY    = 208
	partyRowHeight  = 21
	healthBarHeight = 2
	healthBarGap    = 1
	healthBarColour = 2 // green
	healthBarEmpty  = 8 // dark grey
)

// tileStatusSlot is the background a party member's picture is drawn on (TIL_19B).
const tileStatusSlot = 411

// status is what the status area shows.
type status struct {
	// StatusDisplay is the command whose view is shown: cmdStatusParty, cmdStatusPortrait,
	// cmdStatusEquipment or cmdStatusContainer.
	StatusDisplay int
	// statusMember is the party member shown (D_04B3), partyScroll the first one in the
	// party list (D_07CC).
	statusMember int
	partyScroll  int
	// InConversation is set while the player talks to someone.
	InConversation bool
}

// RefreshStatus draws the status area again when it changed.
// C: seg_0A33:1AB7 (RefreshStatus)
func (g *Game) RefreshStatus() {
	if g.StatusDirty {
		if g.SoloMember >= 0 {
			g.statusMember = int(g.SoloMember)
		}
		switch g.StatusDisplay {
		case cmdStatusParty:
			g.partyView(g.partyScroll)
		case cmdStatusPortrait:
			g.portraitView(g.statusMember)
		case cmdStatusEquipment:
			g.equipmentView()
		case cmdStatusContainer:
			g.showCharacter(int(g.Party[g.statusMember]))
		}
		g.drawStatusCursor(1)
	}
	g.StatusDirty = false
	g.refreshWindows()
}

// partyView draws the sky and the party column from member first.
// C: seg_155D:000C (C_155D_000C), the port's party column
func (g *Game) partyView(first int) {
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(statusX0, statusY0, statusX1, statusY1)
	g.drawSky()
	g.gr.FillRect(partyColumnX, partyColumnY, partyColumnX+tileSize-1, partyColumnY+partyListRows*partyRowHeight-1)
	for i := first; i < int(g.PartySize) && i < first+partyListRows; i++ {
		n := int(g.Party[i])
		t := ObjectType(g.NPCs[n].OrigShape & typeMask)
		const south = 4
		tile := directionFrame(t, south) + int(g.BaseTile[t])
		y := partyColumnY + (i-first)*partyRowHeight
		g.drawTileOver(tile, tileStatusSlot, partyColumnX, y)
		npc := &g.NPCs[n]
		colour := byte(healthBarColour)
		if npc.Has(Poisoned) {
			colour = g.colours.poisoned
		} else if npc.HitPoints < lowHitPoints {
			colour = g.colours.highlight
		}
		barY := y + tileSize + healthBarGap
		g.gr.SetColor(healthBarEmpty)
		g.gr.FillRect(partyColumnX, barY, partyColumnX+tileSize-1, barY+healthBarHeight-1)
		if full := g.maxHitPoints(n); npc.HitPoints > 0 {
			g.gr.SetColor(colour)
			g.gr.FillRect(partyColumnX, barY, partyColumnX+min(int(npc.HitPoints), full)*tileSize/full-1, barY+healthBarHeight-1)
		}
	}
	g.setClip(clipMessage)
}

// drawTileOver draws tile over background at (x, y).
// C: seg_0C9C:0BCE (C_0C9C_0BCE)
// C: seg_0C9C:0E57 (C_0C9C_0E57), its wrapper
func (g *Game) drawTileOver(tile, background, x, y int) {
	g.gr.DrawTileOffscreen(background, 0, 0)
	g.gr.DrawTileOffscreen(tile, 0, 0)
	g.gr.CopyBackBufferToScreen(0, 0, tileSize-1, tileSize-1, x, y)
}

// SetActive makes party member i the one the player controls.
// C: seg_0A33:1B36 (SetActive)
func (g *Game) SetActive(i int) {
	if int(g.Active) == i {
		return
	}
	g.Active = uint8(i)
	p := g.object(int(g.Party[i])).Pos
	g.MapX, g.MapY, g.MapZ = int16(p.X), int16(p.Y), int16(p.Z)
	g.enterArea()
	g.worldShown = true
	g.promptFlag = true
}

// SetPartyMode ends solo mode: the leader is commanded, the others follow.
// C: seg_0A33:1BB2 (SetPartyMode)
func (g *Game) SetPartyMode() {
	if g.SoloMember < 0 {
		g.NPCs[g.Party[0]].Mode = AIMode(g.SavedMode)
	} else {
		g.NPCs[g.Party[g.SoloMember]].Mode = AIMode(g.SavedMode)
	}
	g.SavedMode = int8(g.NPCs[g.Party[0]].Mode)
	g.NPCs[g.Party[0]].Mode = AICommand
	for i := 1; i < int(g.PartySize); i++ {
		n := g.Party[i]
		if !g.object(int(n)).Has(StatusCharmed) {
			g.NPCs[n].Mode = AIFollow
		}
	}
	g.SetActive(0)
	g.SoloMember = -1
	g.drawStatusCursor(0)
	g.soloLeader = 0
}
