package game

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
	"github.com/memmaker/u6-go/internal/sound"
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
	// with the FM Towns CD's files (u6 -import-cd) in $U6_TOWNS_DATA, Tholden speaks
	towns := os.Getenv("U6_TOWNS_DATA")
	if towns != "" {
		if err := os.CopyFS(filepath.Join(dir, "CD"), os.DirFS(filepath.Join(towns, "CD"))); err != nil {
			t.Fatal(err)
		}
		t.Setenv("U6GO_INPUT", "silent") // no audio device: the voices stay queued
		sound.Start(os.DirFS(dir))
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
	writeUIPNGs(t, g, steps, settle)

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
	if towns != "" && !sound.Speaking() {
		t.Error("Tholden's name has no voice")
	}
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
	if strings.Contains(text, "~") || strings.Contains(text, "+") {
		t.Error("the transcript shows FM Towns marks")
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
	img := image.NewRGBA(image.Rect(0, 0, g.screenWidth, g.screenHeight))
	level := func(v byte) uint8 { return uint8(int(v) * 255 / 63) }
	for i, c := range frame.Pixels {
		p := frame.Palette[int(c)*3:]
		shade := 255
		if frame.Shade != nil {
			shade = int(frame.Shade[i])
		}
		dim := func(v byte) uint8 { return uint8(int(level(v)) * shade / 255) }
		img.Set(i%g.screenWidth, i/g.screenWidth, color.RGBA{dim(p[0]), dim(p[1]), dim(p[2]), 255})
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

// compareClassicView checks tiles of the middle 11x11 against the original's screen from
// the same save: testdata/original_rows.bin is GAME.EXE's view in DOSBox-X, the inner
// columns 1..9 of rows 8 and 9, which hold the carpet, the steps and the windows but no one
// who moves.
func compareClassicView(t *testing.T, g *Game) {
	want, err := os.ReadFile(filepath.Join("testdata", "original_rows.bin"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		firstColumn = 1
		firstRow    = 8
		width       = (classicViewTiles - 2*firstColumn) * tileSize
	)
	offsetX := (g.viewCenterX - classicViewCenter + firstColumn) * tileSize
	offsetY := (g.viewCenterY - classicViewCenter + firstRow) * tileSize
	bad := 0
	for y := range len(want) / width {
		for x := range width {
			if g.gr.BackBuffer[(y+offsetY)*vga.BackBufferWidth+x+offsetX] != want[y*width+x] {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Errorf("%d pixels of the original's view differ", bad)
	}
}

// compareClassicTiles checks the middle 11x11 tiles of the view, the hidden ones and the
// walls cut to the sides one sees included, against the original's view from the same
// place: testdata/classic_tiles.json is the main branch's.
func compareClassicTiles(t *testing.T, g *Game) {
	raw, err := os.ReadFile(filepath.Join("testdata", "classic_tiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var classic [classicViewTiles][classicViewTiles]int
	if err := json.Unmarshal(raw, &classic); err != nil {
		t.Fatal(err)
	}
	for y := range classicViewTiles {
		for x := range classicViewTiles {
			vx, vy := g.viewCenterX-classicViewCenter+x, g.viewCenterY-classicViewCenter+y
			if want, got := classic[y][x], g.tileView[vy][vx]; want != got {
				t.Errorf("tile (%d,%d) of the view is %d, the original shows %d", vx, vy, got, want)
			}
		}
	}
}

// writeUIPNGs drives the port's overlays and writes them for docs/ui: the help, the Enter
// menu, a right click's context menu on Tholden, the toasts of a Look and a dock tooltip.
func writeUIPNGs(t *testing.T, g *Game, steps func(int), settle int) {
	enter := host.Key{Scan: host.ScanEnter, Char: keyEnter}
	host.PushChar('?')
	steps(settle)
	writePNG(t, g, "help.png")
	host.PushChar(' ')
	steps(settle)
	host.KeyQueue = append(host.KeyQueue, enter)
	steps(settle)
	writePNG(t, g, "menu.png")
	host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanEscape, Char: keyEscape})
	steps(settle)
	if g.popup != nil {
		t.Fatal("Esc left the menu open")
	}
	// right click Tholden, right of the party
	centreX, centreY := g.mapCentre()
	host.MouseX, host.MouseY = centreX+tileSize-g.viewPixels().Min.X, centreY-g.viewPixels().Min.Y
	host.MouseRight = true
	steps(settle)
	host.MouseRight = false
	steps(settle)
	if g.popup == nil {
		t.Fatal("no context menu on Tholden")
	}
	writePNG(t, g, "context.png")
	host.KeyQueue = append(host.KeyQueue, enter) // Look
	steps(settle)
	host.PushChar(' ') // past the portrait
	steps(settle)
	writePNG(t, g, "toasts.png")
	if len(g.toasts) == 0 {
		t.Error("Look left no toast")
	}
	g.toasts = nil
	// over the dock's Talk icon
	talk := cmdTalk - cmdAttack
	host.MouseX = (g.screenWidth-iconsPanel.Dx())/2 + iconsX - iconsPanel.Min.X + talk*tileSize + tileSize/2
	host.MouseY = g.screenHeight - 1
	steps(settle)
	writePNG(t, g, "tooltip.png")
	if g.dockTooltip() != "Talk" {
		t.Errorf("tooltip %q, want Talk", g.dockTooltip())
	}
	host.MouseX, host.MouseY = g.screenWidth/2, g.screenHeight/2
	steps(settle)
}
