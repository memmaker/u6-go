package game

import "github.com/memmaker/u6-go/internal/vga"

// The corners of the spellbook page (indexes of D_EBAD).
const (
	cornerPreviousCircle = iota
	cornerLeft
	cornerRight
	cornerNextCircle
)

// What a corner of the page turns to.
const (
	noCorner      = -1
	turnToCircle  = 1
	turnToPage    = 2
	pageRows      = 8
	arrowChar     = 26  // the arrow before the selected spell
	allSpellsRead = 255 // the quality of the spell object that holds every spell
	noReagentCap  = 99
)

// circleNames are the names of the circles (D_0DC4).
var circleNames = [circleCount]string{"1ST", "2ND", "3RD", "4TH", "5TH", "6TH", "7TH", "8TH"}

// Where the spellbook is drawn.
const (
	bookX0, bookY0, bookX1, bookY1 = 168, 6, 311, 103
	pageX0, pageY0, pageX1, pageY1 = 176, 14, 303, 79
	cornerY                        = 88
	quantityColumn                 = 13
)

// refreshSpellList gathers the spells of the spellbooks and finds the page of the
// selected spell.
// C: seg_1944:0321 (SPELL_RefreshList)
func (g *Game) refreshSpellList() {
	g.spellList = [circleCount]uint16{}
	every := false
	for _, book := range g.spellbooks {
		if book == noObject {
			continue
		}
		first := 0
		for i := g.FindInv(book); i >= 0; i = g.NextInv() {
			s := g.object(i).Quality()
			if s == allSpellsRead {
				every = true
				break
			}
			if !g.hasSpell(s/circleSlots, s%circleSlots) {
				if first == 0 {
					first = s
				}
				g.spellList[s/circleSlots] |= 1 << (s % circleSlots)
			}
		}
		if g.object(book).Quality() == 0 && g.FindInvType(book, ObjSpell1, 0) == noObject {
			g.object(book).setQuality(first)
		}
	}
	if every {
		for i := range g.spellList {
			g.spellList[i] = allSpellsMask
		}
	}
	for i := range g.spellList {
		g.spellList[i] &= allSpellsMask
	}
	if g.spellbooks[0] != noObject {
		g.selectedSpell = g.object(g.spellbooks[0]).Quality()
	} else {
		g.selectedSpell = g.object(g.spellbooks[1]).Quality()
	}
	g.pageStart = -1
	count := 0
	circle := g.selectedSpell / circleSlots
	for c := 0; c <= circle; c++ {
		for i := range circleSlots {
			if !g.hasSpell(c, i) {
				continue
			}
			if c == circle {
				if g.pageStart == -1 {
					g.pageStart = count
				}
				if c*circleSlots+i == g.selectedSpell {
					if g.pageStart+pageRows <= count {
						g.pageStart += pageRows
					}
					break
				}
			}
			count++
		}
	}
}

// drawSpellbook draws the parchment of the spellbook.
// C: seg_1944:05B1 (SPELL_DrawBG)
func (g *Game) drawSpellbook() {
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(bookX0, bookY0, bookX1, bookY1)
	if p, err := vga.LoadPixmap(g.files, "NEWMAGIC.BMP"); err == nil {
		g.gr.DrawPixmap(p, bookX0, bookY0)
	}
}

// printSpellQuantity prints how often spell can be cast with the reagents n carries and
// remembers it for row.
// C: seg_1944:0639 (PrintSpellQuantity)
func (g *Game) printSpellQuantity(n, spell, row int) {
	need := reagentsNeeded[spell]
	if need == noReagents && spell != castHelp && spell != castArmageddon {
		return
	}
	most := noReagentCap
	for r := range reagentKinds {
		if need>>r&1 != 0 {
			most = min(most, g.reagentCount(n, reagentTypes[r], 0))
		}
	}
	if spell == castHelp || spell == castArmageddon {
		most = 1
	}
	g.pageCounts[row] = most
	g.printf("%d", most)
}

// pageRow is the row of spell on the page, or -1.
// C: seg_1944:06CA (C_1944_06CA)
func (g *Game) pageRow(spell int) int {
	for i, s := range g.pageSpells {
		if s == spell {
			return i
		}
	}
	return -1
}

