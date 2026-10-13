package game

import (
	"image"
	"slices"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
)

// The port's inventory windows and dragging (after Nuvie's doll and container gumps and its
// drag manager). A window shows a party member's equipment view as the status area would;
// a click in it first turns the status area to that member, so the game's own clicks,
// commands and checks then apply. A drag runs the command that does the same: Get from the
// map, Drop on the map, Move to another member or into a container, Ready on the doll,
// Unready off it.

const (
	// windowTitleHeight is the row of the member's name: dragging it moves the window
	windowTitleHeight = cellSize
	// dragDistance is how far the pointer moves, held, before a press becomes a drag
	dragDistance = 4
)

// selection is a target picked for a command: a map place or slot, and an object.
type selection struct{ x, y, obj int }

// inventoryWindow is a party member's floating equipment view: where it is and its picture,
// the status area's size.
type inventoryWindow struct {
	member int
	at     image.Point
	pixels []byte
}

// toggleWindow opens or closes member's window (Shift+F1..F9).
func (g *Game) toggleWindow(member int) {
	for i, w := range g.windows {
		if w.member == member {
			g.windows = slices.Delete(g.windows, i, i+1)
			return
		}
	}
	edge := margin + scrollPadding
	step := len(g.windows) * tileSize
	w := &inventoryWindow{member: member}
	g.placeWindow(w, image.Pt(edge+step, edge+step))
	g.windows = append(g.windows, w)
	g.refreshWindows()
}

// placeWindow moves w's corner to at, kept on the screen.
func (g *Game) placeWindow(w *inventoryWindow, at image.Point) {
	edge := margin + scrollPadding
	w.at.X = min(max(at.X, edge), g.screenWidth-edge-statusPanel.Dx())
	w.at.Y = min(max(at.Y, edge), g.screenHeight-edge-statusPanel.Dy())
}

// refreshWindows draws each window's member's equipment view on the canvas and keeps the
// picture, then puts back the status area and its state.
func (g *Game) refreshWindows() {
	g.windows = slices.DeleteFunc(g.windows, func(w *inventoryWindow) bool { return w.member >= int(g.PartySize) })
	if len(g.windows) == 0 || g.InConversation {
		return
	}
	shown := g.canvasPixels(statusPanel)
	inv, member, display, current := g.inventory, g.statusMember, g.StatusDisplay, g.current
	for _, w := range g.windows {
		g.statusMember, g.StatusDisplay, g.showNPC = w.member, cmdStatusEquipment, 0
		g.equipmentView()
		w.pixels = g.canvasPixels(statusPanel)
	}
	g.inventory, g.statusMember, g.StatusDisplay, g.current = inv, member, display, current
	for y := range statusPanel.Dy() {
		at := (statusPanel.Min.Y+y)*vga.ScreenWidth + statusPanel.Min.X
		copy(g.gr.Screen[at:at+statusPanel.Dx()], shown[y*statusPanel.Dx():])
	}
}

// canvasPixels copies rectangle r of the canvas.
func (g *Game) canvasPixels(r image.Rectangle) []byte {
	pixels := make([]byte, 0, r.Dx()*r.Dy())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		pixels = append(pixels, g.gr.Screen[y*vga.ScreenWidth+r.Min.X:y*vga.ScreenWidth+r.Max.X]...)
	}
	return pixels
}

// focusWindow turns the status area to the member of the window under the pointer.
func (g *Game) focusWindow() {
	w := g.pointedWindow
	if w == nil || g.StatusDisplay == cmdStatusEquipment && g.statusMember == w.member && g.showNPC == 0 {
		return
	}
	g.StatusDisplay, g.statusMember, g.showNPC = cmdStatusEquipment, w.member, 0
	g.StatusDirty = true
	g.RefreshStatus()
}

// drag follows a press of the left button at the command prompt. A press on a window's
// title moves the window; a press on a thing that the pointer then carries away drops it
// where the button is let go and returns the command that does so. It returns false when
// the press was a click.
func (g *Game) drag() (int, bool) {
	start := image.Pt(host.MouseX, host.MouseY)
	if w := g.pointedWindow; w != nil && g.PointerY < statusPanel.Min.Y+windowTitleHeight {
		grip := start.Sub(w.at)
		for host.MouseLeft {
			g.placeWindow(w, image.Pt(host.MouseX, host.MouseY).Sub(grip))
			g.osiDelay(1)
			g.gr.MovePointer(host.MouseX, host.MouseY)
		}
		return 0, true
	}
	pointer, overUI := image.Pt(g.PointerX, g.PointerY), g.pointerOverUI
	picked := g.mouseSelection()
	source := g.Selection
	if picked == 0 || source.obj == noObject || source.obj == g.container {
		return 0, false
	}
	for {
		if !host.MouseLeft {
			return 0, false
		}
		d := image.Pt(host.MouseX, host.MouseY).Sub(start)
		if abs(d.X) > dragDistance || abs(d.Y) > dragDistance {
			break
		}
		g.osiDelay(1)
		g.gr.MovePointer(host.MouseX, host.MouseY)
	}
	g.dragging, g.draggedTile = true, g.tileOf(source.obj)
	for host.MouseLeft {
		g.osiDelay(1)
		g.gr.MovePointer(host.MouseX, host.MouseY)
	}
	g.dragging = false
	cmd := g.dropOn(source, picked == cmdSelectMap)
	if cmd == 'G' { // Get turns toward the pointer: toward the thing
		g.PointerX, g.PointerY, g.pointerOverUI = pointer.X, pointer.Y, overUI
	}
	return cmd, true
}

// dropOn returns the command that puts source where the pointer is, with its targets
// queued, or 0 when nothing does.
func (g *Game) dropOn(source selection, fromMap bool) int {
	g.PointerX, g.PointerY, g.pointerOverUI = g.pointAt(host.MouseX, host.MouseY)
	w := g.pointedWindow
	switch {
	case fromMap && g.pointerOverUI: // into an inventory: Get, by the active member
		g.queuedTargets = []selection{source}
		return 'G'
	case fromMap:
		return 0
	case !g.pointerOverUI: // the active member drops; Drop shows the active member's things
		if g.statusMember != int(g.Active) {
			return 0
		}
		g.mouseSelection()
		g.queuedTargets = []selection{source, g.Selection}
		return 'D'
	case w != nil && w.member != g.statusMember:
		g.queuedTargets = []selection{source, {obj: int(g.Party[w.member])}}
		return 'M'
	case w == nil && !image.Pt(g.PointerX, g.PointerY).In(statusPanel):
		return 0
	}
	equipped := g.object(source.obj).CoordUse() == Equipped
	switch g.mouseSelection() {
	case cmdSelectEquipment:
		if !equipped {
			g.Selection = source
			return cmdReady
		}
	case cmdSelectInventory:
		if into := g.Selection.obj; into != noObject && into != source.obj && isContainer(g.object(into).Type()) {
			g.queuedTargets = []selection{source, {obj: into}}
			return 'M'
		}
		if equipped {
			g.Selection = source
			return cmdUnready
		}
	}
	return 0
}

// drawDragged draws the dragged thing under the pointer.
func (g *Game) drawDragged() {
	if g.dragging {
		g.gr.DrawTileOn(g.shown, g.screenWidth, g.draggedTile, host.MouseX-tileSize/2, host.MouseY-tileSize/2)
	}
}
