package game

import "testing"

// TestQuickSaveAndLoad quick saves, changes the game, saves it normally and quick loads:
// the quick save comes back.
func TestQuickSaveAndLoad(t *testing.T) {
	inGame(t, func(g *Game) {
		avatar := int(g.Party[0])
		saved := g.NPCs[avatar].HitPoints
		g.quickSave()
		g.NPCs[avatar].HitPoints = saved + 1
		if err := g.saveGame(); err != nil {
			t.Fatal(err)
		}
		g.quickLoad()
		if g.NPCs[avatar].HitPoints != saved {
			t.Errorf("hit points %d after quick load, quick saved %d", g.NPCs[avatar].HitPoints, saved)
		}
		g.popup = g.centredPopup([]string{"Save", "Load", "Quick save", "Quick load", "Load latest", "Settings", "Quit"})
		g.popup.choice = 0
		writePNG(t, g, "gamemenu.png")
		g.popup = nil
		if got := g.gameMenuKey(keyF11); got != cmdQuickSave {
			t.Errorf("F11: %d", got)
		}
	})
}
