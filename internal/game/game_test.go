package game

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// dataCopy copies the game files ($U6DATA, default ~/Projects/u6-re/gog/inst) to a temp
// dir, so the tests never write to the originals.
func dataCopy(t *testing.T) string {
	t.Helper()
	src := os.Getenv("U6DATA")
	if src == "" {
		home, _ := os.UserHomeDir()
		src = filepath.Join(home, "Projects/u6-re/gog/inst")
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Skip("no game data:", err)
	}
	dst := t.TempDir()
	for _, e := range entries {
		if e.IsDir() || e.Name() == mapFile {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

func TestCoordRoundTrip(t *testing.T) {
	for _, c := range []Coord{{0, 0, 0}, {1023, 1023, 15}, {307, 352, 0}, {5, 900, 3}} {
		if got := decodeCoord(encodeCoord(c)); got != c {
			t.Errorf("%v -> %v", c, got)
		}
	}
}

func TestRandRange(t *testing.T) {
	g := &Game{}
	g.random.randomize(1)
	for range 1000 {
		if v := g.rand(3, 7); v < 3 || v > 7 {
			t.Fatalf("rand(3, 7) = %d", v)
		}
	}
}

func TestFaceTurnsToNearestSide(t *testing.T) {
	// straight directions face that way; a diagonal keeps a facing next to it
	if face(4, 0) != 2 || face(1, 0) != 0 || face(1, 2) != 0 || face(3, 3) != 1 {
		t.Error(face(4, 0), face(1, 0), face(1, 2), face(3, 3))
	}
}

func TestGameFilesAndMap(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	f, err := readObjlist(filepath.Join(dir, saveDir, objlistFile))
	if err != nil {
		t.Fatal(err)
	}
	if f.Block.PortraitPlusOne != 1 || cString(f.Names[0][:]) != "Avatar" {
		t.Fatalf("character not written: %d %q", f.Block.PortraitPlusOne, f.Names[0])
	}
	// the objlist survives a round trip through Game
	g := &Game{}
	g.fromObjlist(f)
	if *g.toObjlist() != *f {
		t.Error("objlist round trip differs")
	}
	var w worldMap
	if err := w.loadMap(dir); err != nil {
		t.Fatal(err)
	}
	// chunks of the surface and of a dungeon are in range
	for _, p := range []Coord{{307, 352, 0}, {0, 0, 0}, {100, 100, 1}} {
		if c := w.chunkAt(p.X, p.Y, p.Z); c <= 0 || c*chunkBytes >= len(w.chunks) {
			t.Errorf("chunk at %v = %d", p, c)
		}
	}
}

// TestPlayHeadless starts the game, walks the avatar with the arrow keys and writes the
// screen to a PNG ($U6_PNG_DIR, else the test's temp dir).
func TestPlayHeadless(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	var g *Game
	started := make(chan error, 1)
	host.StartHeadless(func() {
		var err error
		if g, err = New(dir); err != nil {
			started <- err
			return
		}
		started <- nil
		g.Play()
	})
	for len(started) == 0 {
		if !host.Step() {
			break
		}
	}
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	const settle = 30
	for range settle {
		host.Step()
	}
	removeGargoyles(g)
	start := g.object(int(g.Party[g.Active])).Pos
	keys := []byte{host.ScanRight, host.ScanRight, host.ScanDown, host.ScanLeft, host.ScanRight}
	for _, k := range keys {
		host.KeyQueue = append(host.KeyQueue, host.Key{Scan: k})
		for range settle {
			host.Step()
		}
	}
	end := g.object(int(g.Party[g.Active])).Pos
	t.Logf("avatar moved from %v to %v", start, end)
	if end == start {
		t.Error("the avatar did not move")
	}
	out := t.TempDir()
	if d := os.Getenv("U6_PNG_DIR"); d != "" {
		out = d
	}
	snap := func(name string) {
		frame := g.Frame()
		img := image.NewRGBA(image.Rect(0, 0, g.screenWidth, g.screenHeight))
		level := func(v byte) uint8 { return uint8(int(v) * 255 / 63) }
		for i, c := range frame.Pixels {
			p := frame.Palette[int(c)*3:]
			img.Set(i%g.screenWidth, i/g.screenWidth, color.RGBA{level(p[0]), level(p[1]), level(p[2]), 255})
		}
		file, err := os.Create(filepath.Join(out, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, img); err != nil {
			t.Fatal(err)
		}
	}
	snap("game")
	press := func(keys ...host.Key) {
		for _, k := range keys {
			host.KeyQueue = append(host.KeyQueue, k)
			for range settle {
				host.Step()
			}
		}
	}
	char := func(c byte) host.Key { return host.Key{Char: c} }
	for _, step := range []struct {
		name string
		keys []host.Key
	}{
		{"look", []host.Key{char('l'), {Scan: host.ScanUp}, {Scan: host.ScanEnter, Char: '\r'}}},
		{"drop", []host.Key{char('d'), {Scan: host.ScanEscape}}},
		{"inventory", []host.Key{char('i')}},
	} {
		press(step.keys...)
		snap(step.name)
		if step.name == "look" {
			press(host.Key{Scan: host.ScanEscape})
			testGetAndUse(t, g, press, snap)
		}
	}
	testFight(t, g, press, snap)
}

// testGetAndUse gets gold and lights a candle with the target cursor, first with the
// keys, then with the mouse.
func testGetAndUse(t *testing.T, g *Game, press func(...host.Key), snap func(string)) {
	p := g.object(int(g.Party[g.Active])).Pos
	click := func(dx, dy int) {
		const inTile = 4
		host.MouseX = (p.X+dx-int(g.MapX)+g.viewCenterX)*tileSize + inTile - g.viewPixels().Min.X
		host.MouseY = (p.Y+dy-int(g.MapY)+g.viewCenterY)*tileSize + inTile - g.viewPixels().Min.Y
		for _, down := range []bool{true, false} {
			host.MouseLeft = down
			for range 30 {
				host.Step()
			}
		}
	}
	for _, mouse := range []bool{false, true} {
		suffix := map[bool]string{false: "", true: "_mouse"}[mouse]
		gold := g.AddMapObj(shapeOf(ObjGold, 0), StatusOwned, 5, p.X-1, p.Y, p.Z)
		candle := g.AddMapObj(shapeOf(ObjCandle, 0), 0, 1, p.X, p.Y+1, p.Z)
		g.refreshArea()
		if mouse {
			press(host.Key{Char: 'g'})
			click(-1, 0)
		} else {
			press(host.Key{Char: 'g'}, host.Key{Scan: host.ScanLeft})
		}
		snap("get" + suffix)
		if o := g.object(gold); o.Shape != 0 && o.CoordUse() == OnMap {
			t.Errorf("get%s: the gold is still on the map", suffix)
		}
		if mouse {
			press(host.Key{Char: 'u'})
			click(0, 1)
		} else {
			press(host.Key{Char: 'u'}, host.Key{Scan: host.ScanDown})
		}
		snap("use" + suffix)
		if g.object(candle).Frame() == 0 {
			t.Errorf("use%s: the candle is not lit: %v sel %d candle %d", suffix, *g.object(candle), g.Selection.obj, candle)
		}
		g.DeleteObj(candle)
	}
}

// testFight puts a hostile rat near the party and attacks it until it dies.
func testFight(t *testing.T, g *Game, press func(...host.Key), snap func(string)) {
	p := g.object(int(g.Party[g.Active])).Pos
	rat := g.generate(int(ObjGiantRat), p.X, p.Y-2, p.Z)
	const ratHitPoints = 20 // lasts a few rounds, so the party gets to move
	g.NPCs[rat].HitPoints = ratHitPoints
	g.refreshArea()
	positions := func() []Coord {
		var c []Coord
		for i := range int(g.PartySize) {
			c = append(c, g.object(int(g.Party[i])).Pos)
		}
		return c
	}
	before := positions()
	moved := map[int]bool{}
	if g.InCombat == 0 {
		press(host.Key{Char: 'B'})
	}
	snap("fight_begin")
	for round := 0; round < 40 && g.object(rat).Shape != 0 && g.object(rat).Type() == ObjGiantRat && !g.NPCs[rat].Has(Dead); round++ {
		r := g.object(rat).Pos
		var dir host.Key
		switch {
		case r.X > p.X:
			dir = host.Key{Scan: host.ScanRight}
		case r.X < p.X:
			dir = host.Key{Scan: host.ScanLeft}
		case r.Y > p.Y:
			dir = host.Key{Scan: host.ScanDown}
		default:
			dir = host.Key{Scan: host.ScanUp}
		}
		press(host.Key{Char: 'a'}, dir)
		p = g.object(int(g.Party[g.Active])).Pos
		for i, c := range positions() {
			if c != before[i] {
				moved[i] = true
			}
		}
	}
	snap("fight_end")
	if g.object(rat).Shape != 0 && g.object(rat).Type() == ObjGiantRat && !g.NPCs[rat].Has(Dead) {
		t.Errorf("the rat survived: %v %v", *g.object(rat), g.NPCs[rat].Status)
	}
	t.Logf("%d party members moved during the fight", len(moved))
	if len(moved) == 0 {
		t.Error("no AI moves happened")
	}
}

// removeGargoyles takes away the gargoyles a new game starts with in the throne room
// and the eggs that hatch more, so the tests that play there do not start in a fight.
func removeGargoyles(g *Game) {
	var found []int
	const reach = 32 // the throne room and the halls around it
	mx, my := int(g.MapX), int(g.MapY)
	for n := g.SearchArea(mx-reach, my-reach, mx+reach, my+reach); n >= 0; n = g.NextArea() {
		if t := g.object(n).Type(); t == ObjGargoyle || t == ObjWingedGargoyle || t == ObjEgg {
			found = append(found, n)
		}
	}
	for _, n := range found {
		g.DeleteObj(n)
	}
}
