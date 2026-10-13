package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/vga"
)

// tholden lives in Lord British's castle; Lord British himself would first ask the copy
// protection question, and Tholden first talks with Iolo.
const tholden = 8

func TestKeywordMatches(t *testing.T) {
	for _, c := range []struct {
		keyword, input string
		want           bool
	}{
		{"name", "what is your NAME", true},
		{"job", "jo", false},
		{"bye", "bye", true},
		{"j?b", "jab", true},
		{"heal", "", false},
	} {
		if got := keywordMatches(c.keyword, c.input, min(len(c.keyword), keywordLength)); got != c.want {
			t.Errorf("keywordMatches(%q, %q) = %v", c.keyword, c.input, got)
		}
	}
}

// scrollText reads the message clip off the canvas, matching each cell with the font.
func scrollText(g *Game) string { return clipText(g, g.clips[clipMessage]) }

// clipText reads the characters drawn in c.
func clipText(g *Game, c clip) string {
	var lines []string
	for row := range c.height {
		var line []byte
		for col := range c.width {
			x0, y0 := (c.x0+col)*cellSize, (c.y0+row)*cellSize
			var cell [cellSize]byte
			for y := range cellSize {
				for x := range cellSize {
					if g.gr.Screen[(y0+y)*vga.ScreenWidth+x0+x] != c.bg {
						cell[y] |= 1 << (cellSize - 1 - x)
					}
				}
			}
			ch := byte('?')
			for i := ' '; i < '~'; i++ {
				if g.gr.Font[i] == cell {
					ch = byte(i)
					break
				}
			}
			line = append(line, ch)
		}
		lines = append(lines, strings.TrimRight(string(line), " "))
	}
	return strings.Join(lines, "\n")
}

// TestTalkToTholden puts Tholden next to the avatar of a fresh game and asks his name and job, then says bye; it writes the screen to talk.png ($U6_PNG_DIR).
func TestTalkToTholden(t *testing.T) {
	// the host runs one game per process, and other tests start theirs
	const child = "U6_TALK_TEST_CHILD"
	if os.Getenv(child) == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTalkToTholden$", "-test.v")
		cmd.Env = append(os.Environ(), child+"=1")
		out, err := cmd.CombinedOutput()
		t.Logf("%s", out)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
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
		g.SpellFx[spellTimeStop] = 1 // the same first view each run: nothing fidgets
		started <- nil
		g.Play()
	})
	for len(started) == 0 && host.Step() {
	}
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	const settle = 30
	// the first view drawn
	for g.gr.BackBuffer == [len(g.gr.BackBuffer)]byte{} && host.Step() {
	}
	compareClassicView(t, g)
	g.SpellFx[spellTimeStop] = 0
	steps := func(n int) {
		for range n {
			host.Step()
		}
	}
	steps(settle)
	compareClassicTiles(t, g)
	me := g.object(int(g.Party[g.Active])).Pos
	g.MoveObj(tholden, me.X+1, me.Y, me.Z)
	g.NPCs[tholden].Mode = AIMotionless
	steps(settle)
	// the overlays for docs/ui
	writePNG(t, g, "world.png")
	g.dockOffset = iconsPanel.Dy()
	writePNG(t, g, "dock.png")
	g.dockOffset = 0
	g.showLog = true
	writePNG(t, g, "log.png")
	g.showLog, g.showDate = false, true
	writePNG(t, g, "date.png")
	g.showDate = false

	var transcript strings.Builder
	// wait answers the paging prompts until the script asks for input
	wait := func() {
		for range 50 {
			steps(settle)
			if g.promptChar == morePrompt && len(host.KeyQueue) == 0 {
				transcript.WriteString(scrollText(g) + "\n")
				host.PushChar(' ')
				continue
			}
			if !g.InConversation || len(g.talkOptions) > 0 {
				break
			}
		}
		transcript.WriteString(scrollText(g) + "\n")
	}
	say := func(words string) {
		for i := range len(words) {
			host.PushChar(words[i])
			steps(2)
		}
		host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanEnter, Char: '\r'})
		wait()
	}

	// Talk, aim right, select
	host.PushChar('t')
	steps(settle)
	host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanRight})
	steps(settle)
	host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanEnter, Char: '\r'})
	wait()
	if !g.InConversation {
		t.Fatalf("no conversation:\n%s", transcript.String())
	}
	say("name")
	say("job")
	writePNG(t, g, "talk.png")
	writePNG(t, g, "conversation.png")
	say("bye")
	// the farewell stays on the speech scroll after the log clip comes back
	transcript.WriteString(clipText(g, clip{x0: speechX0, y0: speechY0, width: speechColumns, height: speechRows, bg: g.clips[clipMessage].bg}))
	text := transcript.String()
	t.Logf("transcript:\n%s", text)
	if g.InConversation {
		t.Error("still talking after bye")
	}
	for _, want := range []string{"You see a", "I am Tholden von", "paperwork", "Tholden bows"} {
		if !strings.Contains(text, want) {
			t.Errorf("transcript lacks %q", want)
		}
	}
	if g.NPCs[tholden].TalkFlags&talkFlagMet == 0 {
		t.Error("Tholden is not met after asking his name")
	}
}

