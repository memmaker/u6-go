package game

import (
	"image"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// TestClickPartyIcons clicks each member's icon in the party column: the game opens the
// member's inventory and comes back to the prompt, even with a stale container.
func TestClickPartyIcons(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	steps := func(n int) {
		for range n {
			host.Step()
			g.Frame() // as the browser draws each tick
		}
	}
	steps(settleTicks)
	// a bag opened in an inventory and since put on the map: the status area's last
	// container is stale until the equipment view draws again
	p := g.object(int(g.Party[0])).Pos
	bag := g.AddMapObj(shapeOf(ObjBag, 0), 0, 0, p.X+1, p.Y, p.Z)
	defer host.Fit(host.ScreenWidth, host.ScreenHeight)
	for _, size := range []image.Point{{1280, 960}, {2560, 1080}, {1080, 1920}, {1920, 1080}, {800, 600}} {
		host.Fit(size.X, size.Y)
		steps(settleTicks)
		for k := range int(g.PartySize) {
			g.StatusDisplay = cmdStatusParty
			g.StatusDirty = true
			steps(settleTicks)
			g.container = bag
			host.MouseX = g.screenWidth - margin - tileSize/2
			host.MouseY = margin + k*partyRowHeight + tileSize/2
			host.MouseLeft = true
			steps(settleTicks)
			host.MouseLeft = false
			steps(settleTicks)
			g.Frame()
			if !g.atMainPrompt || g.StatusDisplay != cmdStatusEquipment || g.statusMember != k {
				t.Fatalf("%v member %d: prompt %v, display %d, member %d", size, k, g.atMainPrompt, g.StatusDisplay, g.statusMember)
			}
		}
	}
}

// TestHoldStatusButton holds the button down on the equipment view's next member arrow:
// the held click repeats the command once a tick, and the game keeps running.
func TestHoldStatusButton(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	steps := func(n int) {
		for range n {
			host.Step()
			g.Frame()
		}
	}
	steps(settleTicks)
	g.StatusDisplay, g.statusMember, g.StatusDirty = cmdStatusEquipment, 0, true
	steps(settleTicks)
	gp := g.statusGump()
	at := image.Pt(g.screenWidth-margin-gp.size.X, margin).Add(nextArrow).Add(image.Pt(2, 2))
	host.MouseX, host.MouseY = at.X, at.Y
	host.MouseLeft = true
	steps(settleTicks)
	host.MouseLeft = false
	steps(settleTicks)
	if !g.atMainPrompt || g.statusMember == 0 {
		t.Fatalf("prompt %v, member %d", g.atMainPrompt, g.statusMember)
	}
}

// TestHoldWhileAsked holds the button down on the map while the game asks yes or no.
func TestHoldWhileAsked(t *testing.T) {
	dir := dataCopy(t)
	if err := WriteCharacter(dir, DefaultCharacter()); err != nil {
		t.Fatal(err)
	}
	g, err := startHeadless(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	steps := func(n int) {
		for range n {
			host.Step()
			g.Frame()
		}
	}
	steps(settleTicks)
	host.KeyQueue = append(host.KeyQueue, host.Key{Char: keyCtrlQ})
	steps(settleTicks)
	for _, at := range []image.Point{{g.screenWidth / 2, g.screenHeight / 2}, {g.screenWidth/2 + 3*tileSize, g.screenHeight / 2}, {g.screenWidth - 20, 20}} {
		host.MouseX, host.MouseY = at.X, at.Y
		for _, b := range []*bool{&host.MouseLeft, &host.MouseRight} {
			*b = true
			steps(settleTicks)
			*b = false
			steps(settleTicks)
		}
	}
	host.KeyQueue = append(host.KeyQueue, host.Key{Char: 'n'})
	steps(settleTicks)
	if !g.atMainPrompt {
		t.Fatal("not back at the prompt")
	}
}

// TestMissileAngleFar: a flight beyond the angle table's reach takes its direction's angle.
func TestMissileAngleFar(t *testing.T) {
	if missileAngle(-20, 0) != missileAngle(-missileReach, 0) || missileAngle(9, 18) != missileAngle(3, 7) {
		t.Error("far flights do not keep their direction")
	}
}
