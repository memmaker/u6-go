package game

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// TestFuzzUI plays random keys, clicks, drags and screen sizes for a while, a frame drawn
// every tick, to find panics and hangs. It runs with U6_FUZZ_SEED set.
func TestFuzzUI(t *testing.T) {
	seed, err := strconv.Atoi(os.Getenv("U6_FUZZ_SEED"))
	if err != nil {
		t.Skip("U6_FUZZ_SEED not set")
	}
	r := rand.New(rand.NewSource(int64(seed)))
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err2 := startHeadless(t, dir)
	if err2 != nil {
		t.Fatal(err2)
	}
	defer host.Fit(host.ScreenWidth, host.ScreenHeight)
	step := func(n int) {
		for range n {
			if !host.Step() {
				t.Fatal("ended")
			}
			g.Frame()
		}
	}
	step(settleTicks)
	keys := []host.Key{{Char: 'i'}, {Char: 'I'}, {Scan: host.ScanEscape, Char: keyEscape}, {Scan: host.ScanEnter, Char: '\r'}, {Scan: host.ScanTab, Char: '\t'}, {Char: '*'}, {Char: '+'}, {Char: '-'}, {Char: '/'}, {Char: ' '}, {Char: '1'}, {Char: '2'}, {Char: '0'}}
	for range 3000 {
		switch r.Intn(6) {
		case 0:
			host.ShiftDown = r.Intn(2) == 0
			host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanF1 + byte(r.Intn(10))})
		case 1:
			host.ShiftDown = false
			host.KeyQueue = append(host.KeyQueue, keys[r.Intn(len(keys))])
		case 2, 3:
			host.MouseX, host.MouseY = r.Intn(g.screenWidth), r.Intn(g.screenHeight)
			if r.Intn(2) == 0 {
				host.MouseLeft = true
			} else {
				host.MouseRight = true
			}
			step(r.Intn(3) + 1)
			host.MouseX += r.Intn(40) - 20
			step(r.Intn(3))
			host.MouseLeft, host.MouseRight = false, false
		case 4:
			if r.Intn(20) == 0 {
				sizes := [][2]int{{1280, 960}, {2560, 1080}, {1080, 1920}, {1920, 1080}}
				s := sizes[r.Intn(len(sizes))]
				host.Fit(s[0], s[1])
			}
		case 5:
			if r.Intn(10) == 0 {
				g.PortraitPlusOne = byte(r.Intn(20))
			}
		}
		step(r.Intn(15) + 1)
	}
}
