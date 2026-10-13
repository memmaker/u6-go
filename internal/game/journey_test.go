package game

import (
	"errors"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// startHeadless runs New(dir) and Play on a headless host until New has returned.
func startHeadless(t *testing.T, dir string) (*Game, error) {
	t.Helper()
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
	for len(started) == 0 && host.Step() {
	}
	return g, <-started
}

// settleTicks is how many host ticks a key is given to take effect.
const settleTicks = 30

func pressKeys(keys ...host.Key) {
	for _, k := range keys {
		host.KeyQueue = append(host.KeyQueue, k)
		for range settleTicks {
			host.Step()
		}
	}
}

// gargoylesNear counts the throne room's gargoyles.
func gargoylesNear(g *Game) int {
	n := 0
	const reach = 32
	mx, my := int(g.MapX), int(g.MapY)
	for o := g.SearchArea(mx-reach, my-reach, mx+reach, my+reach); o >= 0; o = g.NextArea() {
		if t := g.object(o).Type(); t == ObjGargoyle || t == ObjWingedGargoyle {
			n++
		}
	}
	return n
}

// TestJourneyOnwardWithoutCharacter: with no character the game says so, waits for a key
// and goes back to the intro, as GAME.EXE ran ULTIMA6.EXE again.
func TestJourneyOnwardWithoutCharacter(t *testing.T) {
	dir := dataCopy(t)
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	var err error
	done := make(chan struct{})
	host.StartHeadless(func() {
		_, err = New(dir)
		close(done)
	})
	for range settleTicks {
		host.Step()
	}
	select {
	case <-done:
		t.Fatal("the game did not wait for a key")
	default:
	}
	pressKeys(host.Key{Char: ' '})
	<-done
	if !errors.Is(err, ErrNoCharacter) {
		t.Errorf("err = %v, want ErrNoCharacter", err)
	}
}

// TestJourneyOnward plays a new game, saves it, and starts again as Journey Onward does:
// the avatar's position, the party and the changed objects come back.
func TestJourneyOnward(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if gargoylesNear(g) == 0 {
		t.Fatal("a new game has no gargoyles in the throne room")
	}
	removeGargoyles(g)
	pressKeys(host.Key{Scan: host.ScanRight}, host.Key{Scan: host.ScanRight}, host.Key{Scan: host.ScanDown})
	before := gargoylesNear(g)
	pos := g.object(int(g.Party[g.Active])).Pos
	party := g.PartySize
	pressKeys(host.Key{Char: keyCtrlS}, host.Key{Char: 'y'})

	g, err = startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := g.object(int(g.Party[g.Active])).Pos; got != pos {
		t.Errorf("avatar at %v, saved at %v", got, pos)
	}
	if g.PartySize != party {
		t.Errorf("party of %d, saved %d", g.PartySize, party)
	}
	if n := gargoylesNear(g); n != before {
		t.Errorf("%d gargoyles after loading, %d saved", n, before)
	}
}
