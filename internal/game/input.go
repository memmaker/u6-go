package game

import (
	"github.com/memmaker/u6-go/internal/host"
)

// Commands (cmd.h CMD_xx): what a key or a mouse click asks for. The values continue the
// key codes and are kept as the original numbers since they are stored in saved games
// (DefaultCommand) and drawn as icon positions.
const (
	cmdAdvance         = 128 // walk in AdvanceDir
	cmdAttack          = 129
	cmdCast            = 130
	cmdTalk            = 131
	cmdLook            = 132
	cmdGet             = 133
	cmdDrop            = 134
	cmdMove            = 135
	cmdUse             = 136
	cmdRest            = 137
	cmdCombat          = 138 // begin or break off combat
	cmdSelected        = 142 // a target was selected
	cmdPass            = 143
	cmdStatusPortrait  = 144
	cmdStatusParty     = 145
	cmdStatusEquipment = 146
	cmdPartyScrollUp   = 147
	cmdPartyScrollDown = 148
	cmdInventoryUp     = 149
	cmdInventoryDown   = 150
	cmdNextMember      = 153
	cmdPreviousMember  = 154
	cmdSelectEquipment = 155
	cmdSelectInventory = 156
	cmdSelectMap       = 157
	cmdStatusContainer = 158
	cmdCloseContainer  = 159
	cmdOpenContainer   = 160
	cmdStatusSpellbook = 161
	cmdSpellPrevious   = 162
	cmdSpellPageLeft   = 163
	cmdSpellPageRight  = 164
	cmdSpellNext       = 165
	cmdSpellNothing    = 166
	cmdReady           = 167
	cmdUnready         = 168
	cmdDefault         = 169
	cmdSave            = 170
	// noDefaultCommand is DefaultCommand when none is set
	noDefaultCommand = 255
)

// Key codes the game tests that are not commands.
const (
	keyCtrlE  = 5
	keyCtrlH  = 8
	keyTab    = 9
	keyCtrlQ  = 17
	keyCtrlR  = 18
	keyCtrlS  = 19
	keyCtrlV  = 22
	keyCtrlZ  = 26
	keyEnter  = '\r'
	keyDelete = 127
	// alt+key codes
	keyAltA = extendedKey + 30
	keyAltB = extendedKey + 48
	keyAltC = extendedKey + 46
	keyAltD = extendedKey + 32
	keyAltG = extendedKey + 34
	keyAltL = extendedKey + 38
	keyAltM = extendedKey + 50
	keyAltP = extendedKey + 25
	keyAltR = extendedKey + 19
	keyAltT = extendedKey + 20
	keyAltU = extendedKey + 22
	keyAltX = extendedKey + 45
	// keyAlt213 and friends are typed as alt + 2 1 3 on the keypad (debug commands)
	keyAlt213 = 213
	keyAlt214 = 214
	keyAlt215 = 215
)

// input is the state of the command input (seg_0C9C).
type input struct {
	// SelectMode is what arrow keys move: 0 the party, 1 the aim cursor on the map, 2 the
	// cursor in the status area, 3 a spell incantation.
	SelectMode int
	// AimX, AimY is the aim cursor in view tiles; SelectRange how far it may go (-1: only
	// next to the party).
	AimX, AimY  int
	SelectRange int
	// selectByTab is set when tab started the selection (D_0497)
	selectByTab bool
	// the cursor in the status area (D_0499, D_049A)
	statusCursorX, statusCursorY int
	// pendingCommand is the command waiting for a target (D_04C2)
	pendingCommand int
	// MouseMode is 1 while a command waits for a target.
	MouseMode int
	// AdvanceDir is the direction of cmdAdvance.
	AdvanceDir int
	// allowMouseMove lets holding the button walk (AllowMouseMov); mouseMapX/Y is the map
	// tile walked to (MouseMapX/Y), mouseTileX/Y the view tile (D_0527, D_0529)
	allowMouseMove         bool
	mouseMapX, mouseMapY   int
	mouseTileX, mouseTileY int
	// mouse buttons: held (MousePress), just pressed (MouseTrigger), as pressed
	// (D_B72B), the command of the press (D_B64F)
	mousePressed, mouseTriggered bool
	mouseButtons                 int
	mousePressCommand            int
	// PointerX, PointerY is where the pointer (or the keyboard cursor) points.
	PointerX, PointerY int
	// pointerCentreX/Y is the centre of the map view, where the mouse directions start
	// (D_04CA, D_04CB)
	pointerCentreX, pointerCentreY int
	mouse                          struct {
		x, y        int
		left, right bool
	}
	// Selection is the target picked for a command.
	Selection struct{ x, y, obj int }
	// selectedDark says the selected tile cannot be seen (D_B6DF)
	selectedDark bool
	// repeatTarget makes the next getch return cmdSelected for the default command
	// (D_0578)
	repeatTarget bool
	// cancelled is set when typing a string was cancelled (D_04AD)
	cancelled bool
	// peering is set while a gem shows the map (D_17AE)
	peering bool
}

