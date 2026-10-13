package game

// The status area at the right of the screen: the party list, a member's portrait or
// equipment (seg_155D).

// partyListRows is how many party members the party list shows at once.
const partyListRows = 5

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
}

// partyView draws the party list from member first.
// C: seg_155D:000C (C_155D_000C)
func (g *Game) partyView(first int) {
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(168, 6, 312, 103)
	g.drawSky()
	g.setClip(clipScreen)
	const upArrow, downArrow = 24, 25
	if first != 0 {
		g.gotoxy(21, 3)
		g.putch(upArrow)
	}
	if first < partyMax && int(g.PartySize) > first+partyListRows {
		g.gotoxy(21, 12)
		g.putch(downArrow)
	}
	g.setClip(clipParty)
	const lowHitPoints = 10
	for i := first; i < int(g.PartySize) && i < first+partyListRows; i++ {
		n := int(g.Party[i])
		t := ObjectType(g.NPCs[n].OrigShape & typeMask)
		const south = 4
		tile := directionFrame(t, south) + int(g.BaseTile[t])
		g.drawTileOver(tile, tileStatusSlot, 176, (i-first)*tileSize+24)
		g.gotoxy(14, (i-first)<<1+2)
		if g.NPCs[n].Has(Poisoned) {
			g.current.fg = g.colours.poisoned
		} else if g.NPCs[n].HitPoints < lowHitPoints {
			g.current.fg = g.colours.highlight
		}
		g.printf("%3d\n", int(g.NPCs[n].HitPoints))
		g.current.fg = g.colours.text
		g.gotoxy(3, (i-first)<<1+3)
		g.printf("%s\n", cString(g.Names[i][:]))
	}
	g.setClip(clipMessage)
}

// drawTileOver draws tile over background at (x, y).
// C: seg_0C9C:0BCE (C_0C9C_0BCE)
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
