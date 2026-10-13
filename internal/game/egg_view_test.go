package game

import "testing"

// TestViewChecks: the wider view counts as on screen, and eggs hatch only past it plus the margin.
func TestViewChecks(t *testing.T) {
	inGame(t, func(g *Game) {
		x, y := int(g.MapX), int(g.MapY)
		if !g.onScreen(x+viewCenterX, y+viewCenterY) || g.onScreen(x+viewCenterX+1, y) || g.onScreen(x, y+viewCenterY+1) {
			t.Error("onScreen does not match the visible view")
		}
		const margin = 8 - classicViewCenter
		if !g.inView(margin, x+viewCenterX+margin, y) || g.inView(margin, x+viewCenterX+margin+1, y) {
			t.Error("egg hatch distance does not follow the visible view")
		}
	})
}
