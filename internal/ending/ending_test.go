package ending

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

func dataDir(t *testing.T) string {
	dir := os.Getenv("U6DATA")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Projects/u6-re/gog/inst")
	}
	if _, err := os.Stat(filepath.Join(dir, shapeFile)); err != nil {
		t.Skip("no game data:", err)
	}
	return dir
}

// savePNG writes the screen ($U6PNG, default the test's temp dir).
func savePNG(t *testing.T, dir, name string, e *ending) {
	f := e.frame()
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	for i, p := range f.Pixels {
		c := f.Palette[int(p)*rgbBytes:]
		img.Set(i%screenWidth, i/screenWidth, color.RGBA{scale(c[0]), scale(c[1]), scale(c[2]), 255})
	}
	out, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		t.Fatal(err)
	}
}

func scale(c byte) uint8 { return uint8(int(c) * 255 / 63) }

func TestSpell(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{{1, "one year"}, {2, "two years"}, {29, "twenty-nine years"}, {42, "42 years"}, {-3, "-3 years"}} {
		if got := spell(c.n, "year"); got != c.want {
			t.Errorf("spell(%d) = %q, want %q", c.n, got, c.want)
		}
	}
	e := &ending{years: spell(1, "year"), days: spell(3, "day")}
	if got, want := e.congratulations().text[len("CONGRATULATIONS\nThou hast completed Ultima VI: The False Prophet, "):],
		"in one year, and three days.\nReport thy feat unto Lord British at Origin Systems!"; got != want {
		t.Errorf("congratulations: %q", got)
	}
}

// TestRun plays the whole ending, pressing space every few seconds.
func TestRun(t *testing.T) {
	t.Setenv("U6GO_INPUT", "test") // no sound
	pngs := os.Getenv("U6PNG")
	if pngs == "" {
		pngs = t.TempDir()
	}
	e := &ending{fsys: os.DirFS(dataDir(t)), args: Args{Mouse: true, Music: true, Years: 1, Months: 2, Days: 3}}
	if err := e.load(); err != nil {
		t.Fatal(err)
	}
	host.ShowFrame = e.frame
	host.StartHeadless(e.main)
	defer func() { activeTimer = nil }()

	const (
		maxTicks  = 20000
		keyEvery  = 300
		shotAfter = 250
	)
	shots, tick := 0, 0
	for ; tick < maxTicks && host.Step(); tick++ {
		switch tick % keyEvery {
		case 0:
			host.PushChar(' ')
		case shotAfter:
			shots++
			savePNG(t, pngs, fmt.Sprintf("%03d_scene%02d", shots, e.scene), e)
		}
	}
	if tick == maxTicks {
		t.Fatalf("still running after %d ticks (scene %d)", maxTicks, e.scene)
	}
	if e.scene != congratulationsScene || e.effectDone != 3 {
		t.Errorf("ended in scene %d, effect %d", e.scene, e.effectDone)
	}
	t.Logf("%d ticks, %d shots in %s", tick, shots, pngs)
}