// refreshPage writes the spells of the page with how often each can be cast, and the
// corners.
// C: seg_1944:0773 (SPELL_RefreshPage)
func (g *Game) refreshPage() {
	lastCircle, lastIndex, pageCircle, firstCircle := -1, -1, -1, -1
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(pageX0, pageY0, pageX1, pageY1)
	g.setClip(clipParty)
	n := g.caster()
	g.corners = [4]int{noCorner, noCorner, noCorner, noCorner}
	// fix: the original kept the counts and marks of the page shown before
	g.pageCounts, g.pageMarked = [pageRows]int{}, [pageRows]bool{}
	row, skipped := 0, 0
	for circle := range circleCount {
		for i := range circleSlots {
			if !g.hasSpell(circle, i) {
				continue
			}
			if g.pageStart > skipped {
				skipped++
				if firstCircle == -1 {
					firstCircle = circle
				}
				lastCircle, lastIndex = circle, i
				continue
			}
			if pageCircle == -1 {
				pageCircle = circle
			}
			if row < pageRows && circle == pageCircle {
				if circle == lastCircle && i > lastIndex && row == 0 {
					g.corners[cornerLeft] = turnToPage
				}
				if circle > firstCircle && row == 0 {
					back := 0
					if g.corners[cornerLeft] != noCorner {
						back = g.corners[cornerLeft]
					}
					g.corners[cornerPreviousCircle] = lastCircle - back
					// fix: the original turned to a circle before the first
					if g.corners[cornerPreviousCircle] < 0 {
						g.corners[cornerPreviousCircle] = noCorner
					}
				}
				if g.corners[cornerPreviousCircle] != noCorner && g.corners[cornerLeft] == noCorner {
					g.corners[cornerLeft] = turnToCircle
				}
				spell := circle*circleSlots + i
				g.pageSpells[row] = spell
				g.gotoxy(1, row+1)
				g.printf("%s", spellNames[spell])
				g.gotoxy(1+quantityColumn, row+1)
				g.printSpellQuantity(n, spell, row)
				if g.selectedSpell == spell {
					g.gotoxy(0, row+1)
					g.putch0(arrowChar)
					g.pageMarked[row] = true
				}
				lastCircle, lastIndex = circle, i
				row++
				continue
			}
			if g.corners[cornerRight] == noCorner && circle == lastCircle && i > lastIndex {
				g.corners[cornerRight] = turnToPage
			}
			if g.corners[cornerRight] == noCorner {
				g.corners[cornerRight] = turnToCircle
			}
			if g.corners[cornerNextCircle] == noCorner && circle > lastCircle {
				g.corners[cornerNextCircle] = circle
			}
		}
	}
	g.pageLength = row
	for c := cornerLeft; c <= cornerRight; c++ {
		x := (c*2 + partyClipX0) * cellSize
		if g.corners[c] != noCorner {
			g.gr.DrawTile(tileStatusSlot+c, x, cornerY)
		} else {
			g.gr.DrawTile(tileStatusSlot, x, cornerY)
		}
	}
	g.gotoxy(11, 10)
	// fix: the original read before the names when no spell was on the page
	if pageCircle >= 0 {
		g.printf("%s", circleNames[pageCircle])
	}
	g.gotoxy(10, 11)
	g.printf("level")
	g.setClip(clipMessage)
}

// selectSpell makes spell the selected spell of the spellbooks.
// C: seg_0C9C:1991 (C_0C9C_1991)
func (g *Game) selectSpell(spell int) {
	g.selectedSpell = spell
	for _, book := range g.spellbooks {
		if book != noObject {
			g.object(book).setQuality(spell)
		}
	}
}

// moveSpellArrow moves the arrow from row from to row to.
func (g *Game) moveSpellArrow(from, to int) {
	g.setClip(clipParty)
	g.gotoxy(0, from+1)
	g.putch0(' ')
	g.gotoxy(0, to+1)
	g.putch0(arrowChar)
	g.setClip(clipMessage)
	g.selectSpell(g.pageSpells[to])
}

