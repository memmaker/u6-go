package game

import (
	"image"
	"strings"

	"github.com/memmaker/u6-go/internal/host"
)

// The port's own overlays, drawn over the composed screen: the help scroll, the command
// menus, the dock's tooltips and the message toasts.

const (
	// toastSeconds is how long a toast stays, toastsShown how many are shown at most and
	// toastColumns how wide one is, in characters
	toastSeconds = 4
	toastsShown  = 5
	toastColumns = 30
	// noChoice is a popup without a highlighted line, or a cancelled menu
	noChoice = -1
)

// helpText is the help scroll (Ctrl+H or '?').
var helpText = []string{
	"A)ttack  C)ast  T)alk  L)ook",
	"G)et  D)rop  M)ove  U)se",
	"R)est  B)egin/break off combat",
	"Arrows walk, space passes",
	"1-9 solo, 0 party mode",
	"F1-F9 inventory, again: close",
	"Tab aim with the keys",
	"Esc cancel, close inventory",
	"Enter    command menu",
	"Click    a thing's actions",
	"Hold     walk",
	"Right    default command",
	"Alt+key  set the default",
	"Bottom   command dock",
	"Ctrl+S   save, Ctrl+R restore",
	"Ctrl+D   date",
	"Ctrl+L   message log",
	"Ctrl+F   party formation",
	"Ctrl+Z   sound, Ctrl+Q quit",
	"Ctrl+H,? this help",
}

// dockNames are the names of the dock's icons, from cmdAttack.
var dockNames = [...]string{"Attack", "Cast", "Talk", "Look", "Get", "Drop", "Move", "Use", "Rest", "Combat"}

// popup is a scroll of lines over the screen; choice is the highlighted line.
type popup struct {
	lines  []string
	at     image.Point
	choice int
}

// textRect is the popup's text on the screen.
func (p *popup) textRect() image.Rectangle {
	columns := 0
	for _, l := range p.lines {
		columns = max(columns, len(l))
	}
	return image.Rect(0, 0, columns*cellSize, len(p.lines)*cellSize).Add(p.at)
}

// toast is a message shown for a while at the bottom left.
type toast struct {
	lines []string
	until uint64 // host tick
}

// menuItem is a command of a menu and the key that runs it.
type menuItem struct {
	name string
	key  int
}

// placePopup makes a popup of lines with its text's corner at at, kept on the screen.
func placePopup(lines []string, at image.Point) *popup {
	p := &popup{lines: lines, choice: noChoice}
	r := p.textRect()
	edge := margin + scrollPadding
	p.at.X = min(max(at.X, edge), host.ScreenWidth-edge-r.Dx())
	p.at.Y = min(max(at.Y, edge), host.ScreenHeight-edge-r.Dy())
	return p
}

// centredPopup makes a popup of lines in the middle of the screen.
func centredPopup(lines []string) *popup {
	r := (&popup{lines: lines}).textRect()
	return placePopup(lines, image.Pt((host.ScreenWidth-r.Dx())/2, (host.ScreenHeight-r.Dy())/2))
}

// waitRelease waits until no mouse button is held.
func (g *Game) waitRelease() {
	for host.MouseLeft || host.MouseRight {
		g.osiDelay(1)
	}
}

// runPopup shows p until a pick: with a menu (choice set) the arrows and the pointer choose,
// Enter or a click picks, Esc or a click outside cancels (noChoice); without one any key or
// click closes it.
func (g *Game) runPopup(p *popup) int {
	g.popup = p
	defer func() { g.popup = nil }()
	g.gr.SetPointer(pointerArrow)
	g.waitRelease()
	host.KeyQueue = host.KeyQueue[:0]
	menu := p.choice != noChoice
	for {
		g.osiDelay(1)
		g.gr.MovePointer(host.MouseX, host.MouseY)
		pt, r := image.Pt(host.MouseX, host.MouseY), p.textRect()
		if menu && pt.In(r) {
			p.choice = (pt.Y - r.Min.Y) / cellSize
		}
		if host.MouseLeft || host.MouseRight {
			g.waitRelease()
			if menu && pt.In(r) {
				return p.choice
			}
			return noChoice
		}
		switch ch := readKey(false); {
		case ch == 0:
		case !menu || ch == keyEscape:
			return noChoice
		case ch == keyUp:
			p.choice = (p.choice + len(p.lines) - 1) % len(p.lines)
		case ch == keyDown:
			p.choice = (p.choice + 1) % len(p.lines)
		case ch == keyEnter:
			return p.choice
		}
	}
}

