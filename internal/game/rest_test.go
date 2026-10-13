package game

import (
	"testing"

	"github.com/memmaker/u6-go/internal/host"
)

// TestRest camps alone in the wilderness for eight hours: the time passes, the Avatar heals and
// gets all the magic back.
func TestRest(t *testing.T) {
	inGame(t, func(g *Game) {
		g.PartySize = 1
		avatar := int(g.Party[0])
		found := false
		const surfaceSize, step = 1024, 16
		for y := step; y < surfaceSize && !found; y += step {
			for x := step; x < surfaceSize && !found; x += step {
				g.partyTeleport(x, y, 0, 0)
				found = !g.notWilderness() && !g.foesNear() && !g.othersNear() && !g.noRoom()
			}
		}
		if !found {
			t.Error("no place to rest")
			return
		}
		if g.InCombat != 0 {
			g.breakOff()
		}
		npc := &g.NPCs[avatar]
		hour := g.Hour
		// a night ambush wakes the camp before it heals: chase the ambushers off and rest again
		for range 10 {
			npc.HitPoints, npc.Magic = 1, 0
			g.NextSleep = 0
			g.giveObj(avatar, ObjPortionOfMeat, 1)
			host.PushChar('8')
			g.rest()
			if npc.HitPoints > 1 {
				break
			}
			for n := firstMonster; n < npcCount; n++ {
				if g.object(n).Shape != 0 && g.NPCs[n].Status&alignmentMask == alignEvil {
					g.DeleteObj(n)
				}
			}
			g.InCombat = 0
			g.breakOff()
			for i := range npcCount {
				g.NPCs[i].Status &^= Asleep
			}
		}
		if g.Hour == hour {
			t.Errorf("no time passed:\n%s", scrollText(g))
		}
		if npc.HitPoints <= 1 || int(npc.Magic) != g.maxMagic(avatar) {
			t.Errorf("hit points %d, magic %d after rest:\n%s", npc.HitPoints, npc.Magic, scrollText(g))
		}
		t.Logf("%s", scrollText(g))
	})
}

// TestRestoreGame throws away a change and gets the saved game back.
func TestRestoreGame(t *testing.T) {
	inGame(t, func(g *Game) {
		avatar := int(g.Party[0])
		saved := g.NPCs[avatar].HitPoints
		g.NPCs[avatar].HitPoints = saved + 1
		g.restoreGame()
		if g.NPCs[avatar].HitPoints != saved {
			t.Errorf("hit points %d after restore, saved %d", g.NPCs[avatar].HitPoints, saved)
		}
	})
}
