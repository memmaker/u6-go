package game

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/memmaker/u6-go/internal/sound"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
)

// The start of the game and its main loop (seg_0903 main, seg_0A33 C_0A33_1CB4).

// Messages (seg_356A).
const (
	whatMsg        = "what?\n"
	partyModeMsg   = "Party mode\n"
	failsToRespond = "`%s fails to respond.\n"
	soloModeMsg    = "Solo mode\n"
	exitToDOSMsg   = "Exit to DOS?"
	saveMsg        = "Save game? "
	passMsg        = "Pass!\n"
	notOnShipMsg   = "Not while aboard ship!\n"
	notInSoloMsg   = "Not in solo mode!\n"
	notInCombatMsg = "Not in combat mode!\n"
	notAllHereMsg  = "Not everyone is here.\n"
	soundOnMsg     = "Sound on\n"
	soundOffMsg    = "Sound off\n"
	versionMsg     = "Version 4.5\n"
	combatMsg      = "Begin combat!\n"
	breakOffMsg    = "Break off combat!\n"
	promptFormat   = "%s:\n%>"
	turnPrompt     = "\n%s:\n%>"
	noCharacterMsg = "You must first create or transfer a character. "
)

// commandNames are printed before a command's target (D_03A2).
var commandNames = [...]string{"Attack", "Cast", "Talk", "Look", "Get", "Drop", "Move", "Use"}

// the screen layout: the clips in character cells, the command icons in pixels
const (
	screenCellsX, screenCellsY = 39, 24
	partyClipX0, partyClipY0   = 22, 1
	partyClipX1, partyClipY1   = 38, 12
	messageClipX0, messageX1   = 22, 38
	messageClipY0, messageY1   = 14, 23
	commandIcons               = 9
	iconsX, iconsY             = 8, 176
	combatIconX                = 152
	// partyHereRange is how close everyone must be for party mode
	partyHereRange = 8
)

// New sets up the game in dir: the driver, the screen and the saved game. It must run on
// the host's game goroutine.
// C: seg_0903:0C5E (main), up to the game loop
func New(dir string) (*Game, error) {
	files := os.DirFS(dir)
	g := &Game{dir: dir, files: files, gr: vga.New(files), colours: defaultColours}
	g.initConsole()
	g.initInput()
	g.initWorldMap()
	g.initTurns()
	g.lastHour = noHour
	g.thrower = noObject
	g.lastTarget = noObject
	g.worldShown = true
	g.StatusDisplay = cmdStatusParty
	g.random.randomize(timeSeed(time.Now()))
	g.gr.Present = host.Yield
	if err := g.gr.LoadPointers(); err != nil {
		return nil, err
	}
	if err := g.gr.LoadTiles(); err != nil {
		return nil, err
	}
	palette, err := vga.LoadPalette(files)
	if err != nil {
		return nil, err
	}
	g.basePalette = palette
	g.gr.Palette = *palette
	if g.gr.Font, err = vga.LoadFont(files); err != nil {
		return nil, err
	}
	g.createClip(clipScreen, 0, 0, screenCellsX, screenCellsY)
	if paper, err := vga.LoadPixmap(files, "PAPER.BMP"); err == nil {
		g.gr.DrawPixmap(paper, 0, 0)
		g.paper = paper
	}
	g.createClip(clipParty, partyClipX0, partyClipY0, partyClipX1, partyClipY1)
	g.createClip(clipOptions, optionsX0, optionsY0, optionsX0+optionsColumns*optionWidth-1, optionsY0+optionsRows-1)
	g.createClip(clipMessage, messageClipX0, messageClipY0, messageX1, messageY1)
	g.setClip(clipMessage)
	g.current.attr |= clipWordWrap | clipScroll | clipUppercase
	g.resetObjectLists()
	if err := g.loadTileTables(files); err != nil {
		return nil, err
	}
	if err := g.loadMap(dir); err != nil {
		return nil, err
	}
	for i := range commandIcons {
		g.gr.DrawTile(tileCommandIcons+i, i*tileSize+iconsX, iconsY)
	}
	g.gr.DrawTile(tileCombatIcon, combatIconX, iconsY)
	g.StatusDirty = true
	if err := g.loadGame(); err != nil {
		return nil, err
	}
	for i := range g.animCount {
		g.gr.SetTileFrame(int(g.animTile[i]), int(g.animFrame[i]))
	}
	if g.PortraitPlusOne == 0 {
		g.printf(noCharacterMsg)
		return nil, errors.New("no character: run U.EXE (the intro) first")
	}
	g.startTurns()
	g.passTime(0)
	if g.InCombat != 0 {
		g.beginCombat()
	}
	g.playAreaSong()
	return g, nil
}

