package game

import "testing"

// TestViewChecks: the wider view counts as on screen, and eggs hatch only past it plus the margin.
func TestViewChecks(t *testing.T) {
	inGame(t, func(g *Game) {
		x, y := int(g.MapX), int(g.MapY)
		if !g.onScreen(x+g.viewCenterX, y+g.viewCenterY) || g.onScreen(x+g.viewCenterX+1, y) || g.onScreen(x, y+g.viewCenterY+1) {
			t.Error("onScreen does not match the visible view")
		}
		const margin = 8 - classicViewCenter
		if !g.inView(margin, x+g.viewCenterX+margin, y) || g.inView(margin, x+g.viewCenterX+margin+1, y) {
			t.Error("egg hatch distance does not follow the visible view")
		}
	})
}