// spellbookArrowKey moves the arrow or turns the page with the arrow keys.
// C: seg_0C9C:19D6 (C_0C9C_19D6)
func (g *Game) spellbookArrowKey() int {
	const north, east, south, west = 0, 2, 4, 6
	switch {
	case g.AdvanceDir == east && g.corners[cornerRight] != noCorner:
		g.pageTurned = true
		return cmdSpellPageRight
	case g.AdvanceDir == west && g.corners[cornerLeft] != noCorner:
		g.pageTurned = true
		return cmdSpellPageLeft
	case g.AdvanceDir == north:
		row := max(g.pageRow(g.selectedSpell), 0)
		if row != 0 {
			g.moveSpellArrow(row, row-1)
		}
	case g.AdvanceDir == south:
		row := max(g.pageRow(g.selectedSpell), 0)
		if g.pageLength-1 > row && row < pageRows-1 {
			g.moveSpellArrow(row, row+1)
		}
	}
	return 0
}

// castSelected picks the spell the spellbooks have selected, or spell when it is not
// negative, and returns the command that casts it.
func (g *Game) pickSpell(spell int) int {
	if spell < 0 {
		return 0
	}
	g.selectSpell(spell)
	g.refreshSpellList()
	g.refreshPage()
	g.Selection.obj = g.pageRow(spell)
	g.Selection.x, g.Selection.y = -1, -1
	return cmdSelected
}

// bookSpell is the spell the first readied spellbook has selected.
func (g *Game) bookSpell() int {
	if g.spellbooks[0] != noObject {
		return g.object(g.spellbooks[0]).Quality()
	}
	return g.object(g.spellbooks[1]).Quality()
}

// spellbookKey handles the keys while the spellbook is shown.
// C: seg_0C9C:1D59 (C_0C9C_1D59), StatusDisplay CMD_A1
func (g *Game) spellbookKey(ch int) int {
	switch ch {
	case ' ', keyEscape:
		g.Selection.x, g.Selection.y = -1, -1
		return cmdPass
	case keyEnter:
		g.SelectMode = 1
		return g.pickSpell(g.bookSpell())
	case cmdAdvance:
		return g.spellbookArrowKey()
	}
	return ch
}

// incantationKey handles a key while the player types the words of a spell.
// C: seg_0C9C:1D59 (C_0C9C_1D59), SelectMode 3
func (g *Game) incantationKey(ch int) int {
	c := g.current
	switch {
	case ch >= 'A' && ch <= 'Z':
		if g.letterCount < spellLetters {
			k := g.letterCount
			g.letters[k] = byte(ch)
			g.letterX[k], g.letterY[k] = c.curX, c.curY
			g.printf("%s", powerWords[ch-'A'])
			g.putch(' ')
			g.putch0(endOfLine)
			if c.curX >= c.width {
				g.putch0(0)
			}
			// the line scrolled up under the words
			if c.curY == g.letterY[k] && c.curX < g.letterX[k] {
				for i := 0; i <= k; i++ {
					g.letterY[i]--
				}
			}
			g.letterCount++
		}
		return 0
	case ch == keyBackspace || ch == keyDelete:
		if g.letterCount != 0 {
			g.letterCount--
			k := g.letterCount
			w := len(powerWords[g.letters[k]-'A'])
			if c.curX > w {
				g.gotoxy(c.curX-w-1, c.curY)
			} else {
				g.gotoxy(g.letterX[k], g.letterY[k])
			}
			for range w {
				g.putch0(' ')
			}
			g.gotoxy(g.letterX[k], g.letterY[k])
		}
		return 0
	case ch == ' ' || ch == keyEscape:
		g.letterCount = 0
		g.Selection.x, g.Selection.y = -1, -1
		return cmdPass
	case ch == keyEnter:
		if g.letterCount == 0 {
			g.SelectMode = 1
			return g.pickSpell(g.bookSpell())
		}
		spell := g.incantationSpell()
		if spell < 0 {
			g.Selection.x, g.Selection.y = -1, -1
			return cmdPass
		}
		return g.pickSpell(spell)
	case ch == cmdAdvance:
		return g.spellbookArrowKey()
	}
	return 0
}

// incantationSpell finds the spell of the spellbooks whose words were typed, or -1.
// C: seg_0C9C:18D8 (__Incant2Spell)
func (g *Game) incantationSpell() int {
	typed := 0
	for _, l := range g.letters[:g.letterCount] {
		typed |= 1 << (l - 'A')
	}
	spell := -1
	for i := range circleCount * circleSlots {
		if !g.hasSpell(i/circleSlots, i%circleSlots) {
			continue
		}
		mask := 0
		for _, l := range spellCodes[i] {
			mask |= 1 << (l - 'A')
		}
		if mask == typed {
			spell = i
			break
		}
	}
	if spell < 0 {
		g.printf(notInBookMsg)
	}
	g.printf("\n")
	g.SelectMode = 1
	return spell
}

