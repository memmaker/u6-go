package game

import (
	"testing"

	"github.com/memmaker/u6-go/internal/sound"
)

// TestNewStyleUI opens the log and an inventory window and writes the screen with the
// scroll (ui_scroll.png) and with Nuvie's border (ui_new_style.png) to $U6_PNG_DIR.
func TestNewStyleUI(t *testing.T) {
	inGame(t, func(g *Game) {
		defer func() { sound.Settings.NewStyleUI = false }()
		g.showLog = true
		g.toggleWindow(0)
		writePNG(t, g, "ui_scroll.png")
		sound.Settings.NewStyleUI = true
		writePNG(t, g, "ui_new_style.png")
		if g.border[borderTopLeft] == nil {
			t.Error("the border was not loaded")
		}
	})
}
