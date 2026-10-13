package game

import (
	"image"
	"testing"

	"github.com/memmaker/u6-go/internal/sound"
)

// TestNewStyleUI shows the log, the avatar's inventory in the status area and the second
// member's window, then the portrait, a menu and a sign, and writes each with the scroll
// look (ui_*_scroll.png) and the new style (ui_*_new_style.png) to $U6_PNG_DIR. The new
// style's gumps must send clicks on their parts to the status area's places under them.
func TestNewStyleUI(t *testing.T) {
	inGame(t, func(g *Game) {
		defer func() { sound.Settings.NewStyleUI = false }()
		both := func(name string) {
			sound.Settings.NewStyleUI = false
			writePNG(t, g, "ui_"+name+"_scroll.png")
			sound.Settings.NewStyleUI = true
			writePNG(t, g, "ui_"+name+"_new_style.png")
		}
		g.showLog = true
		g.StatusDisplay, g.statusMember = cmdStatusEquipment, 0
		g.StatusDirty = true
		g.RefreshStatus()
		if g.PartySize > 1 {
			g.toggleWindow(1)
		}
		both("inventory")

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
		both("portrait")

		g.popup = g.centredPopup([]string{"Save", "Load", "Settings"})
		g.popup.choice = 0
		both("menu")

		g.popup = &popup{sign: signText("<BRITAIN>"), choice: noChoice}
		both("sign")
		g.popup = nil
	})
}
