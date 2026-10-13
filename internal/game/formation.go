package game

import "image"

// Party formations, the port's: how the party lines up behind its leader. Ctrl+F opens a
// scroll to pick one. The choice lasts for the session; the save keeps the original's
// layout, which has no room for it.

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

// the formation scroll on the canvas, in character cells: the options' clip, free outside
// conversations
const (
	formationX0, formationY0 = optionsX0, optionsY0
	formationColumns         = 12
	formationTitleRows       = 1
)

var formationPanel = cellRect(formationX0, formationY0, formationColumns, formationTitleRows+int(formationCount))

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

// chooseFormation shows the formation scroll until a formation is picked (Enter, its
// number) or the scroll is put away (Esc).
func (g *Game) chooseFormation() {
	choice := g.formation
	g.formationShown = true
	defer func() { g.formationShown = false }()
	for {
		g.drawFormations(choice)
		ch := 0
		for ch == 0 {
			ch = g.prompt()
		}
		switch {
		case ch == keyUp:
			choice = (choice + formationCount - 1) % formationCount
		case ch == keyDown:
			choice = (choice + 1) % formationCount
		case ch >= '1' && ch < '1'+int(formationCount):
			choice = formation(ch - '1')
			fallthrough
		case ch == keyEnter:
			g.formation = choice
			g.printf("Formation: %s\n", formationNames[choice])
			return
		case ch == keyEscape:
			return
		}
	}
}

// drawFormations writes the formation scroll on the canvas, choice highlighted.
func (g *Game) drawFormations(choice formation) {
	saved := g.current
	g.current = &g.clips[clipOptions]
	defer func() { g.current = saved }()
	g.gr.SetColor(g.colours.background)
	g.gr.FillRect(formationPanel.Min.X, formationPanel.Min.Y, formationPanel.Max.X-1, formationPanel.Max.Y-1)
	line := func(text string, row int) {
		for i := range min(len(text), formationColumns) {
			g.putCharAt(text[i], formationX0+i, formationY0+row)
		}
	}
	g.current.fg = g.colours.text
	line("Formation", 0)
	for f := range formationCount {
		g.current.fg = g.colours.text
		if f == choice {
			g.current.fg = g.colours.highlight
		}
		line(string(rune('1'+f))+" "+formationNames[f], formationTitleRows+int(f))
	}
	g.current.fg = g.colours.text
}