func writePNG(t *testing.T, g *Game, name string) {
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
	file, err := os.Create(filepath.Join(out, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

// compareClassicView checks that the middle 11x11 tiles of the view are drawn as the
// original drew its whole view: testdata/classic_view.bin is the main branch's back buffer
// (before the frame went over it) when this test's game first drew it. Its ring of tiles
// is left out: there the original's sight stopped at its edge, so walls missed the sides
// seen beyond it (compareClassicTiles checks those tiles from cameras that see around them).
func compareClassicView(t *testing.T, g *Game) {
	want, err := os.ReadFile(filepath.Join("testdata", "classic_view.bin"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		classicSize = classicViewTiles * tileSize
		offsetX     = (viewCenterX - classicViewCenter) * tileSize
		offsetY     = (viewCenterY - classicViewCenter) * tileSize
	)
	bad := 0
	for y := tileSize; y < classicSize-tileSize; y++ {
		for x := tileSize; x < classicSize-tileSize; x++ {
			if g.gr.BackBuffer[(y+offsetY)*vga.BackBufferWidth+x+offsetX] != want[y*classicSize+x] {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Errorf("%d pixels of the classic 11x11 view differ", bad)
	}
}

// compareClassicTiles checks the tiles all over the view, walls joined to the sides one
// sees included, against the original's 11x11 view with the camera moved so that each
// tile is well inside it: testdata/classic_tiles.json holds the main branch's views for
// cameras moved by "dx,dy" from this test's place. Tiles hidden from either camera are
// skipped: what one sees changes with where one looks from.
func compareClassicTiles(t *testing.T, g *Game) {
	raw, err := os.ReadFile(filepath.Join("testdata", "classic_tiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var views map[string][classicViewTiles][classicViewTiles]int
	if err := json.Unmarshal(raw, &views); err != nil {
		t.Fatal(err)
	}
	const inside = classicViewCenter - 1 // the ring of the 11x11 view is left out
	compared := 0
	for key, classic := range views {
		var dx, dy int
		fmt.Sscanf(key, "%d,%d", &dx, &dy)
		for y := -inside; y <= inside; y++ {
			for x := -inside; x <= inside; x++ {
				vx, vy := viewCenterX+dx+x, viewCenterY+dy+y
				want, got := classic[classicViewCenter+y][classicViewCenter+x], g.tileView[vy][vx]
				if want == tileHidden || got == tileHidden {
					continue
				}
				compared++
				if want != got {
					t.Errorf("tile (%d,%d) of the view is %d, the original's view moved by %s shows %d", vx, vy, got, key, want)
				}
			}
		}
	}
	if compared < classicViewTiles*classicViewTiles {
		t.Error("no tile compared")
	}
}