// commandKeys are the letters of the commands that ask for a target, with how the target
// is picked: the select mode and range.
var commandKeys = map[int]struct{ command, mode, rng int }{
	'A': {cmdAttack, 1, 7}, 'C': {cmdCast, 3, 7}, 'T': {cmdTalk, 1, 7}, 'L': {cmdLook, 1, 7},
	'G': {cmdGet, 1, -1}, 'D': {cmdDrop, 2, 7}, 'M': {cmdMove, 1, -1}, 'U': {cmdUse, 1, -1},
	'R': {cmdRest, 1, 7},
}

// Play runs the game loop until the player exits.
// C: seg_0A33:1CB4 (C_0A33_1CB4)
func (g *Game) Play() {
	g.printf(promptFormat, cString(g.Names[g.Active][:]))
	for {
		g.RefreshStatus()
		if g.MouseMode == 0 {
			g.allowMouseMove = true
		}
		g.atMainPrompt = true
		ch := g.getch()
		g.atMainPrompt = false
		g.allowMouseMove = false
		if k, ok := commandKeys[ch]; ok {
			ch = k.command
			g.drawStatusCursor(0)
			g.selectByTab = false
			g.SelectMode = k.mode
			g.AimX, g.AimY = viewCenterX, viewCenterY
			g.SelectRange = k.rng
			if ch == cmdAttack && g.lastTarget >= 0 {
				p := g.object(g.lastTarget).Pos
				g.AimX = p.X&byteMask - int(g.MapX) + viewCenterX
				g.AimY = p.Y&byteMask - int(g.MapY) + viewCenterY
				if g.AimX < 0 || g.AimX >= viewTilesX || g.AimY < 0 || g.AimY >= viewTilesY {
					g.AimX, g.AimY = viewCenterX, viewCenterY
				}
			}
			if ch == cmdDrop {
				g.statusCursorX, g.statusCursorY = 3, 0
				if !g.inVehicle() && (g.StatusDisplay != cmdStatusEquipment || int(g.Active) != g.statusMember) {
					g.StatusDisplay = cmdStatusEquipment
					g.statusMember = int(g.Active)
					g.StatusDirty = true
					g.RefreshStatus()
				}
				g.drawStatusCursor(1)
			}
		} else if ch == keyCtrlS {
			ch = cmdSave
		} else if ch == 'B' {
			ch = cmdCombat
		}
		if ch >= '0' && ch <= '9' && ch < int(g.PartySize)+'1' {
			g.switchMember(ch - '0')
		} else if g.command(ch) {
			return
		}
		// C_262D
		switch g.diskMode {
		case 'R':
			g.restoreGame()
		case 'G':
			g.saveGame()
		}
		g.runTurns()
		g.erupt()
		if ch != cmdAdvance {
			g.promptFlag = true
		}
		if g.promptFlag {
			g.printf(turnPrompt, cString(g.Names[g.Active][:]))
			g.promptFlag = false
		}
	}
}

// switchMember goes to party mode (0) or solo mode with member n (1-9).
// C: seg_0A33:1CB4 (C_0A33_1CB4), the character switch
func (g *Game) switchMember(n int) {
	defer func() {
		g.SelectMode = 0
		g.pendingCommand = cmdPass
		g.updatePointerShape()
	}()
	switch {
	case g.inVehicle():
		g.printf(notOnShipMsg)
		return
	case g.InCombat != 0:
		g.printf(notInCombatMsg)
		return
	}
	if n == 0 {
		for i := range int(g.PartySize) {
			p := g.object(int(g.Party[i])).Pos
			if abs(int(g.MapX)-p.X) >= partyHereRange || abs(int(g.MapY)-p.Y) >= partyHereRange {
				g.printf(notAllHereMsg)
				return
			}
		}
		g.printf(partyModeMsg)
		g.SetPartyMode()
		g.MouseMode = 0
		return
	}
	i := n - 1
	m := int(g.Party[i])
	if g.incapacitated(m) || g.draggedUnder(m) || g.object(m).Has(StatusCharmed) {
		g.printf(failsToRespond, cString(g.Names[i][:]))
		return
	}
	if g.SoloMember < 0 {
		g.NPCs[g.Party[g.Active]].Mode = AIMode(g.SavedMode)
	} else {
		g.NPCs[g.Party[g.SoloMember]].Mode = AIMode(g.SavedMode)
	}
	g.SavedMode = int8(g.NPCs[m].Mode)
	for k := range int(g.PartySize) {
		if g.NPCs[g.Party[k]].Mode == AIFollow {
			g.NPCs[g.Party[k]].Mode = AIMotionless
		}
	}
	g.SetActive(i)
	g.NPCs[g.Party[g.Active]].Mode = AICommand
	g.SoloMember = int8(g.Active)
	if g.StatusDisplay != cmdStatusParty {
		g.StatusDirty = true
	}
	g.printf(soloModeMsg)
	g.MouseMode = 0
	g.drawStatusCursor(0)
}