const (
	leftButton  = 1
	rightButton = 2
)

func (in *input) initInput() {
	in.pendingCommand = cmdPass
	in.mouseMapX, in.mouseMapY = -1, -1
	in.mouseTileX, in.mouseTileY = -1, -1
	in.pointerCentreX, in.pointerCentreY = mapCentrePixel, mapCentrePixel
	in.statusCursorX = 3
}

// mapCentrePixel is the centre of the map view on the screen (88).
const mapCentrePixel = viewCenter*tileSize + tileSize/2 + borderSize

const (
	tileSize   = 16
	borderSize = 8
	// directionDeadZone and directionZone are the half sizes, in pixels, of the square in
	// which the pointer points at the party and of the 3x3 tiles around it
	directionDeadZone = 8
	directionZone     = 24
)

// serveMouse reads the mouse and moves the pointer.
// C: MOUSE.ASM MOUSE_SERVE
func (g *Game) serveMouse() {
	g.mouse.x, g.mouse.y = host.MouseX, host.MouseY
	g.mouse.left, g.mouse.right = host.MouseLeft, host.MouseRight
	g.gr.MovePointer(g.mouse.x, g.mouse.y)
	g.updatePointerShape()
}

// pointerDirection returns the direction from the party to screen point (x, y), or -1
// when it points at the party.
// C: seg_0C9C:0069 (MkDirection)
func (g *Game) pointerDirection(x, y int) int {
	cx, cy := g.pointerCentreX, g.pointerCentreY
	dx, dy := abs(x-cx), abs(y-cy)
	if cx-directionZone <= x && cx+directionZone > x && cy-directionZone <= y && cy+directionZone > y {
		column := 2
		if cx-directionDeadZone > x {
			column = 0
		} else if cx+directionDeadZone > x {
			column = 1
		}
		row := 2
		if cy-directionDeadZone > y {
			row = 0
		} else if cy+directionDeadZone > y {
			row = 1
		}
		return [3][3]int{{7, 6, 5}, {0, -1, 4}, {1, 2, 3}}[column][row]
	}
	dir := -1
	if dx > dy {
		if dy == 0 || dx/dy >= 2 {
			dir = 6
			if cx <= x {
				dir = 2
			}
		}
	} else if dx == 0 || dy/dx >= 2 {
		dir = 0
		if cy <= y {
			dir = 4
		}
	}
	if dir == -1 {
		switch {
		case cx <= x && cy <= y:
			dir = 3
		case cx <= x:
			dir = 1
		case cy <= y:
			dir = 5
		default:
			dir = 7
		}
	}
	return dir
}

// Pointer shapes in U6MCGA.PTR.
const (
	pointerArrow     = 0
	pointerCrosshair = 1
	// pointerNorth is the first of the eight direction arrows
	pointerNorth = 2
)

// updatePointerShape shows a direction arrow over the map view, the arrow elsewhere and
// the crosshair while a command waits for a target.
// C: seg_0C9C:01FB (C_0C9C_01FB)
func (g *Game) updatePointerShape() {
	x, y := g.mouse.x, g.mouse.y
	const viewEnd = borderSize + viewTiles*tileSize - borderSize*2 + borderSize
	if x >= borderSize && x < viewEnd && y >= borderSize && y < viewEnd {
		if g.MouseMode == 0 {
			if d := g.pointerDirection(x, y); d >= 0 {
				g.gr.SetPointer(pointerNorth + d)
				return
			}
		}
	} else if g.MouseMode == 0 {
		g.gr.SetPointer(pointerArrow)
		return
	}
	g.gr.SetPointer(pointerCrosshair)
}

