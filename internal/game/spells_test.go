package game

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// TestCastSpells casts Light, Create Food and Heal from a fresh game with the Cast command
// and the words of power.
func TestCastSpells(t *testing.T) {
	const child = "U6_SPELL_TEST_CHILD"
	if os.Getenv(child) == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCastSpells$", "-test.v")
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
		started <- nil
		g.Play()
	})
	for len(started) == 0 && host.Step() {
	}
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	steps := func(n int) {
		for range n {
			host.Step()
		}
	}
	const settle = 30
	steps(settle)
	removeGargoyles(g)

	avatar := int(g.Party[g.Active])
	book := g.AddInvObj(shapeOf(ObjSpellbook, 0), StatusLocal|Equipped, 0, avatar)
	for _, s := range []int{castCreateFood, castHeal, castLight} {
		g.AddInvObj(shapeOf(ObjSpell1, 0), StatusLocal|InInventory, uint16(qualQuan(s, 1)), book)
	}
	for _, r := range reagentTypes {
		g.giveObj(avatar, r, 10)
	}
	npc := &g.NPCs[avatar]
	npc.Level, npc.Magic, npc.HitPoints = 8, 60, 5
	enter := func() {
		host.KeyQueue = append(host.KeyQueue, host.Key{Scan: host.ScanEnter, Char: '\r'})
		steps(settle)
	}
	cast := func(spell int, aim bool) {
		host.PushChar('c')
		steps(settle)
		for _, c := range spellCodes[spell] {
			host.PushChar(byte(c))
			steps(settle)
		}
		enter()
		if aim {
			enter()
		}
		// wait for the prompt of the next command
		for range 200 {
			steps(settle)
			if g.promptChar == morePrompt {
				host.PushChar(' ')
				continue
			}
			lines := strings.Split(strings.TrimRight(scrollText(g), " \n"), "\n")
			if strings.HasPrefix(lines[len(lines)-1], ">") && len(strings.TrimSpace(lines[len(lines)-1])) <= 2 {
				break
			}
		}
	}

	cast(castLight, false)
	if g.SpellFx[spellLight] == 0 {
		t.Errorf("Light did not light:\n%s", scrollText(g))
	}
	meat := func() int { return g.reagentCount(avatar, ObjPortionOfMeat, 0) }
	before := meat()
	cast(castCreateFood, false)
	if meat() <= before {
		t.Errorf("Create Food made no food:\n%s", scrollText(g))
	}
	cast(castHeal, true)
	if npc.HitPoints <= 5 {
		t.Errorf("Heal did not heal (%d hit points):\n%s", npc.HitPoints, scrollText(g))
	}
	if !strings.Contains(scrollText(g), "Heal") {
		t.Errorf("the spell's name is not printed:\n%s", scrollText(g))
	}

	// the spellbook page, as the Cast command shows it
	host.PushChar('c')
	steps(settle)
	writePNG(t, g, "spellbook.png")
	host.PushChar(' ')
	steps(settle)
	t.Logf("%s", scrollText(g))
}
