package game

import (
	"fmt"
	"image"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// TestWidescreen widens the screen to 16:9 and 21:9 windows and heightens it to a portrait
// one: the view grows to cover it with the party in the middle, the panels keep to their
// edges, and the pointer still points at the tile under it. With U6_PNG_DIR set it saves
// the frames.
func TestWidescreen(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Fit(host.ScreenWidth, host.ScreenHeight)
	settle := func() {
		for range settleTicks {
			host.Step()
		}
	}
	settle()
	removeGargoyles(g)
	// out of the castle, where there is more to see
	const fieldX, fieldY = 307, 440
	for k := range int(g.PartySize) {
		g.MoveObj(int(g.Party[k]), fieldX+k, fieldY, 0)
	}
	g.MapX, g.MapY = fieldX, fieldY
	g.enterArea()
	for _, window := range []struct {
		name string
		w, h int
		free int // inventory windows that fit beside the status area's gump
	}{
		{"21x9", 2560, 1080, 1}, {"portrait", 1080, 1920, 2}, {"16x9", 1920, 1080, 0}, {"320x200", 1280, 960, 0},
	} {
		host.Fit(window.w, window.h)
		settle()
		// the last size's windows stay on the new screen
		g.Frame()
		assertPanels(t, g, window.name+" resized", false)
		g.windows = nil
		if g.screenWidth != host.FitWidth || g.screenHeight != host.FitHeight {
			t.Fatalf("%s: screen %dx%d, want %dx%d", window.name, g.screenWidth, g.screenHeight, host.FitWidth, host.FitHeight)
		}
		if g.viewTilesX*tileSize < g.screenWidth || g.viewTilesY*tileSize < g.screenHeight ||
			g.viewTilesX%2 == 0 || g.viewTilesY%2 == 0 {
			t.Errorf("%s: %dx%d tiles do not cover the screen around the party", window.name, g.viewTilesX, g.viewTilesY)
		}
		// the tile right of the party, through the pointer
		v := g.viewPixels()
		x, y := (g.viewCenterX+1)*tileSize+tileSize/2-v.Min.X, g.viewCenterY*tileSize+tileSize/2-v.Min.Y
		if bx, by, overUI := g.pointAt(x, y); overUI || bx/tileSize-g.viewCenterX != 1 || by/tileSize != g.viewCenterY {
			t.Errorf("%s: the pointer at the tile right of the party maps to tile (%d, %d)", window.name, bx/tileSize, by/tileSize)
		}
		x, y = int(g.MapX)+g.viewCenterX, int(g.MapY)
		if !g.onScreen(x, y) || g.onScreen(x+1, y) {
			t.Errorf("%s: onScreen does not follow the view", window.name)
		}
		_, panels := g.panels()
		for _, p := range panels {
			at := p.onScreen()
			if p.from == skyPanel && at.Min.X != (g.screenWidth-skyPanel.Dx())/2 ||
				p.from.Min == partyPanel.Min && at.Max.X != g.screenWidth-margin {
				t.Errorf("%s: a panel left its edge: %v", window.name, at)
			}
		}
		writePNG(t, g, fmt.Sprintf("widescreen_%s.png", window.name))

		// the status area's gump, the date, the log and the windows that fit: apart
		g.StatusDisplay, g.statusMember, g.showDate, g.showLog = cmdStatusEquipment, 0, true, true
		g.StatusDirty = true
		g.RefreshStatus()
		for m := 1; m <= window.free && m < int(g.PartySize); m++ {
			g.toggleWindow(m)
		}
		assertPanels(t, g, window.name, window.name != "320x200")
		writePNG(t, g, fmt.Sprintf("widescreen_%s_inventory.png", window.name))
		// one more window than fits: on the screen still
		if m := window.free + 1; m < int(g.PartySize) {
			g.toggleWindow(m)
			assertPanels(t, g, window.name+" crowded", false)
		}
		g.StatusDisplay, g.showDate, g.showLog = cmdStatusParty, false, false
		g.StatusDirty = true
		g.RefreshStatus()
	}
}

// assertPanels fails when a panel leaves the screen or, with apart, two overlap.
func assertPanels(t *testing.T, g *Game, name string, apart bool) {
	t.Helper()
	_, panels := g.panels()
	screen := image.Rect(0, 0, g.screenWidth, g.screenHeight)
	for i, p := range panels {
		if !p.outer().In(screen) {
			t.Errorf("%s: panel %v at %v leaves the screen", name, p.from, p.outer())
		}
		for _, q := range panels[:i] {
			if apart && p.outer().Overlaps(q.outer()) {
				t.Errorf("%s: panels at %v and %v overlap", name, p.outer(), q.outer())
			}
		}
	}
}
