package intro

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
)

// dataCopy copies the game files ($U6DATA, default ~/Projects/u6-re/gog/inst) to a temp
// dir, so the test never writes to the originals.
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
		if e.IsDir() {
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

// savePNG writes the frame the host would show ($U6PNG, default the test's temp dir).
func savePNG(t *testing.T, dir, name string, u *intro) {
	f := u.frame()
	_ = os.WriteFile(filepath.Join(dir, name+".raw"), f.Pixels, 0o644)
	_ = os.WriteFile(filepath.Join(dir, name+".pal"), f.Palette[:], 0o644)
	img := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	for i, p := range f.Pixels {
		c := f.Palette[int(p)*3:]
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

// scale makes a six-bit DAC component eight bits.
func scale(c byte) uint8 { return uint8(int(c) * 255 / 63) }

// TestRun plays the boot intro, the introduction and a character creation with keys and
// checks the saved avatar.
func TestRun(t *testing.T) {
	t.Setenv("U6GO_INPUT", "test") // no sound
	dir := dataCopy(t)
	pngs := os.Getenv("U6PNG")
	if pngs == "" {
		pngs = t.TempDir()
	}
	u := &intro{fsys: os.DirFS(dir), saveDir: dir}
	if err := u.load(); err != nil {
		t.Fatal(err)
	}
	var result Result
	var runErr error
	host.ShowFrame = u.frame
	host.StartHeadless(func() { result, runErr = u.main() })

	const (
		phaseBoot = iota
		phaseTitle
		phaseCreate
		phaseGypsy
	)
	phase, since, menus, shots, wasInMenu := phaseBoot, 0, 0, 0, false
	push := func(c byte) { host.PushChar(c) }
	inMenu := func() bool { return u.menuDrawn && u.curMusic == sound.SongUltima }
	shot := func(name string) { shots++; savePNG(t, pngs, fmt.Sprintf("%03d_%s", shots, name), u) }
	const maxTicks = 60000
	tick := 0
	for ; tick < maxTicks && host.Step(); tick++ {
		since++
		if inMenu() != wasInMenu {
			wasInMenu, since = !wasInMenu, 0
		}
		if wasInMenu {
			if since == 30 {
				shot("menu")
				menus++
				switch menus {
				case 1:
					push('i')
					phase = phaseTitle
				default:
					push('c')
					phase = phaseCreate
				}
			}
			continue
		}
		switch phase {
		case phaseBoot, phaseTitle:
			if since%150 == 0 {
				shot([]string{"boot", "title"}[phase])
			}
			if since%60 == 0 {
				push(' ')
			}
		case phaseCreate:
			switch since {
			case 60:
				shot("name")
				for _, c := range []byte("Felix\r") {
					push(c)
				}
			case 120:
				push('f')
			case 180:
				shot("portrait")
				push('p')
			case 240:
				shot("portrait2")
				push('c')
				phase, since = phaseGypsy, 0
			}
		case phaseGypsy:
			if since%120 == 0 {
				shot("gypsy")
			}
			if since%50 == 0 {
				push('a')
			}
		}
	}
	if tick == maxTicks {
		shot("timeout")
		t.Fatalf("still running after %d ticks", maxTicks)
	}
	if runErr != nil || result != StartGame {
		t.Fatalf("Run = %v, %v; want StartGame", result, runErr)
	}
	b, err := os.ReadFile(filepath.Join(dir, saveGameDir, objlistName))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != objlistSize {
		t.Fatalf("OBJLIST is %d bytes", len(b))
	}
	if got := cString(b[nameAt : nameAt+nameBytes]); got != "Felix" {
		t.Errorf("name %q", got)
	}
	if b[femaleAt] != 1 || b[portraitAt] != 2 {
		t.Errorf("female %d portrait %d, want 1 and 2 (female portrait 1)", b[femaleAt], b[portraitAt])
	}
	str, dex, intl := int(b[strengthAt]), int(b[dexterityAt]), int(b[intelligenceAt])
	if min(str, dex, intl) < startAttribute || str+dex+intl > 3*startAttribute+7*3 {
		t.Errorf("attributes %d %d %d", str, dex, intl)
	}
	if int(b[magicAt]) != intl*2 || b[levelAt] != startLevel || b[hitPointsAt] != startHitPoints {
		t.Errorf("magic %d level %d hit points %d", b[magicAt], b[levelAt], b[hitPointsAt])
	}
	if exp := int(b[experienceAt]) | int(b[experienceAt+1])<<8; exp != startExperience {
		t.Errorf("experience %d", exp)
	}
	t.Logf("%d ticks; str %d dex %d int %d karma %d", tick, str, dex, intl, b[karmaAt])
}