// prompt shows the cursor and returns a key, or 0 after one frame of animation when none
// was pressed.
// C: seg_0C9C:0E6E (CON_prompt)
func (g *Game) prompt() int {
	c := g.current
	fg := c.fg
	c.fg = g.colours.text
	g.linesLeft = c.height
	if g.pagingDisabled {
		c.attr &^= clipPaging
		g.pagingDisabled = false
	}
	if c.curX >= c.width {
		g.putch0(0)
	}
	x, y := c.curX+c.x0, c.curY+c.y0
	ch := readKey(g.drainKeys)
	if ch == 0 {
		g.putCharAt(byte(g.promptChar+g.promptCount&3), x, y)
		g.promptCount++
		g.animationTick = true
		if g.animatePrompt {
			g.OtherAnimations()
			g.paletteAnimation()
			// the original animated as fast as the CPU allowed; a timer tick paces it on the host
			g.osiDelay(1)
		} else {
			g.osiDelay(1)
		}
		g.serveMouse()
	}
	g.putCharAt(' ', x, y)
	c.fg = fg
	return ch
}

// screen zones for the mouse (D_052B)
var mouseZones = [4][4]int{
	{8, 8, 167, 167},   // map view
	{8, 176, 167, 191}, // command icons
	{168, 8, 311, 103}, // status area
	{0, 0, 319, 199},   // the rest of the screen
}

const (
	zoneMap = iota
	zoneIcons
	zoneStatus
	zoneScreen
)

// mouseCommand returns the command of a mouse press.
// C: seg_0C9C:1714 (mkMouseCommand)
func (g *Game) mouseCommand() int {
	zone := 0
	for ; ; zone++ {
		z := mouseZones[zone]
		if z[0] <= g.PointerX && z[2] >= g.PointerX && z[1] <= g.PointerY && z[3] >= g.PointerY {
			break
		}
	}
	cmd := 0
	switch zone {
	case zoneMap:
		if g.MouseMode != 0 {
			if g.mouseTriggered {
				cmd = cmdSelected
				g.mouseSelection()
			}
			break
		}
		if g.DefaultCommand != noDefaultCommand && g.mouseButtons&rightButton != 0 {
			if g.mouseTriggered {
				cmd = cmdDefault
				g.mouseSelection()
			}
			break
		}
		dir := g.mouseDirection()
		if dir == -1 {
			if g.mouseTriggered {
				cmd = cmdPass
			}
		} else if g.mouseTriggered || !g.peering || dir != g.AdvanceDir {
			cmd = cmdAdvance
			g.AdvanceDir = dir
		}
	case zoneIcons:
		if g.mouseTriggered {
			icon := cmdAttack + (g.PointerX-borderSize)/tileSize
			if g.mouseButtons&leftButton != 0 {
				cmd = icon
			} else {
				g.setDefaultCommand(icon)
			}
		}
	case zoneStatus:
		if !g.mouseTriggered {
			break
		}
		if g.DefaultCommand != noDefaultCommand && g.MouseMode == 0 && g.mouseButtons&rightButton != 0 {
			cmd = cmdDefault
			g.mouseSelection()
			break
		}
		switch g.StatusDisplay {
		case cmdStatusParty:
			cmd = g.clickPartyView()
		case cmdStatusPortrait:
			cmd = g.clickPortraitView()
		case cmdStatusEquipment:
			cmd = g.clickEquipmentView()
			if cmd == 0 && g.MouseMode == 1 && g.mouseSelection() != 0 {
				cmd = cmdSelected
			}
		case cmdStatusSpellbook:
			// TODO(port): do_CMD_A1
		}
	case zoneScreen:
		if g.mouseTriggered {
			cmd = cmdPass
			g.Selection.x, g.Selection.y = -1, -1
		}
	}
	return cmd
}

// mouseDirection returns the direction the pointer points to and remembers the map tile
// under it while the mouse walks the party.
// C: seg_0C9C:1054 (mkMouseDir)
func (g *Game) mouseDirection() int {
	tx, ty := g.PointerX/tileSize, g.PointerY/tileSize
	dir := g.pointerDirection(g.PointerX, g.PointerY)
	if !g.allowMouseMove {
		g.mouseMapX, g.mouseMapY = -1, -1
		return dir
	}
	mx, my := int(g.MapX), int(g.MapY)
	if g.mouseTriggered || g.mouseMapX == -1 ||
		g.mouseMapX == mx && abs(g.mouseMapY-my) < 2 ||
		g.mouseMapY == my && abs(g.mouseMapX-mx) < 2 ||
		tx != g.mouseTileX || ty != g.mouseTileY {
		g.mouseMapX = tx + mx - viewCenter
		g.mouseMapY = ty + my - viewCenter
		g.mouseTileX, g.mouseTileY = tx, ty
	}
	return dir
}