// runMenu shows items in a menu at at and returns the key of the one picked, or 0.
func (g *Game) runMenu(items []menuItem, at image.Point) int {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	p := placePopup(names, at)
	p.choice = 0
	if i := g.runPopup(p); i != noChoice {
		return items[i].key
	}
	return 0
}

// showHelp shows the help scroll until a key or a click.
func (g *Game) showHelp() { g.runPopup(centredPopup(helpText)) }

// commandMenu is the Enter menu: the commands that do something now. It returns the key of
// the one picked, as if it was typed.
func (g *Game) commandMenu() int {
	items := g.availableCommands()
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	return g.runMenu(items, centredPopup(names).at)
}

// availableCommands are the commands the game would not refuse at once, checked as the
// commands check them.
func (g *Game) availableCommands() []menuItem {
	var items []menuItem
	add := func(ok bool, name string, key int) {
		if ok {
			items = append(items, menuItem{name, key})
		}
	}
	vehicle := g.inVehicle()
	ship := vehicle && g.object(int(g.Party[g.Active])).Type() == ObjShip
	party := g.SoloMember < 0
	add(!vehicle || ship, "Attack", 'A')
	if !vehicle && g.SpellFx[spellNegateMagic] == 0 {
		book, _ := g.heldSpellbooks()
		add(book != noObject, "Cast", 'C')
	}
	add(true, "Talk", 'T')
	add(true, "Look", 'L')
	add(!vehicle, "Get", 'G')
	add(!vehicle, "Drop", 'D')
	add(!vehicle, "Move", 'M')
	add(true, "Use", 'U')
	add(g.InCombat == 0 && party && (!vehicle || ship) &&
		!g.notWilderness() && !g.foesNear() && !g.othersNear() && !g.noRoom(), "Rest", 'R')
	add(party && g.InCombat == 0 && !vehicle, "Begin combat", 'B')
	add(party && g.InCombat != 0, "Break off combat", 'B')
	add(!party && g.InCombat == 0 && !vehicle, "Party mode", '0')
	add(party && !vehicle, "Formation", keyCtrlF)
	add(true, "Pass", ' ')
	add(true, "Save game", keyCtrlS)
	add(true, "Help", '?')
	return items
}

// contextMenu is the menu of a click on the world: the actions for the Selection. It returns
// the command picked, or 0; Walk here starts walking instead.
func (g *Game) contextMenu(at image.Point) int {
	obj := g.Selection.obj
	thing := obj != noObject
	npc := thing && obj < npcCount
	member := npc && g.isPlayerControlled(obj)
	var items []menuItem
	add := func(ok bool, name string, cmd int) {
		if ok {
			items = append(items, menuItem{name, cmd})
		}
	}
	add(true, "Look", cmdLook)
	add(npc && obj != int(g.Party[g.Active]), "Talk", cmdTalk)
	add(thing && g.gettable(obj), "Get", cmdGet)
	add(thing, "Use", cmdUse)
	add(npc && !member, "Attack", cmdAttack)
	add(thing && !g.inVehicle(), "Move", cmdMove)
	add(true, "Walk here", cmdAdvance)
	target := image.Pt(g.Selection.x, g.Selection.y)
	cmd := g.runMenu(items, at)
	if cmd == cmdAdvance {
		g.walkTo, g.walking, g.moveBlocked = target, true, false
		return 0
	}
	g.repeatTarget = cmd != 0 // the command's target is the one clicked
	return cmd
}

// gettable says whether Get could lift obj, as get checks it.
func (g *Game) gettable(obj int) bool {
	if obj < npcCount {
		return false
	}
	t := g.object(obj).Type()
	weight := g.weightOf(obj)
	return (weight != 0 || isLight(t)) && weight < unliftable && t != ObjDragon
}

