package game

import (
	"image"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
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
	cmdSelectInventory = 155
	cmdSelectEquipment = 156
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
	// PointerX, PointerY is where the pointer (or the keyboard cursor) points: on the back
	// buffer's map view, or on a panel of the canvas when pointerOverUI.
	PointerX, PointerY int
	pointerOverUI      bool
	// pointerCentreX/Y is the centre of the map view, where the mouse directions start
	// (D_04CA, D_04CB)
	pointerCentreX, pointerCentreY int
	mouse                          struct {
		x, y        int
		overUI      bool
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
	in.pointerCentreX, in.pointerCentreY = mapCentreX, mapCentreY
	in.statusCursorX = 3
}

// mapCentreX, mapCentreY is the centre of the map view on the back buffer, where the
// pointer's map coordinates are counted.
const (
	mapCentreX = viewCenterX*tileSize + tileSize/2
	mapCentreY = viewCenterY*tileSize + tileSize/2
)

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
	g.mouse.x, g.mouse.y, g.mouse.overUI = g.pointAt(host.MouseX, host.MouseY)
	g.mouse.left, g.mouse.right = host.MouseLeft, host.MouseRight
	g.gr.MovePointer(host.MouseX, host.MouseY)
	g.updatePointerShape()
}

// pointerDirection returns the direction from the party to screen point (x, y), or -1
// when it points at the party.
// C: seg_0C9C:0069 (MkDirection)
func (g *Game) pointerDirection(x, y int) int {
	centreX, centreY := g.pointerCentreX, g.pointerCentreY
	distX, distY := abs(x-centreX), abs(y-centreY)
	if centreX-directionZone <= x && centreX+directionZone > x && centreY-directionZone <= y && centreY+directionZone > y {
		column := 2
		if centreX-directionDeadZone > x {
			column = 0
		} else if centreX+directionDeadZone > x {
			column = 1
		}
		row := 2
		if centreY-directionDeadZone > y {
			row = 0
		} else if centreY+directionDeadZone > y {
			row = 1
		}
		return [3][3]int{{7, 6, 5}, {0, -1, 4}, {1, 2, 3}}[column][row]
	}
	dir := -1
	if distX > distY {
		if distY == 0 || distX/distY >= 2 {
			dir = 6
			if centreX <= x {
				dir = 2
			}
		}
	} else if distX == 0 || distY/distX >= 2 {
		dir = 0
		if centreY <= y {
			dir = 4
		}
	}
	if dir == -1 {
		switch {
		case centreX <= x && centreY <= y:
			dir = 3
		case centreX <= x:
			dir = 1
		case centreY <= y:
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
)

// updatePointerShape shows the crosshair while a command waits for a target, else the arrow.
// The port shows no direction arrows over the map.
// C: seg_0C9C:01FB (C_0C9C_01FB)
func (g *Game) updatePointerShape() {
	if g.MouseMode == 0 {
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
	g.prompting = true
	defer func() { g.prompting = false }()
	ch := g.hotKey(readKey(g.drainKeys))
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
var mouseZones = [...][4]int{
	{8, 8, 167, 167},   // map view
	{8, 176, 167, 191}, // command icons
	{168, 8, 311, 103}, // status area
	{partyColumnX, partyColumnY, partyColumnX + tileSize - 1, partyColumnY + partyListRows*partyRowHeight - 1},
	{0, 0, vga.ScreenWidth - 1, vga.CanvasHeight - 1}, // the rest of the canvas
}

const (
	zoneMap = iota
	zoneIcons
	zoneStatus
	zoneParty
	zoneScreen
)

// mouseCommand returns the command of a mouse press.
// C: seg_0C9C:1714 (mkMouseCommand)
func (g *Game) mouseCommand() int {
	// the port: the map fills the screen, the other zones are the panels over it
	zone := zoneMap
	for g.pointerOverUI {
		zone++
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
		if g.mouseTriggered && dir != -1 && g.mouseButtons&leftButton != 0 && g.mouseSelection() == cmdSelectMap {
			// the port: a click on a thing opens its menu
			cmd = g.contextMenu(image.Pt(host.MouseX, host.MouseY))
			if cmd == 0 {
				cmd = cmdSpellNothing
			}
			break
		}
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
			cmd = g.clickSpellbook()
		}
	case zoneParty:
		if g.mouseTriggered {
			cmd = g.clickPartyView()
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
		g.mouseMapX = tx + mx - viewCenterX
		g.mouseMapY = ty + my - viewCenterY
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
	switch {
	case !g.pointerOverUI:
		tx, ty := g.PointerX/tileSize, g.PointerY/tileSize
		g.Selection.x = tx + int(g.MapX) - viewCenterX
		g.Selection.y = ty + int(g.MapY) - viewCenterY
		// fix: the original reads past the view for points on its border
		tx, ty = min(tx, viewTilesX-1), min(ty, viewTilesY-1)
		if g.objectView[ty][tx] != 0 || g.tileView[ty][tx] >= firstObjectTile {
			g.Selection.obj = g.objectAtForSelection(int(g.Party[g.Active]), g.Selection.x, g.Selection.y)
			return cmdSelectMap
		}
		g.Selection.obj = noObject
		t := g.tileView[ty][tx]
		g.selectedDark = t == tileHidden || t == tileDarkness
	case g.StatusDisplay == cmdStatusEquipment:
		g.Selection.x, g.Selection.y = -1, -1
		if g.pointInventory() {
			return cmdSelectInventory
		}
		if g.pointEquipment() {
			return cmdSelectEquipment
		}
	default:
		g.Selection.x, g.Selection.y, g.Selection.obj = -1, -1, noObject
	}
	return 0
}

// clickPartyView returns the command of a click on the party column: the member's
// equipment.
// C: seg_0C9C:121F (do_CMD_91), for the port's party column
func (g *Game) clickPartyView() int {
	row := (g.PointerY - partyColumnY) / partyRowHeight
	if g.PointerY < partyColumnY || row+g.partyScroll >= int(g.PartySize) {
		return 0
	}
	g.Selection.obj = row + g.partyScroll
	return cmdStatusEquipment
}

// statusButton returns the command of the buttons under the portrait and equipment views.
func (g *Game) statusButton(other int) int {
	switch {
	case g.PointerX >= buttonParty && g.PointerX < buttonSwitch:
		return cmdStatusParty
	case g.PointerX >= buttonSwitch && g.PointerX < buttonNext:
		g.Selection.obj = noObject
		return other
	case g.PointerX >= buttonNext && g.PointerX < buttonCombat:
		return cmdNextMember
	case g.PointerX >= buttonPrevious && g.PointerX < buttonParty:
		return cmdPreviousMember
	}
	return 0
}

// clickPortraitView returns the command of a click in the portrait view.
// C: seg_0C9C:12B7 (do_CMD_90)
func (g *Game) clickPortraitView() int {
	if g.PointerY >= buttonsY {
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
	if g.statusMember != 0 && g.PointerX >= buttonCombat && g.PointerX < buttonsEnd && g.PointerY >= buttonsY && g.PointerY < buttonsBottom {
		n := g.Party[g.statusMember]
		npc := &g.NPCs[n]
		npc.CombatMode++
		if npc.CombatMode == AIAssault+1 {
			npc.CombatMode = AICommand
		}
		if g.InCombat != 0 && int(g.Active) != g.statusMember && !g.object(int(n)).Has(StatusCharmed) {
			npc.Mode = npc.CombatMode
		}
		g.refreshCombatMode()
		return 0
	}
	if g.PointerY >= buttonsY {
		return g.statusButton(cmdStatusPortrait)
	}
	x, y := g.PointerX, g.PointerY
	switch {
	case x >= scrollArrowX && x < scrollArrowRight:
		switch {
		case y >= scrollUpY0 && y < scrollUpY1:
			if g.inventoryScroll != 0 {
				return cmdInventoryUp
			}
		case y >= scrollDnY0 && y < scrollDnY1:
			if g.invenSize-g.inventoryScroll > inventorySlots {
				return cmdInventoryDown
			}
		}
	case x >= containerX && x < containerX+tileSize && y >= containerY && y < containerY+tileSize && g.MouseMode == 0:
		return cmdCloseContainer
	case g.MouseMode == 0 && g.pendingCommand != cmdMove:
		picked := g.mouseSelection()
		sel := g.Selection.obj
		if sel == noObject {
			return 0
		}
		switch picked {
		case cmdSelectInventory:
			t := g.object(sel).Type()
			const lockedFrame = 2
			switch {
			case !isContainer(t) || t == ObjSpellbook:
				return cmdReady
			case t == ObjOpenChest && g.object(sel).Frame()&lockedFrame != 0:
				g.printf("It's locked.\n")
				g.printf("\n%s:\n%>", cString(g.Names[g.Active][:]))
				g.putch0(endOfLine)
			case t == ObjOpenChest && g.object(sel).Frame() == 1 && g.FindInvType(sel, ObjEffect, castTrap) != noObject:
				g.trap(sel)
				g.printf("\n%s:\n%>", cString(g.Names[g.Active][:]))
				return cmdOpenContainer
			default:
				return cmdOpenContainer
			}
		case cmdSelectEquipment:
			return cmdUnready
		}
	}
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
		if dir, ok := g.walkStep(); ok && g.atMainPrompt {
			g.AdvanceDir = dir
			return cmdAdvance
		}
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
			g.pointerOverUI = g.mouse.overUI
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
				ch = g.incantationKey(ch)
			} else {
				ch = g.selectTarget(ch)
			}
		case ch == keyTab && g.StatusDisplay != cmdStatusContainer && !g.InConversation:
			if g.StatusDisplay == cmdStatusEquipment {
				g.SelectMode = 2
				g.drawStatusCursor(1)
			} else if g.StatusDisplay != cmdStatusSpellbook {
				g.SelectMode = 1
			}
			if g.SelectMode != 0 {
				g.AimX, g.AimY = viewCenterX, viewCenterY
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
			g.pointerOverUI = false
			g.mouseSelection()
		} else if g.SelectMode == 2 && (g.StatusDisplay == cmdStatusEquipment || g.StatusDisplay == cmdStatusPortrait) {
			g.drawStatusCursor(2)
			if g.StatusDisplay == cmdStatusEquipment {
				ch = g.clickEquipmentView()
			} else {
				ch = g.clickPortraitView()
			}
			if ch == 0 && g.MouseMode == 1 && g.mouseSelection() != 0 {
				ch = cmdSelected
			}
		}
		return ch
	case ch == cmdAdvance:
		if g.SelectMode == 1 {
			x, y := g.AimX+dirStepX[g.AdvanceDir], g.AimY+dirStepY[g.AdvanceDir]
			if x >= 0 && x < viewTilesX && y >= 0 && y < viewTilesY && g.combatRange(x, y, viewCenterX, viewCenterY) <= g.SelectRange {
				g.AimX, g.AimY = x, y
				g.drawMap(true)
			} else if g.SelectRange == -1 {
				g.PointerX, g.PointerY = x*tileSize, y*tileSize
				g.pointerOverUI = false
				g.mouseSelection()
				return cmdSelected
			}
			return 0
		}
		if g.SelectMode == 2 && g.StatusDisplay == cmdStatusEquipment {
			return g.moveEquipmentCursor()
		}
		if g.SelectMode == 2 && g.StatusDisplay == cmdStatusPortrait {
			g.drawStatusCursor(0)
			g.statusCursorX = min(max(g.statusCursorX+dirStepX[g.AdvanceDir], 0), cursorButtonsRow)
			g.statusCursorY = cursorButtonsRow
			g.drawStatusCursor(1)
		}
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

// moveEquipmentCursor moves the status cursor of the equipment view in AdvanceDir: over
// the paper doll and the grid, scrolling the grid at its edges, up to the container and
// down to the buttons.
// C: seg_0C9C:1D59 (C_0C9C_1D59, the cursor keys in the equipment view)
func (g *Game) moveEquipmentCursor() int {
	x, y := g.statusCursorX+dirStepX[g.AdvanceDir], g.statusCursorY+dirStepY[g.AdvanceDir]
	const dollCentre = 1
	switch {
	case x >= 0 && x < cursorColumnsCount && y >= 0 && y < cursorRowsCount:
		g.drawStatusCursor(0)
		if x == dollCentre && y == dollCentre { // the doll's body is no slot: skip it
			x += dirStepX[g.AdvanceDir]
			y += dirStepY[g.AdvanceDir]
		}
		g.statusCursorX, g.statusCursorY = x, y
		g.drawStatusCursor(1)
	case x >= cursorDollColumns && x < cursorColumnsCount || y >= cursorRowsCount:
		const south = 4
		switch {
		case y >= cursorRowsCount && g.invenSize-g.inventoryScroll > inventorySlots && g.AdvanceDir == south:
			g.statusCursorX = x
			return cmdInventoryDown
		case y < 0 && g.inventoryScroll != 0:
			g.statusCursorX = x
			return cmdInventoryUp
		case y < 0 && !g.spellPeek:
			g.drawStatusCursor(0)
			g.statusCursorY = cursorOnContainer
			g.drawStatusCursor(1)
		case y >= cursorRowsCount && !g.spellPeek:
			g.drawStatusCursor(0)
			g.statusCursorY = cursorButtonsRow
			g.statusCursorX = min(max(x, 0), cursorLastButton)
			g.drawStatusCursor(1)
		}
	}
	return 0
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
	if ch == keyEscape && (g.StatusDisplay == cmdStatusEquipment || g.StatusDisplay == cmdStatusPortrait || g.StatusDisplay == cmdStatusContainer) {
		return cmdStatusParty // the port: Esc closes the inventory
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
			g.PointerX, g.PointerY = buttonCombat+tileSize/2, buttonsY+tileSize/2
			g.pointerOverUI = true
			g.clickEquipmentView()
			return 0
		}
		if m, ok := member(ch); ok {
			if m == g.statusMember {
				return cmdStatusParty // the port: the member's key again closes the inventory
			}
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
		return g.spellbookKey(ch)
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
		case cmdSpellPrevious, cmdSpellNext, cmdSpellPageLeft, cmdSpellPageRight:
			g.spellbookCommand(ch)
		case cmdInventoryUp, cmdInventoryDown:
			g.scrollInventory(ch == cmdInventoryDown)
			g.drawStatusCursor(1)
		case cmdCloseContainer:
			if g.container < npcCount {
				g.StatusDisplay = cmdStatusParty
				g.statusMember = 0
				g.StatusDirty = true
			} else {
				g.container = g.object(g.container).Pos.Owner()
				g.inventoryScroll = 0
				g.refreshInventory(g.container)
				g.drawStatusCursor(1)
			}
		case cmdOpenContainer:
			g.container = g.Selection.obj
			g.inventoryScroll = 0
			g.refreshInventory(g.container)
			g.drawStatusCursor(1)
		case cmdReady:
			g.ready()
			g.drawStatusCursor(1)
		case cmdUnready:
			g.unready()
			g.drawStatusCursor(1)
		case cmdSpellNothing:
		default:
			return ch
		}
		if ch != cmdSpellNothing {
			g.RefreshStatus()
		}
	}
}