// mouseSelection makes the point under the pointer the Selection; it returns
// cmdSelectMap when there is something there, cmdSelectEquipment/cmdSelectInventory for
// an item in the equipment view, else 0.
// C: seg_0C9C:1107 (mkMouseSelection)
func (g *Game) mouseSelection() int {
	g.selectedDark = false
	const statusAreaX = 168
	switch {
	case g.PointerX < statusAreaX:
		tx, ty := g.PointerX/tileSize, g.PointerY/tileSize
		g.Selection.x = tx + int(g.MapX) - viewCenter
		g.Selection.y = ty + int(g.MapY) - viewCenter
		// fix: the original reads past the view for points on its border
		tx, ty = min(tx, viewTiles-1), min(ty, viewTiles-1)
		if g.objectView[ty][tx] != 0 || g.tileView[ty][tx] >= firstObjectTile {
			g.Selection.obj = g.objectAtForSelection(int(g.Party[g.Active]), g.Selection.x, g.Selection.y)
			return cmdSelectMap
		}
		g.Selection.obj = noObject
		t := g.tileView[ty][tx]
		g.selectedDark = t == tileHidden || t == tileDarkness
	case g.StatusDisplay == cmdStatusEquipment:
		g.Selection.x, g.Selection.y = -1, -1
		// TODO(port): C_155D_1267, C_155D_130E (the equipment and inventory slots)
	default:
		g.Selection.x, g.Selection.y, g.Selection.obj = -1, -1, noObject
	}
	return 0
}

// objectAtForSelection returns the object at (x, y) a command would act on.
// TODO(port): C_2337_08F1
func (g *Game) objectAtForSelection(actor, x, y int) int {
	return g.FindLoc(x, y, int(g.MapZ))
}

// clickPartyView returns the command of a click in the party list.
// C: seg_0C9C:121F (do_CMD_91)
func (g *Game) clickPartyView() int {
	if g.PointerY <= 24 {
		return 0
	}
	switch {
	case g.PointerX > 192:
		g.Selection.obj = (g.PointerY-28)/tileSize + g.partyScroll
		if int(g.PartySize) <= g.Selection.obj {
			return 0
		}
		return cmdStatusPortrait
	case g.PointerX > 176:
		g.Selection.obj = (g.PointerY-24)/tileSize + g.partyScroll
		if int(g.PartySize) <= g.Selection.obj {
			return 0
		}
		return cmdStatusEquipment
	case g.PointerY >= 24 && g.PointerY < 40:
		if g.partyScroll != 0 {
			return cmdPartyScrollUp
		}
	case g.PointerY >= 88:
		if g.partyScroll < int(g.PartySize)-partyListRows {
			return cmdPartyScrollDown
		}
	}
	return 0
}

// statusButton returns the command of the buttons under the portrait and equipment views.
func (g *Game) statusButton(other int) int {
	switch {
	case g.PointerX >= 192 && g.PointerX < 208:
		return cmdStatusParty
	case g.PointerX >= 208 && g.PointerX < 224:
		g.Selection.obj = noObject
		return other
	case g.PointerX >= 224 && g.PointerX < 240:
		return cmdNextMember
	case g.PointerX >= 176 && g.PointerX < 192:
		return cmdPreviousMember
	}
	return 0
}

// clickPortraitView returns the command of a click in the portrait view.
// C: seg_0C9C:12B7 (do_CMD_90)
func (g *Game) clickPortraitView() int {
	if g.PointerY >= 88 {
		return g.statusButton(cmdStatusEquipment)
	}
	return 0
}

// clickEquipmentView returns the command of a click in the equipment view.
// C: seg_0C9C:131D (do_CMD_92)
func (g *Game) clickEquipmentView() int {
	if g.InConversation {
		return 0
	}
	if g.PointerY >= 88 && !(g.statusMember != 0 && g.PointerX >= 240 && g.PointerX < 256 && g.PointerY < 104) {
		return g.statusButton(cmdStatusPortrait)
	}
	// TODO(port): do_CMD_92 combat mode button, inventory scrolling, containers, ready
	return 0
}

