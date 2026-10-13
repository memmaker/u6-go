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
	frame := g.Frame()
	img := image.NewRGBA(image.Rect(0, 0, host.ScreenWidth, host.ScreenHeight))
	level := func(v byte) uint8 { return uint8(int(v) * 255 / 63) }
	for i, c := range frame.Pixels {
		p := frame.Palette[int(c)*3:]
		img.Set(i%host.ScreenWidth, i/host.ScreenWidth, color.RGBA{level(p[0]), level(p[1]), level(p[2]), 255})
	}
	out := t.TempDir()
	if d := os.Getenv("U6_PNG_DIR"); d != "" {
		out = d
	}
	file, err := os.Create(filepath.Join(out, "game.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}
