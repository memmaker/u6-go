package game

import (
	"image"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// TestDragBetweenWindows opens the second member's window, drags a dagger of the avatar's
// onto it (Move, run as the command runs) and back onto the avatar's doll (Ready), and
// writes the screen to windows.png ($U6_PNG_DIR).
// It runs in the scroll look and again in the new style UI (windows_new_style.png).
func TestDragBetweenWindows(t *testing.T) {
	inGame(t, func(g *Game) {
		if g.PartySize < 2 {
			t.Skip("a party of one")
		}
		defer func() { sound.Settings.NewStyleUI = false }()
		for _, newStyle := range []bool{false, true} {
			sound.Settings.NewStyleUI = newStyle
			g.windows = nil
			dragBetweenWindows(t, g)
		}
	})
}

// windowScreenPoint is where the status area's point pt shows in window w.
func (g *Game) windowScreenPoint(w *inventoryWindow, pt image.Point) image.Point {
	if g.newStyle() {
		if at, ok := inventoryGump(w.info).screenPoint(pt, w.at); ok {
			return at
		}
	}
	return w.at.Add(pt.Sub(statusPanel.Min))
}

func dragBetweenWindows(t *testing.T, g *Game) {
	{
		avatar, second := int(g.Party[0]), int(g.Party[1])
		dagger := g.AddInvObj(shapeOf(ObjDagger, 0), StatusLocal|InInventory, 0, avatar)
		g.StatusDisplay, g.statusMember = cmdStatusEquipment, 0
		g.StatusDirty = true
		g.RefreshStatus()
		g.toggleWindow(1)
		if len(g.windows) != 1 || g.windows[0].pixels == nil {
			t.Fatal("no window for the second member")
		}
		// the pointer over the window's grid
		w := g.windows[0]
		at := g.windowScreenPoint(w, image.Pt(inventoryX, inventoryY))
		host.MouseX, host.MouseY = at.X+1, at.Y+1
		g.dragging, g.draggedTile = true, g.tileOf(dagger)
		if g.newStyle() {
			writePNG(t, g, "windows_new_style.png")
		} else {
			writePNG(t, g, "windows.png")
		}
		g.dragging = false
		if cmd := g.dropOn(selection{obj: dagger}, false); cmd != 'M' || g.queuedTargets[1].obj != second {
			t.Fatalf("drop on the window: command %d, targets %v", cmd, g.queuedTargets)
		}
		if cmd := g.dropOn(selection{obj: dagger}, true); cmd != 'G' {
			t.Errorf("a thing of the map dropped on the window: command %d, want Get", cmd)
		}
		g.queuedTargets = []selection{{obj: dagger}, {obj: second}}
		g.command(cmdMove)
		if owner := g.object(dagger).Pos.Owner(); owner != second {
			t.Fatalf("the dagger is held by %d, not %d:\n%s", owner, second, scrollText(g))
		}
		// from the second's window (now the status area) onto its doll
		g.StatusDisplay, g.statusMember = cmdStatusEquipment, 1
		g.StatusDirty = true
		g.RefreshStatus()
		at = g.windowScreenPoint(w, image.Pt(slotPlaces[slotRightHand][0], slotPlaces[slotRightHand][1]))
		host.MouseX, host.MouseY = at.X+1, at.Y+1
		if cmd := g.dropOn(selection{obj: dagger}, false); cmd != cmdReady {
			t.Errorf("drop on the doll: command %d", cmd)
		}
	}
}
