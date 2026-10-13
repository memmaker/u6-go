package game

import (
	"image"
	"testing"
)

// TestFormations walks the party a few steps in each formation and checks that every
// follower stands on its place.
func TestFormations(t *testing.T) {
	inGame(t, func(g *Game) {
		removeGargoyles(g)
		if g.InCombat != 0 {
			g.breakOff()
		}
		const steps = 5
		const south, north = 4, 0
		for i, f := range []formation{formationDefault, formationFollow, formationTriangle} {
			g.formation = f
			dir := south
			if i%2 == 1 {
				dir = north
			}
			leader := int(g.Party[g.Active])
			for range steps {
				g.advance(dir)
			}
			lp := g.object(leader).Pos
			back := image.Pt(-dirStepX[dir], -dirStepY[dir])
			place := 1
			for k := range int(g.PartySize) {
				member := int(g.Party[k])
				if member == leader {
					continue
				}
				p := g.object(member).Pos
				got := image.Pt(p.X-lp.X, p.Y-lp.Y)
				var want image.Point
				switch f {
				case formationFollow:
					want = back.Mul(place)
				case formationTriangle:
					// south and north: sideways is along x
					want = back.Mul(triangleBack[place]).Add(image.Pt(-dirStepY[dir]*triangleSide[place], 0))
				default:
					want = back.Mul(followBack[place]).Add(image.Pt(-dirStepY[dir]*followSide[place], 0))
				}
				t.Logf("%s place %d at %v, want %v", formationNames[f], place, got, want)
				if got != want {
					t.Errorf("%s: follower %d stands at %v from the leader, want %v", formationNames[f], place, got, want)
				}
				place++
			}
		}
	})
}
