package game

import (
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
// (before the frame went over it) when this test's game first drew it. The walls on the
// original's right edge differ: they now join the walls beyond it, which it never looked at.
func compareClassicView(t *testing.T, g *Game) {
	want, err := os.ReadFile(filepath.Join("testdata", "classic_view.bin"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		classicSize = classicViewTiles * tileSize
		offsetX     = (viewCenterX - classicViewCenter) * tileSize
		offsetY     = (viewCenterY - classicViewCenter) * tileSize
		edge        = classicViewTiles - 1
		joinRow     = 3 // the wall that meets the room's east wall
	)
	wallJoins := map[image.Point]bool{{edge, joinRow}: true, {edge, edge}: true}
	bad := 0
	for y := range classicSize {
		for x := range classicSize {
			if wallJoins[image.Pt(x/tileSize, y/tileSize)] {
				continue
			}
			if g.gr.BackBuffer[(y+offsetY)*vga.BackBufferWidth+x+offsetX] != want[y*classicSize+x] {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Errorf("%d pixels of the classic 11x11 view differ", bad)
	}
}
