package game

import "image"

// Party formations, the port's: how the party lines up behind its leader. Ctrl+F (or the
// Enter menu) opens a scroll to pick one. The choice lasts for the session; the save keeps
// the original's layout, which has no room for it.

type formation int

const (
	// formationDefault is the original's wedge (followSide, followBack)
	formationDefault formation = iota
	// formationFollow is a single line, each member stepping where the one in front was
	formationFollow
	// formationTriangle puts the leader in front and widening rows behind
	formationTriangle
	formationCount
)

var formationNames = [formationCount]string{"Default", "Follow", "Triangle"}

// keyCtrlF opens the formation scroll; U6 does not use it.
const keyCtrlF = 6

// the triangle's places behind the leader: the second member right behind, the next two
// beside it, then rows of five, sideways and back as followSide and followBack
var (
	triangleSide = [11]int{0, 0, -1, 1, 0, -1, 1, -2, 2, 0, -1}
	triangleBack = [11]int{0, 1, 1, 1, 2, 2, 2, 2, 2, 3, 3}
)

// formationPlace is where follower place should stand behind the leader at (x, y) facing
// facing; front is where the member ahead of it stood before this move.
func (g *Game) formationPlace(place, x, y, facing int, front image.Point) (int, int) {
	side, back := followSide[:], followBack[:]
	switch g.formation {
	case formationFollow:
		return front.X, front.Y
	case formationTriangle:
		side, back = triangleSide[:], triangleBack[:]
	}
	// fix: the formation has places for ten followers
	slot := min(place, len(side)-1)
	return x - dirStepX[facing]*back[slot] - dirStepY[facing]*side[slot],
		y + dirStepX[facing]*side[slot] - dirStepY[facing]*back[slot]
}

// chooseFormation offers the formations on a scroll, the current one highlighted.
func (g *Game) chooseFormation() {
	p := g.centredPopup(formationNames[:])
	p.choice = int(g.formation)
	if i := g.runPopup(p); i != noChoice {
		g.formation = formation(i)
		g.printf("Formation: %s\n", formationNames[i])
	}
}