// clickSpellbook is the command of a click on the spellbook.
// C: seg_0C9C:15EB (do_CMD_A1)
func (g *Game) clickSpellbook() int {
	x, y := g.PointerX, g.PointerY
	sel := &g.Selection
	switch {
	case y >= cornerY && y < cornerY+2*tileSize && x >= pageX0 && x < pageX0+4*tileSize:
		cmd := 0
		if c := (x - pageX0) / tileSize; g.corners[c] != noCorner {
			cmd = cmdSpellPrevious + c
		}
		sel.x, sel.y = -1, -1
		return cmd
	case x > pageX0+tileSize && y >= 2*tileSize && y < 10*tileSize:
		cmd := cmdSelected
		sel.obj = (y - 2*tileSize) / tileSize
		if g.pageLength <= sel.obj {
			cmd = 0
		}
		sel.x, sel.y = -1, -1
		return cmd
	}
	return 0
}

// spellsBefore counts the spells of the circles before circle.
func (g *Game) spellsBefore(circle int) int {
	count := 0
	for c := 0; c < circle; c++ {
		for i := range circleSlots {
			if g.hasSpell(c, i) {
				count++
			}
		}
	}
	return count
}

// spellbookCommand turns the pages of the spellbook.
// C: seg_0C9C:2A59 (CON_getch), CMD_A2..CMD_A5
func (g *Game) spellbookCommand(ch int) {
	switch ch {
	case cmdSpellPrevious:
		g.pageStart = g.spellsBefore(g.corners[cornerPreviousCircle])
		g.refreshPage()
	case cmdSpellNext:
		g.pageStart = g.spellsBefore(g.corners[cornerNextCircle])
		g.refreshPage()
	case cmdSpellPageLeft:
		if g.corners[cornerLeft] == turnToPage {
			g.pageStart -= pageRows
		} else {
			was := g.pageStart
			g.pageStart = g.spellsBefore(g.corners[cornerPreviousCircle])
			if was-g.pageStart > pageRows {
				g.pageStart += pageRows
			}
		}
		g.turnedPage()
	case cmdSpellPageRight:
		if g.corners[cornerRight] == turnToPage {
			g.pageStart += pageRows
		} else {
			g.pageStart = g.spellsBefore(g.corners[cornerNextCircle])
		}
		g.turnedPage()
	}
}

// turnedPage shows the page turned to, with the arrow at the top when the keys turned it.
// C: seg_0C9C:2A16 (C_0C9C_2A16)
func (g *Game) turnedPage() {
	g.refreshPage()
	if g.pageTurned {
		g.pageTurned = false
		g.selectSpell(g.pageSpells[0])
		g.setClip(clipParty)
		g.gotoxy(0, 1)
		g.putch0(arrowChar)
		g.setClip(clipMessage)
	}
}

// lookSpellbook shows the page of book and the words and reagents of the spell picked.
// C: seg_27A1:0C67 (C_27A1_0C67), the spellbook
func (g *Game) lookSpellbook(book int) {
	g.spellbooks = [2]int{book, noObject}
	active := g.Active
	owner := g.object(book).Pos.Owner()
	for i := range partyMax {
		if int(g.Party[i]) == owner {
			g.Active = byte(i)
			break
		}
	}
	g.refreshSpellList()
	g.drawSpellbook()
	g.refreshPage()
	saved := g.StatusDisplay
	g.StatusDisplay = cmdStatusSpellbook
	if g.getch() == cmdSelected && g.Selection.obj != noObject {
		spell := g.pageSpells[g.Selection.obj]
		g.printf("\n\n")
		g.printf("%s-", spellNames[spell])
		g.printf("%s", incantation(spell))
		g.printf("\nIngredients: \n")
		need := reagentsNeeded[spell]
		if need == noReagents {
			g.printf("None\n")
		}
		for r := range reagentKinds {
			if need>>r&1 != 0 {
				g.printf(" %s\n", reagentNames[r])
			}
		}
	} else {
		g.printf(".\n")
	}
	g.Active = active
	g.StatusDisplay = saved
	g.statusMember = 0
	g.StatusDirty = true
}
