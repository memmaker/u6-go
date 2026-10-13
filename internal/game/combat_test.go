package game

import (
	"os"
	"os/exec"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// inGame runs body on the game goroutine of a fresh game, in a child process because the
// headless host is global.
func inGame(t *testing.T, body func(g *Game)) {
	const child = "U6_GAME_TEST_CHILD"
	if os.Getenv(child) != t.Name() {
		cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
		cmd.Env = append(os.Environ(), child+"="+t.Name())
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
	done := make(chan error, 1)
	host.StartHeadless(func() {
		g, err := New(dir)
		if err == nil {
			body(g)
		}
		done <- err
	})
	for len(done) == 0 && host.Step() {
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// TestNPCCastsInCombat has a monster cast Harm from a charge and fail to cast from a
// spellbook without the magic points for it.
func TestNPCCastsInCombat(t *testing.T) {
	inGame(t, func(g *Game) {
		avatar := int(g.Party[g.Active])
		p := g.object(avatar).Pos
		monster := g.AddMonster(ObjGargoyle, p.X+1, p.Y, p.Z)
		if monster < 0 {
			t.Error("no room for a monster")
			return
		}
		g.NPCs[monster].Intelligence = 30
		g.NPCs[avatar].Intelligence = 1
		g.NPCs[avatar].HitPoints = 30

		book := g.AddInvObj(shapeOf(ObjSpellbook, 0), StatusLocal|Equipped, uint16(qualQuan(castHarm, 0)), monster)
		g.NPCs[monster].Magic = 0
		g.castInCombat(monster, avatar, book)
		if g.NPCs[avatar].HitPoints != 30 {
			t.Errorf("Harm cast from a spellbook without magic points")
		}

		charge := g.AddInvObj(shapeOf(ObjCharge, 0), StatusLocal|Equipped, uint16(qualQuan(castHarm, 0)), monster)
		active := g.Active
		for range 20 {
			g.castInCombat(monster, avatar, charge)
			if g.NPCs[avatar].HitPoints < 30 {
				break
			}
		}
		if g.NPCs[avatar].HitPoints >= 30 {
			t.Errorf("Harm never hurt the Avatar")
		}
		if g.Active != active || !g.TextOn {
			t.Errorf("Active %d, TextOn %v not restored", g.Active, g.TextOn)
		}
	})
}