// walkStep returns the direction of the next step toward the place picked by Walk here, and
// stops walking there, when blocked or on a key or a click.
func (g *Game) walkStep() (int, bool) {
	if !g.walking {
		return 0, false
	}
	dx, dy := g.walkTo.X-int(g.MapX), g.walkTo.Y-int(g.MapY)
	if dx == 0 && dy == 0 || g.moveBlocked || len(host.KeyQueue) > 0 || host.MouseLeft || host.MouseRight {
		g.walking = false
		return 0, false
	}
	sign := func(v int) int { return min(max(v, -1), 1) }
	for d := range dirStepX {
		if dirStepX[d] == sign(dx) && dirStepY[d] == sign(dy) {
			return d, true
		}
	}
	return 0, false
}

// toastChar collects the log's text into toasts, a line at a time; the prompts ("Name:")
// and blank lines are left out.
func (g *Game) toastChar(ch int) {
	if ch >= 0 && ch != '\n' {
		g.toastLine = append(g.toastLine, byte(ch&^highBit))
		return
	}
	line := strings.TrimSpace(string(g.toastLine))
	g.toastLine = g.toastLine[:0]
	if line == "" || strings.HasSuffix(line, ":") {
		return
	}
	g.toasts = append(g.toasts, toast{wrap(line, toastColumns), host.Ticks + toastSeconds*host.TicksPerSecond})
	if len(g.toasts) > toastsShown {
		g.toasts = g.toasts[len(g.toasts)-toastsShown:]
	}
}

// wrap breaks s into lines of at most columns characters at spaces.
func wrap(s string, columns int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > columns {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	return append(lines, line)
}

// drawText writes s on the composed screen with its corner at (x, y), in colour fg over
// what is there.
func (g *Game) drawText(s string, x, y int, fg byte) {
	for i := range len(s) {
		for row, bits := range g.gr.Font[s[i]] {
			for col := range cellSize {
				px, py := x+i*cellSize+col, y+row
				if bits&(highBit>>col) != 0 && px >= 0 && py >= 0 && px < host.ScreenWidth && py < host.ScreenHeight {
					g.shown[py*host.ScreenWidth+px] = fg
				}
			}
		}
	}
}

// drawPopup draws p: a scroll with its lines, the chosen one highlighted.
func (g *Game) drawPopup(p *popup) {
	r := p.textRect()
	g.drawScroll(r.Inset(-scrollPadding))
	for i, l := range p.lines {
		fg := g.colours.text
		if i == p.choice {
			fg = g.colours.highlight
		}
		g.drawText(l, r.Min.X, r.Min.Y+i*cellSize, fg)
	}
}

// drawOverlays draws the toasts, the dock's tooltip and the popup over the screen.
func (g *Game) drawOverlays() {
	if g.gr.Font == nil {
		return
	}
	for len(g.toasts) > 0 && g.toasts[0].until <= host.Ticks {
		g.toasts = g.toasts[1:]
	}
	if !g.logShown() && !g.InConversation {
		y := host.ScreenHeight - margin - g.dockOffset
		for i := len(g.toasts) - 1; i >= 0; i-- {
			p := &popup{lines: g.toasts[i].lines, choice: noChoice}
			y -= p.textRect().Dy() + 2*scrollPadding
			p.at = image.Pt(margin+scrollPadding, y+scrollPadding)
			g.drawPopup(p)
		}
	}
	if name := g.dockTooltip(); name != "" {
		r := (&popup{lines: []string{name}}).textRect()
		at := image.Pt(host.MouseX-r.Dx()/2, host.ScreenHeight-g.dockOffset-margin-scrollPadding-r.Dy())
		g.drawPopup(placePopup([]string{name}, at))
	}
	if g.popup != nil {
		g.drawPopup(g.popup)
	}
}

// dockTooltip is the name of the dock icon under the pointer, or "".
func (g *Game) dockTooltip() string {
	if g.dockOffset < iconsPanel.Dy() || g.popup != nil {
		return ""
	}
	dockX := (host.ScreenWidth - iconsPanel.Dx()) / 2
	x := host.MouseX - dockX + iconsPanel.Min.X - iconsX
	if host.MouseY < host.ScreenHeight-g.dockOffset || x < 0 || x/tileSize >= len(dockNames) {
		return ""
	}
	i := x / tileSize
	switch {
	case cmdAttack+i != cmdCombat:
		return dockNames[i]
	case g.InCombat != 0:
		return "Break off combat"
	}
	return "Begin combat"
}
