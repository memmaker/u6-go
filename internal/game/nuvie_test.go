package game

import (
	"image"
	"testing"
)

// TestUI shows the log, the avatar's inventory in the status area and the second member's
// window, then the portrait, a menu and a sign, and writes each to $U6_PNG_DIR (ui_*.png).
// The gumps must send clicks on their parts to the status area's places under them.
func TestUI(t *testing.T) {
	inGame(t, func(g *Game) {
		shot := func(name string) { writePNG(t, g, "ui_"+name+".png") }
		g.showLog = true
		g.StatusDisplay, g.statusMember = cmdStatusEquipment, 0
		g.StatusDirty = true
		g.RefreshStatus()
		if g.PartySize > 1 {
			g.toggleWindow(1)
		}
		shot("inventory")

		gp := inventoryGump(g.inventoryInfo())
		for _, pt := range []image.Point{
			{inventoryX + 1, inventoryY + 1},
			{slotPlaces[slotHead][0] + 1, slotPlaces[slotHead][1] + 1},
		} {
			at, ok := gp.screenPoint(pt, image.Point{})
			if !ok || g.gumpCanvasPoint(gp, at) != pt {
				t.Errorf("status point %v shows at %v (%v) and maps back to %v", pt, at, ok, g.gumpCanvasPoint(gp, at))
			}
		}

		g.windows = nil
		g.showLog = false
		g.StatusDisplay = cmdStatusPortrait
		g.StatusDirty = true
		g.RefreshStatus()
		shot("portrait")

		g.popup = g.centredPopup([]string{"Save", "Load", "Settings"})
		g.popup.choice = 0
		shot("menu")

		g.popup = &popup{sign: signText("<BRITAIN>"), choice: noChoice}
		shot("sign")
		g.popup = nil
	})
}