// setDefaultCommand makes cmd the command of the right mouse button and underlines its
// icon.
// C: seg_0C9C:166D (setDefaultCommand)
func (g *Game) setDefaultCommand(cmd int) {
	const underlineY = 192
	underline := func(colour byte) {
		x := (int(g.DefaultCommand)-cmdAttack)*tileSize + borderSize
		g.gr.SetColor(colour)
		g.gr.FillRect(x, underlineY, x+tileSize-1, underlineY)
	}
	if g.DefaultCommand != noDefaultCommand {
		underline(g.colours.background)
	}
	g.DefaultCommand = uint8(cmd)
	if g.DefaultCommand != noDefaultCommand {
		underline(g.colours.defaultCommand)
	}
}

// selectionKeys are the keys that repeat a pending command, which confirms its target.
var selectionKeys = map[int]int{
	cmdAttack: 'A', cmdTalk: 'T', cmdLook: 'L', cmdGet: 'G', cmdDrop: 'D', cmdMove: 'M', cmdUse: 'U',
}

// readCommand waits for a key or a mouse click and returns the command it makes.
// C: seg_0C9C:1D59 (C_0C9C_1D59)
func (g *Game) readCommand() int {
	if g.flushOnGetch {
		g.putch0(endOfLine)
	}
	ch := 0
	g.linesLeft = g.current.height
	for ch == 0 {
		g.serveMouse()
		buttons := 0
		if g.mouse.left {
			buttons |= leftButton
		}
		if g.mouse.right {
			buttons |= rightButton
		}
		if buttons != 0 {
			g.mouseTriggered = !g.mousePressed
			g.mousePressed = true
			g.PointerX, g.PointerY = g.mouse.x, g.mouse.y
			g.mouseButtons = buttons
			ch = g.mouseCommand()
			if g.mouseTriggered {
				g.mousePressCommand = ch
			} else if g.mousePressCommand != ch {
				ch = 0
			}
		} else {
			g.mousePressed, g.mouseTriggered = false, false
		}
		if ch != 0 {
			continue
		}
		ch = g.prompt()
		if ch != 0 {
			g.mouseMapX, g.mouseMapY = -1, -1
		}
		if g.uppercaseKeys && ch < extendedKey {
			ch = toUpper(ch)
		}
		switch ch {
		case keyUp:
			ch, g.AdvanceDir = cmdAdvance, 0
		case keyDown:
			ch, g.AdvanceDir = cmdAdvance, 4
		case keyLeft:
			ch, g.AdvanceDir = cmdAdvance, 6
		case keyRight:
			ch, g.AdvanceDir = cmdAdvance, 2
		case keyHome:
			ch, g.AdvanceDir = cmdAdvance, 7
		case keyPageUp:
			ch, g.AdvanceDir = cmdAdvance, 1
		case keyEnd:
			ch, g.AdvanceDir = cmdAdvance, 5
		case keyPageDown:
			ch, g.AdvanceDir = cmdAdvance, 3
		case keyAltA, keyAltC, keyAltT, keyAltL, keyAltG, keyAltD, keyAltM, keyAltU, keyAltR, keyAltB:
			g.setDefaultCommand(map[int]int{
				keyAltA: cmdAttack, keyAltC: cmdCast, keyAltT: cmdTalk, keyAltL: cmdLook, keyAltG: cmdGet,
				keyAltD: cmdDrop, keyAltM: cmdMove, keyAltU: cmdUse, keyAltR: cmdRest, keyAltB: cmdCombat,
			}[ch])
			ch = 0
		}
		if g.MouseMode == 0 && !g.selectByTab {
			g.drawStatusCursor(0)
			g.SelectMode = 0
		}
		if !g.flushOnGetch || g.typingString && ch == keyEnter {
			continue
		}
		switch {
		case g.SelectMode != 0:
			if k, ok := selectionKeys[g.pendingCommand]; ok && ch == k ||
				g.pendingCommand == cmdCast && ch == 'C' && g.SelectMode == 1 {
				ch = keyEnter
			}
			if g.pendingCommand == cmdCast && g.SelectMode == 3 {
				// TODO(port): C_0C9C_1D59 spell incantation (__Incant2Spell, C_0C9C_19D6)
				ch = 0
				continue
			}
			ch = g.selectTarget(ch)
		case ch == keyTab && g.StatusDisplay != cmdStatusContainer && !g.InConversation:
			if g.StatusDisplay == cmdStatusEquipment {
				g.SelectMode = 2
				g.drawStatusCursor(1)
			} else if g.StatusDisplay != cmdStatusSpellbook {
				g.SelectMode = 1
			}
			if g.SelectMode != 0 {
				g.AimX, g.AimY = viewCenter, viewCenter
				g.SelectRange = 7
				g.selectByTab = true
				g.drawMap(true)
				ch = 0
			}
		}
		ch = g.statusKey(ch)
	}
	return ch
}