// yesNo waits for Y or N and echoes it.
func (g *Game) yesNo() bool {
	var k int
	for k != 'Y' && k != 'N' {
		k = g.getch()
	}
	g.printf("%c\n", k)
	return k == 'Y'
}

// endCommand returns the input to walking.
func (g *Game) endCommand() {
	g.drawStatusCursor(0)
	g.MouseMode = 0
	g.SelectMode = 0
	g.updatePointerShape()
}

// command runs the command ch of the game loop; it returns true to exit.
// C: seg_0A33:1CB4 (C_0A33_1CB4), the command switch
func (g *Game) command(ch int) bool {
	switch ch {
	case cmdAdvance:
		g.advance(g.AdvanceDir)
	case keyCtrlE, keyCtrlQ, keyAltX:
		g.printf(exitToDOSMsg)
		return g.yesNo()
	case cmdAttack:
		g.attack()
		g.pendingCommand = cmdPass
	case cmdCombat:
		switch {
		case g.SoloMember >= 0:
			g.printf(notInSoloMsg)
		case g.InCombat != 0:
			g.printf(breakOffMsg)
			g.breakOff()
		case g.inVehicle():
			g.printf(notOnShipMsg)
		default:
			g.printf(combatMsg)
			g.beginCombat()
		}
	case cmdCast, cmdGet, cmdDrop, cmdMove, cmdTalk, cmdLook, cmdUse:
		if ch != cmdTalk && ch != cmdLook && ch != cmdUse && g.inVehicle() {
			g.printf("%s-", commandNames[ch-cmdAttack])
			g.printf(notOnShipMsg)
			break
		}
		g.MouseMode = 1
		g.pendingCommand = ch
		g.updatePointerShape()
		g.printf("%s-", commandNames[ch-cmdAttack])
		if ch == cmdCast {
			g.letterCount = 0
			g.cast()
		} else if g.getch() == cmdSelected {
			switch g.pendingCommand {
			case cmdTalk:
				g.talkTo(int(g.Active), g.Selection.obj, true)
				g.playAreaSong()
			case cmdLook:
				g.look()
			case cmdDrop:
				g.drop()
			case cmdGet:
				g.get()
			case cmdMove:
				g.move()
			case cmdUse:
				g.use()
			}
		} else {
			g.printf(whatMsg)
		}
		g.MouseMode = 0
		g.pendingCommand = cmdPass
		g.endCommand()
	case cmdRest:
		if !g.inVehicle() {
			g.playSong(sound.SongStones)
		}
		g.rest()
		if !g.inVehicle() {
			g.playAreaSong()
		}
		g.endCommand()
	case cmdSave:
		g.printf(saveMsg)
		if g.yesNo() {
			if err := g.saveGame(); err != nil {
				g.printf("%s\n", err.Error())
			}
			g.endCommand()
		}
	case keyCtrlR:
		g.printf("Restore game? ")
		if g.yesNo() {
			g.restoreGame()
		}
	case keyEscape, ' ', cmdPass:
		g.printf(passMsg)
		g.NPCs[g.Party[g.Active]].MovePoints = 0
		g.endCommand()
		if t := g.object(int(g.Party[g.Active])).Type(); (t == ObjRaft || t == ObjBalloon2) && g.WindDir >= 0 {
			g.advance(int(g.WindDir))
		}
		g.MoveFollowers(int(g.Party[g.Active]), true)
	case keyCtrlZ:
		if g.SoundOn != 0 {
			g.playSong(0)
			g.printf(soundOffMsg)
			g.SoundOn = 0
		} else {
			g.SoundOn = 1
			g.playAreaSong()
			g.printf(soundOnMsg)
		}
	case keyCtrlH:
		g.printf("Help ")
		g.Help = !g.Help
		if g.Help {
			g.printf("on")
		} else {
			g.printf("off")
		}
		g.printf("!\n")
	case keyAltP:
		g.gr.Palette = *g.basePalette
	case keyCtrlV:
		g.printf(versionMsg)
	case keyAlt213:
		g.printf("%s\n", fmt.Sprintf("%02d%02d%02d%03x%03x%1x", g.Karma, g.Hour, g.Minute, g.MapX, g.MapY, g.MapZ))
		g.peer()
	case keyAlt214:
		g.goTo()
	case keyAlt215:
		const hour = 60
		g.passTime(hour)
		g.allowNPCTeleport = true
		g.schedulePaths()
		g.allowNPCTeleport = false
	default:
		g.printf(whatMsg)
		g.endCommand()
	}
	return false
}
