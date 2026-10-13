package game

import (
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// TestLighting plays a night in the castle and a dungeon by a lamp with the lighting
// options off and on: smooth lighting must shade the dark in more than the original's few
// levels, and map tile lighting must not darken anything. With U6_PNG_DIR set it saves the
// frames.
func TestLighting(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { sound.Settings = sound.Options{} }()
	settle := func() {
		for range settleTicks {
			host.Step()
		}
	}
	settle()
	removeGargoyles(g)
	shoot := func(scene string) {
		for _, smooth := range []bool{false, true} {
			sound.Settings.SmoothLighting, sound.Settings.MapTileLighting = smooth, smooth
			g.computeView()
			settle()
			name := "lighting_" + scene + "_original.png"
			shades := map[byte]bool{}
			for _, s := range g.Frame().Shade {
				shades[s] = true
			}
			if smooth {
				name = "lighting_" + scene + "_smooth.png"
				const fewestShades = 8
				if len(shades) < fewestShades {
					t.Errorf("%s: smooth lighting shows %d shades", scene, len(shades))
				}
			} else if len(shades) != 0 {
				t.Errorf("%s: the original lighting shades pixels", scene)
			}
			writePNG(t, g, name)
		}
	}

	const midnight = 23
	g.Hour = midnight
	g.passTime(0)
	shoot("night")

	// by the first lamp found on the first dungeon level with room to stand below it
	const dungeon, levelSize = 1, 2 * regionSize
	g.partyTeleport(levelSize/2, levelSize/2, dungeon, 0)
	var lamps [][2]int
	for n := g.SearchArea(0, 0, levelSize-1, levelSize-1); n >= 0; n = g.NextArea() {
		if o := g.object(n); g.tileLight(g.tileFrame(n)) != 0 {
			lamps = append(lamps, [2]int{o.Pos.X, o.Pos.Y})
		}
	}
	for _, at := range lamps {
		if !g.blockedAt(at[0], at[1]+1) {
			g.partyTeleport(at[0], at[1]+1, dungeon, 0)
			break
		}
	}
	if g.MapZ != dungeon || len(lamps) == 0 {
		t.Fatal("no lamp in the dungeon")
	}
	shoot("dungeon")
}