// selectTarget handles a key while the aim cursor (SelectMode 1) or the status cursor
// (SelectMode 2) is shown.
// C: seg_0C9C:1D59 (C_0C9C_1D59, the part for SelectMode 1 and 2)
func (g *Game) selectTarget(ch int) int {
	switch {
	case ch == keyTab:
		if g.SelectMode == 1 {
			g.SelectMode = 2
			if g.StatusDisplay != cmdStatusEquipment {
				g.statusMember = int(g.Active)
				if g.inVehicle() {
					g.statusMember = 0
				}
				g.StatusDisplay = cmdStatusEquipment
				g.StatusDirty = true
				g.RefreshStatus()
			}
			g.drawStatusCursor(1)
		} else if g.StatusDisplay != cmdStatusContainer && !g.InConversation {
			g.drawStatusCursor(0)
			g.SelectMode = 1
		}
		g.drawMap(true)
		return 0
	case ch == ' ' || ch == keyEscape:
		g.drawStatusCursor(0)
		g.SelectMode = 0
		return 0
	case ch == keyEnter:
		if g.SelectMode == 1 {
			ch = cmdSelected
			if g.MouseMode == 0 {
				if g.selectByTab && g.DefaultCommand != noDefaultCommand {
					ch = cmdDefault
				}
				g.selectByTab = false
			}
			g.SelectMode = 0
			g.PointerX, g.PointerY = g.AimX*tileSize, g.AimY*tileSize
			g.mouseSelection()
		} else if g.SelectMode == 2 {
			// TODO(port): C_0C9C_1D59 selecting in the equipment and portrait views
		}
		return ch
	case ch == cmdAdvance:
		if g.SelectMode == 1 {
			x, y := g.AimX+dirStepX[g.AdvanceDir], g.AimY+dirStepY[g.AdvanceDir]
			if x >= 0 && x < viewTiles && y >= 0 && y < viewTiles && g.combatRange(x, y, viewCenter, viewCenter) <= g.SelectRange {
				g.AimX, g.AimY = x, y
				g.drawMap(true)
			} else if g.SelectRange == -1 {
				g.PointerX, g.PointerY = x*tileSize, y*tileSize
				g.mouseSelection()
				return cmdSelected
			}
			return 0
		}
		// TODO(port): C_0C9C_1D59 moving the cursor in the equipment and portrait views
		return 0
	case ch >= '1' && ch-'1' < int(g.PartySize) && g.SelectMode == 1:
		g.selectedDark = false
		g.SelectMode = 0
		g.Selection.obj = int(g.Party[ch-'1'])
		if g.inVehicle() {
			g.Selection.x, g.Selection.y = int(g.MapX), int(g.MapY)
		} else {
			p := g.object(g.Selection.obj).Pos
			g.Selection.x, g.Selection.y = p.X, p.Y
		}
		return cmdSelected
	}
	return ch
}

// rangeTable is the range of each distance on x and y (D_196C).
var rangeTable = [8][8]int{
	{0, 1, 2, 3, 4, 5, 6, 7},
	{1, 1, 2, 3, 4, 5, 6, 7},
	{2, 2, 2, 3, 4, 5, 6, 7},
	{3, 3, 3, 4, 5, 6, 7, 7},
	{4, 4, 4, 5, 6, 7, 7, 8},
	{5, 5, 5, 6, 7, 7, 8, 8},
	{6, 6, 6, 7, 7, 8, 8, 8},
	{7, 7, 7, 7, 8, 8, 8, 8},
}

// combatRange is the distance used for ranges.
// C: seg_2337 (COMBAT_getCathesus)
func (g *Game) combatRange(x0, y0, x1, y1 int) int {
	dx, dy := abs(x0-x1), abs(y0-y1)
	if dx >= len(rangeTable) || dy >= len(rangeTable) {
		return len(rangeTable)
	}
	return rangeTable[dy][dx]
}

// statusKey turns the keys of the status area into commands.
// C: seg_0C9C:1D59 (C_0C9C_1D59, the status keys)
func (g *Game) statusKey(ch int) int {
	if g.InConversation {
		return ch
	}
	member := func(ch int) (int, bool) {
		if ch >= keyF1 && int(g.PartySize)+keyF1 > ch {
			return ch - keyF1, true
		}
		return 0, false
	}
	switch g.StatusDisplay {
	case cmdStatusEquipment, cmdStatusPortrait:
		other := cmdStatusPortrait
		if g.StatusDisplay == cmdStatusPortrait {
			other = cmdStatusEquipment
		}
		switch {
		case ch == '*':
			g.Selection.obj = noObject
			return other
		case ch == '/' || ch == keyF10:
			return cmdStatusParty
		case ch == '+':
			return cmdNextMember
		case ch == '-':
			return cmdPreviousMember
		case (ch == '`' || ch == '~') && g.StatusDisplay == cmdStatusEquipment:
			g.PointerX, g.PointerY = 248, 96
			g.clickEquipmentView()
			return 0
		}
		if m, ok := member(ch); ok {
			g.Selection.obj = m
			return g.StatusDisplay
		}
	case cmdStatusParty:
		switch {
		case ch == '+' && g.partyScroll < int(g.PartySize)-partyListRows:
			return cmdPartyScrollDown
		case ch == '-' && g.partyScroll != 0:
			return cmdPartyScrollUp
		}
		if m, ok := member(ch); ok {
			g.Selection.obj = m
			return cmdStatusEquipment
		}
	case cmdStatusSpellbook:
		// TODO(port): C_0C9C_1D59 spellbook keys
	}
	return ch
}

// getch waits for a key or a click and returns the command, handling the commands that
// only change the status area.
// C: seg_0C9C:2A59 (CON_getch)
func (g *Game) getch() int {
	if g.repeatTarget {
		g.repeatTarget = false
		return cmdSelected
	}
	for {
		ch := g.readCommand()
		switch ch {
		case cmdDefault:
			ch = int(g.DefaultCommand)
			if ch != cmdCast && ch != cmdRest && ch != cmdSave {
				g.repeatTarget = true
			}
			return ch
		case cmdStatusParty:
			g.SelectMode = 0
			g.StatusDisplay = cmdStatusParty
			g.statusMember = 0
			g.StatusDirty = true
		case cmdStatusPortrait, cmdStatusEquipment:
			g.StatusDisplay = ch
			if g.Selection.obj >= 0 {
				g.statusMember = g.Selection.obj
			}
			g.StatusDirty = true
		case cmdPreviousMember:
			if g.statusMember > 0 {
				g.statusMember--
				g.StatusDirty = true
				g.Selection.obj = noObject
			}
		case cmdNextMember:
			if int(g.PartySize)-1 > g.statusMember {
				g.statusMember++
				g.StatusDirty = true
				g.Selection.obj = noObject
			}
		case cmdPartyScrollUp:
			g.partyScroll--
			g.StatusDirty = true
		case cmdPartyScrollDown:
			g.partyScroll++
			g.StatusDirty = true
		case cmdInventoryUp, cmdInventoryDown, cmdCloseContainer, cmdOpenContainer, cmdReady, cmdUnready,
			cmdSpellPrevious, cmdSpellNext, cmdSpellPageLeft, cmdSpellPageRight:
			// TODO(port): CON_getch inventory, container and spellbook commands
		case cmdSpellNothing:
		default:
			return ch
		}
		if ch != cmdSpellNothing {
			g.RefreshStatus()
		}
	}
}

// drawStatusCursor draws (mode 1), erases (mode 0) or points at (mode 2) the cursor in
// the equipment and portrait views.
// TODO(port): C_0C9C_1AE5
func (g *Game) drawStatusCursor(mode int) {
	if g.SelectMode != 2 {
		return
	}
	if g.StatusDisplay != cmdStatusEquipment && g.StatusDisplay != cmdStatusPortrait {
		return
	}
}
